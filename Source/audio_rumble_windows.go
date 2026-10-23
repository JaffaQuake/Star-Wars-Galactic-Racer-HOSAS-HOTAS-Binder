package main

import (
	"fmt"
	"math"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// Loopback-captures the default speakers and turns low-frequency energy into a
// 0..255 level that the wheel plays as a third, continuous rumble layer.
// ponytail: captures the whole default output (Discord/music included) and
// ignores device switches; per-process loopback if that turns out to matter.

var audioRumbleLevel atomic.Uint32

const defaultWheelAudioStrength = 40

var (
	ole32                = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree    = ole32.NewProc("CoTaskMemFree")

	clsidMMDeviceEnumerator = diGUID{0xBCDE0395, 0xE52F, 0x467C, [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidIMMDeviceEnumerator  = diGUID{0xA95664D2, 0x9614, 0x4F35, [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	iidIAudioClient         = diGUID{0x1CB9AD4C, 0xDBFA, 0x4C32, [8]byte{0xB1, 0x78, 0xC2, 0xF5, 0x68, 0xA7, 0x03, 0xB2}}
	iidIAudioCaptureClient  = diGUID{0xC8ADBD64, 0xE71E, 0x48A0, [8]byte{0xA4, 0xDE, 0x18, 0x5C, 0x39, 0x5C, 0xD3, 0x17}}
)

type waveFormatEx struct {
	WFormatTag      uint16
	NChannels       uint16
	NSamplesPerSec  uint32
	NAvgBytesPerSec uint32
	NBlockAlign     uint16
	WBitsPerSample  uint16
	CbSize          uint16
}

// audioEnvelope band-passes the mono mix to the rumble range and tracks its
// loudness in 10 ms windows: instant attack, smooth release.
type audioEnvelope struct {
	hpA, lpB, gain float64
	prevIn, hp, lp float64
	sumSq          float64
	n, window      int
	level          float64
}

func newAudioEnvelope(rate int) *audioEnvelope {
	dt := 1 / float64(rate)
	rc := func(hz float64) float64 { return 1 / (2 * math.Pi * hz) }
	return &audioEnvelope{
		hpA:    rc(20) / (rc(20) + dt),
		lpB:    dt / (rc(150) + dt),
		gain:   4, // ponytail: fixed loudness->level gain; wheel_audio_strength is the user knob
		window: rate / 100,
	}
}

func (e *audioEnvelope) feed(samples []float32, channels int) {
	for i := 0; i+channels <= len(samples); i += channels {
		var x float64
		for c := 0; c < channels; c++ {
			x += float64(samples[i+c])
		}
		x /= float64(channels)
		e.hp = e.hpA * (e.hp + x - e.prevIn)
		e.prevIn = x
		e.lp += e.lpB * (e.hp - e.lp)
		e.sumSq += e.lp * e.lp
		if e.n++; e.n >= e.window {
			e.push(math.Sqrt(e.sumSq / float64(e.n)))
			e.sumSq, e.n = 0, 0
		}
	}
}

func (e *audioEnvelope) push(rms float64) {
	e.level = math.Max(math.Min(1, rms*e.gain), e.level*0.85)
}

func (e *audioEnvelope) byteLevel() uint32 { return uint32(e.level*255 + 0.5) }

// runAudioRumble captures until stop closes; reportErr is called once if capture can't start.
func runAudioRumble(stop <-chan struct{}, reportErr func(error)) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer audioRumbleLevel.Store(0)
	if err := captureLoopback(stop); err != nil {
		reportErr(err)
	}
}

func captureLoopback(stop <-chan struct{}) error {
	procCoInitializeEx.Call(0, 0) // COINIT_MULTITHREADED
	var enum, dev, client, capture uintptr
	defer func() {
		for _, p := range []uintptr{capture, client, dev, enum} {
			if p != 0 {
				comCall(p, 2)
			}
		}
	}()
	if hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidMMDeviceEnumerator)), 0, 0x17,
		uintptr(unsafe.Pointer(&iidIMMDeviceEnumerator)), uintptr(unsafe.Pointer(&enum))); uint32(hr) != 0 {
		return fmt.Errorf("audio device list unavailable (0x%08X)", uint32(hr))
	}
	if hr := uint32(comCall(enum, 4, 0, 0, uintptr(unsafe.Pointer(&dev)))); hr != 0 { // GetDefaultAudioEndpoint(eRender, eConsole)
		return fmt.Errorf("no default speakers (0x%08X)", hr)
	}
	if hr := uint32(comCall(dev, 3, uintptr(unsafe.Pointer(&iidIAudioClient)), 0x17, 0, uintptr(unsafe.Pointer(&client)))); hr != 0 { // Activate
		return fmt.Errorf("audio client unavailable (0x%08X)", hr)
	}
	var wf *waveFormatEx
	if hr := uint32(comCall(client, 8, uintptr(unsafe.Pointer(&wf)))); hr != 0 || wf == nil { // GetMixFormat
		return fmt.Errorf("audio mix format unavailable (0x%08X)", hr)
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(wf)))
	// Shared-mode mix formats are float32 in practice (tag 3, or EXTENSIBLE wrapping float).
	if wf.WBitsPerSample != 32 || (wf.WFormatTag != 3 && wf.WFormatTag != 0xFFFE) {
		return fmt.Errorf("unsupported speaker format (%d-bit, tag 0x%X)", wf.WBitsPerSample, wf.WFormatTag)
	}
	channels, rate := int(wf.NChannels), int(wf.NSamplesPerSec)
	// Initialize(SHARED, LOOPBACK, 100 ms buffer, 0, format, nil)
	if hr := uint32(comCall(client, 3, 0, 0x00020000, 1000000, 0, uintptr(unsafe.Pointer(wf)), 0)); hr != 0 {
		return fmt.Errorf("loopback capture refused (0x%08X)", hr)
	}
	if hr := uint32(comCall(client, 14, uintptr(unsafe.Pointer(&iidIAudioCaptureClient)), uintptr(unsafe.Pointer(&capture)))); hr != 0 { // GetService
		return fmt.Errorf("capture client unavailable (0x%08X)", hr)
	}
	if hr := uint32(comCall(client, 10)); hr != 0 { // Start
		return fmt.Errorf("capture start failed (0x%08X)", hr)
	}
	defer comCall(client, 11) // Stop

	env := newAudioEnvelope(rate)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
		}
		got := false
		for {
			var data *float32 // WASAPI-owned memory, valid until ReleaseBuffer
			var frames, flags uint32
			if hr := uint32(comCall(capture, 3, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&frames)), uintptr(unsafe.Pointer(&flags)), 0, 0)); hr != 0 || frames == 0 {
				break // AUDCLNT_S_BUFFER_EMPTY or error: try again next tick
			}
			if flags&0x2 == 0 && data != nil { // not AUDCLNT_BUFFERFLAGS_SILENT
				env.feed(unsafe.Slice(data, int(frames)*channels), channels)
			} else {
				env.push(0)
			}
			comCall(capture, 4, uintptr(frames)) // ReleaseBuffer
			got = true
		}
		if !got { // Windows sends no packets while nothing plays
			env.push(0)
		}
		audioRumbleLevel.Store(env.byteLevel())
	}
}
