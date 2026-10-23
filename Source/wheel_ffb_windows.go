package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// The game sends XInput rumble to the virtual Xbox pad; ViGEm hands it back to
// this process via the x360 notification. Wheel mode replays it on the wheel as
// two DirectInput sine effects (large/small motor), a third sine driven by
// game audio (audio_rumble_windows.go), plus a centering spring,
// because the game itself never sends the physical wheel any force.

var (
	rumbleLarge, rumbleSmall atomic.Uint32
	x360NotificationCallback = syscall.NewCallback(func(client, target, large, small, led, user uintptr) uintptr {
		rumbleLarge.Store(uint32(large & 0xFF))
		rumbleSmall.Store(uint32(small & 0xFF))
		return 0
	})
)

const (
	defaultWheelRumbleStrength    = 50
	defaultWheelCenteringStrength = 30
)

// ffbPercent maps a config value to 0..100: unset (0) uses the default, negative turns the effect off.
func ffbPercent(v, def int) int {
	switch {
	case v == 0:
		return def
	case v < 0:
		return 0
	case v > 100:
		return 100
	}
	return v
}

// wheelFFBMagnitudes returns the spring coefficient and the three sine magnitudes, all in 0..10000.
func wheelFFBMagnitudes(large, small, audio uint32, rumblePct, audioPct, centerPct int) [4]int32 {
	scale := func(v uint32, pct int) int32 { return int32(v) * int32(pct) * 10000 / (255 * 100) }
	return [4]int32{int32(centerPct) * 100, scale(large, rumblePct), scale(small, rumblePct), scale(audio, audioPct)}
}

var (
	diGUIDSpring = diGUID{Data1: 0x13541C27, Data2: 0x8E33, Data3: 0x11D0, Data4: [8]byte{0x9A, 0xD0, 0x00, 0xA0, 0xC9, 0xA0, 0x6E, 0x35}}
	diGUIDSine   = diGUID{Data1: 0x13541C23, Data2: 0x8E33, Data3: 0x11D0, Data4: [8]byte{0x9A, 0xD0, 0x00, 0xA0, 0xC9, 0xA0, 0x6E, 0x35}}
	// Large motor = low heavy thump, small motor = fine buzz, audio = engine-ish hum.
	wheelSinePeriodsUs = [3]uint32{50000, 16000, 25000}
)

type diEffect struct {
	DwSize                  uint32
	DwFlags                 uint32
	DwDuration              uint32
	DwSamplePeriod          uint32
	DwGain                  uint32
	DwTriggerButton         uint32
	DwTriggerRepeatInterval uint32
	CAxes                   uint32
	RgdwAxes                *uint32
	RglDirection            *int32
	LpEnvelope              uintptr
	CbTypeSpecificParams    uint32
	LpvTypeSpecificParams   unsafe.Pointer
	DwStartDelay            uint32
}

type diPeriodic struct {
	DwMagnitude uint32
	LOffset     int32
	DwPhase     uint32
	DwPeriod    uint32
}

type diCondition struct {
	LOffset              int32
	LPositiveCoefficient int32
	LNegativeCoefficient int32
	DwPositiveSaturation uint32
	DwNegativeSaturation uint32
	LDeadBand            int32
}

type diPropDword struct {
	Header diPropHeader
	DwData uint32
}

type wheelFFB struct {
	dev     uintptr    // diDevice.Ptr the effects live on; 0 = none
	failed  uintptr    // device that refused FFB, so we don't retry every poll
	effects [4]uintptr // spring, large sine, small sine, audio sine
	params  [4]unsafe.Pointer
	last    [4]int32
}

// UpdateWheelFFB runs on every poll. It is cheap when nothing changed and
// returns an error only the first time a device refuses force feedback.
// With trueforce set, DirectInput effects stay off (the wheel would ignore
// them) and the steering position is published for trueforce_windows.go.
func (m *directInputManager) UpdateWheelFFB(guid string, axis int, active, trueforce bool, rumblePct, audioPct, centerPct int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var dev *diDevice
	if active && axis >= 0 && axis < 8 {
		for _, d := range m.devices {
			if strings.EqualFold(d.GUIDString, guid) {
				dev = d
				break
			}
		}
	}
	if dev == nil {
		m.releaseWheelFFBLocked()
		m.ffb.failed = 0 // stopping/restarting the Xbox controller retries
		return nil
	}
	if trueforce {
		m.releaseWheelFFBLocked()
		if st, ok := m.stateLocked(dev); ok {
			tfWheelPos.Store(diAxes(st)[axis])
			tfWheelPosAt.Store(time.Now().UnixNano())
		}
		return nil
	}
	if m.ffb.dev != dev.Ptr {
		m.releaseWheelFFBLocked()
		if m.ffb.failed == dev.Ptr {
			return nil
		}
		if err := m.createWheelFFBLocked(dev.Ptr, uint32(axis*4)); err != nil {
			m.releaseWheelFFBLocked()
			m.ffb.failed = dev.Ptr
			return err
		}
	}
	want := wheelFFBMagnitudes(rumbleLarge.Load(), rumbleSmall.Load(), audioRumbleLevel.Load(), rumblePct, audioPct, centerPct)
	for i, e := range m.ffb.effects {
		if want[i] == m.ffb.last[i] {
			continue
		}
		if i == 0 {
			c := (*diCondition)(m.ffb.params[0])
			c.LPositiveCoefficient, c.LNegativeCoefficient = want[0], want[0]
		} else {
			(*diPeriodic)(m.ffb.params[i]).DwMagnitude = uint32(want[i])
		}
		eff := m.newEffectParams(i)
		// SetParameters(DIEP_TYPESPECIFICPARAMS)
		comCall(e, 6, uintptr(unsafe.Pointer(eff)), 0x00000100)
		runtime.KeepAlive(eff)
		m.ffb.last[i] = want[i]
	}
	return nil
}

func (m *directInputManager) newEffectParams(i int) *diEffect {
	size := uint32(unsafe.Sizeof(diPeriodic{}))
	if i == 0 {
		size = uint32(unsafe.Sizeof(diCondition{}))
	}
	return &diEffect{DwSize: uint32(unsafe.Sizeof(diEffect{})), CbTypeSpecificParams: size, LpvTypeSpecificParams: m.ffb.params[i]}
}

func (m *directInputManager) createWheelFFBLocked(dev uintptr, axisOfs uint32) error {
	m.ffb.dev = dev
	comCall(dev, 8) // Unacquire
	// Force feedback requires exclusive access. DISCL_EXCLUSIVE | DISCL_BACKGROUND.
	if hr := uint32(comCall(dev, 13, m.hwnd, 0x00000001|0x00000008)); hr != 0 {
		return fmt.Errorf("exclusive wheel access refused (0x%08X); close SimHub or other force feedback apps", hr)
	}
	// DIPROP_AUTOCENTER off: our spring does the centering, at a strength we control.
	ac := &diPropDword{Header: diPropHeader{DwSize: uint32(unsafe.Sizeof(diPropDword{})), DwHeaderSize: uint32(unsafe.Sizeof(diPropHeader{}))}}
	comCall(dev, 6, 9, uintptr(unsafe.Pointer(ac)))
	runtime.KeepAlive(ac)
	if hr := uint32(comCall(dev, 7)); hr != 0 && hr != 1 { // Acquire; S_FALSE = already acquired
		return fmt.Errorf("wheel acquire failed (0x%08X)", hr)
	}

	m.ffb.params[0] = unsafe.Pointer(&diCondition{DwPositiveSaturation: 10000, DwNegativeSaturation: 10000})
	m.ffb.params[1] = unsafe.Pointer(&diPeriodic{DwPeriod: wheelSinePeriodsUs[0]})
	m.ffb.params[2] = unsafe.Pointer(&diPeriodic{DwPeriod: wheelSinePeriodsUs[1]})
	m.ffb.params[3] = unsafe.Pointer(&diPeriodic{DwPeriod: wheelSinePeriodsUs[2]})
	axes, dirs := new(uint32), new(int32)
	*axes = axisOfs
	for i, g := range []*diGUID{&diGUIDSpring, &diGUIDSine, &diGUIDSine, &diGUIDSine} {
		eff := m.newEffectParams(i)
		eff.DwFlags = 0x10 | 0x02 // DIEFF_CARTESIAN | DIEFF_OBJECTOFFSETS
		eff.DwDuration = 0xFFFFFFFF
		eff.DwGain = 10000
		eff.DwTriggerButton = 0xFFFFFFFF // DIEB_NOTRIGGER
		eff.CAxes, eff.RgdwAxes, eff.RglDirection = 1, axes, dirs
		var out uintptr
		hr := uint32(comCall(dev, 18, uintptr(unsafe.Pointer(g)), uintptr(unsafe.Pointer(eff)), uintptr(unsafe.Pointer(&out)), 0)) // CreateEffect
		runtime.KeepAlive(eff)
		if hr != 0 || out == 0 {
			return fmt.Errorf("wheel rejected force effect %d (0x%08X)", i, hr)
		}
		m.ffb.effects[i] = out
		comCall(out, 7, 1, 0) // Start
	}
	m.ffb.last = [4]int32{}
	return nil
}

func (m *directInputManager) releaseWheelFFBLocked() {
	for i, e := range m.ffb.effects {
		if e != 0 {
			comCall(e, 8)  // Stop
			comCall(e, 11) // Unload
			comCall(e, 2)  // Release
		}
		m.ffb.effects[i] = 0
		m.ffb.params[i] = nil
	}
	if m.ffb.dev != 0 {
		// Hand the wheel back in the same mode Start() opened it with.
		comCall(m.ffb.dev, 8)
		comCall(m.ffb.dev, 13, m.hwnd, 0x00000008|0x00000002)
		comCall(m.ffb.dev, 7)
		m.ffb.dev = 0
	}
}
