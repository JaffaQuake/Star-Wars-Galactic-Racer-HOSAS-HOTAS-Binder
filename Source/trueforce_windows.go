package main

// Trueforce streaming for Logitech direct-drive wheels (RS50, G PRO).
//
// Protocol and the 68-packet init sequence come from libtrueforce in
// mescon/logitech-rs50-linux-driver (LGPL-2.1-or-later), see
// docs/TRUEFORCE_PROTOCOL.md there. Interface 2 (usage page 0xFFFD) takes
// 64-byte type-0x01 packets at ~1 kHz: bytes 6-9 are the motor torque
// target ("cur") and bytes 12-63 a 13-slot rolling 4 kHz haptic window that
// plays on top of it.
//
// While a stream runs the wheel steers by cur and ignores DirectInput force,
// so in this mode cur carries the centering spring and the game's rumble
// motors, and only the game audio goes to the Trueforce window: texture
// without yanking the rim. G HUB must be closed (it would fight for the
// stream; the endpoint has room for exactly one writer).

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	tfWindowSlots  = 13
	tfSampleRate   = 4000 // window samples per second (4 per 1 ms packet)
	tfCatchupMax   = 12   // most new samples one packet may declare
	tfMaxCur       = 0.8  // hard cap on the torque target, fraction of full scale
	tfMaxTexture   = 0.35 // above ~0.5 FS texture turns into steering pull (protocol notes)
	tfAudioGain    = 2.0  // ponytail: fixed audio->texture gain; wheel_audio_strength is the knob
	tfDamping      = 0.03 // force per (full axis range / s); keeps the 1 kHz spring from ringing
	tfMaxDamp      = 0.08 // log showed velocity glitches driving damping to the 0.8 cap
	tfStalePos     = 100 * time.Millisecond
	tfLockGuardPos = 0.97 // |pos| treated as "at lock" by the runaway guard
	tfLockGuardFor = 400 * time.Millisecond
	genericWrite   = 0x40000000
	tfRingMax      = 240 // 60 ms of 4 kHz audio; oldest dropped beyond this
	tfRingStart    = 60  // 15 ms jitter buffer before audio starts playing
)

// Init sequence (libtrueforce tf_init_data.h), trailing zeros trimmed; sent
// twice with the sequence byte (offset 5) restarting at 1 for each pass.
var tfInitPackets = []string{
	"0100000005", "01000000050030", "01000000050001", "01000000050002", "01000000050003",
	"01000000050004", "01000000050005", "01000000050006", "01000000050007", "01000000050008",
	"01000000050009", "0100000005000a", "0100000005000b", "0100000005000c", "0100000005000d",
	"0100000005000e", "0100000005000f", "01000000050010", "01000000050011", "01000000050012",
	"01000000050013", "01000000050014", "01000000050015", "01000000050016", "01000000050017",
	"01000000050018", "01000000050019", "0100000005001a", "0100000005001b", "0100000005001c",
	"0100000005001d", "0100000005002b", "0100000005002c", "0100000005002d", "0100000005002e",
	"0100000005002f", "01000000050031", "01000000050032", "01000000050033", "01000000050034",
	"01000000050035", "01000000050036", "01000000050037", "01000000050038", "01000000050039",
	"0100000005003a", "0100000005003b", "0100000005003c",
	"01000000010000800080", "010000000e0000c02845", "01000000010000800080", "0100000007",
	"01000000010000800080", "0100000006000101", "01000000010000800080", "0100000006000102",
	"01000000010000800080", "0100000006000103", "01000000010000800080",
	"01000000090002020000803f0000af43", "01000000010000800080", "010000000600010401",
	"01000000010000800080", "010000000600010501", "01000000010000800080", "010000000600010601",
	"0100000004", "0100000003",
}

// tfU16 converts a signed fraction of full scale to the wire's offset binary
// (0x0000 = full left, 0x8000 = none, 0xFFFF = full right).
func tfU16(f float64) uint16 {
	f = math.Max(-1, math.Min(1, f))
	return uint16(int32(math.Round(f*32767)) + 0x8000)
}

// buildTFPacket lays out one type-0x01 packet. n = new samples in window
// (0 = idle/keepalive: byte 11 and the tail must then be zero too).
func buildTFPacket(pkt *[64]byte, seq byte, cur uint16, window *[tfWindowSlots]uint16, n int) {
	*pkt = [64]byte{0: 0x01, 4: 0x01, 5: seq}
	binary.LittleEndian.PutUint16(pkt[6:], cur)
	binary.LittleEndian.PutUint16(pkt[8:], cur)
	if n <= 0 {
		return
	}
	pkt[10], pkt[11] = byte(n), 0x0d // demux pair: samples present
	for i, v := range window {
		binary.LittleEndian.PutUint16(pkt[12+i*4:], v) // L
		binary.LittleEndian.PutUint16(pkt[14+i*4:], v) // R (mono duplicated)
	}
}

// tfSpring is the centering force for a wheel at pos (-1..1 of its full DI
// range, negative = left) moving at vel (range/s): it pushes back toward
// centre like the DirectInput spring it replaces (coefficient = centerPct).
// The damping term is capped: speed comes from poll-rate position samples,
// and a timing glitch there must not turn into a kick at the rim.
func tfSpring(pos, vel float64, centerPct int) float64 {
	return -float64(centerPct)/100*pos - math.Max(-tfMaxDamp, math.Min(tfMaxDamp, tfDamping*vel))
}

// tfRunaway: at lock with our own force pushing further into it. Holding
// lock against the spring is just driving.
func tfRunaway(pos, cur float64) bool {
	return math.Abs(pos) > tfLockGuardPos && pos*cur > 0
}

// tfDecimator turns the 48 kHz capture into 4 kHz texture: band-pass about
// 20-1500 Hz (two one-pole low-passes as the anti-alias filter), keep every
// 12th sample.
type tfDecimator struct {
	hpA, lpB    float64
	prevIn, hp  float64
	lp1, lp2    float64
	phase, step int
}

func newTFDecimator(rate int) *tfDecimator {
	dt := 1 / float64(rate)
	rc := func(hz float64) float64 { return 1 / (2 * math.Pi * hz) }
	return &tfDecimator{hpA: rc(20) / (rc(20) + dt), lpB: dt / (rc(1500) + dt), step: rate / tfSampleRate}
}

func (d *tfDecimator) feed(samples []float32, channels int, out func(float64)) {
	for i := 0; i+channels <= len(samples); i += channels {
		var x float64
		for c := 0; c < channels; c++ {
			x += float64(samples[i+c])
		}
		x /= float64(channels)
		d.hp = d.hpA * (d.hp + x - d.prevIn)
		d.prevIn = x
		d.lp1 += d.lpB * (d.hp - d.lp1)
		d.lp2 += d.lpB * (d.lp1 - d.lp2)
		if d.phase++; d.phase >= d.step {
			d.phase = 0
			out(d.lp2)
		}
	}
}

// tfRing hands decimated audio from the capture goroutine to the streamer.
type tfRing struct {
	mu        sync.Mutex
	buf       []float64
	buffering bool
}

func (r *tfRing) push(v float64) {
	r.mu.Lock()
	r.buf = append(r.buf, v)
	if over := len(r.buf) - tfRingMax; over > 0 {
		r.buf = append(r.buf[:0], r.buf[over:]...)
	}
	r.mu.Unlock()
}

// pop returns up to n samples, holding back until the jitter buffer refills after a starve.
func (r *tfRing) pop(n int) []float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.buffering && len(r.buf) < tfRingStart {
		return nil
	}
	r.buffering = false
	n = min(n, len(r.buf))
	out := append([]float64(nil), r.buf[:n]...)
	r.buf = append(r.buf[:0], r.buf[n:]...)
	if len(r.buf) == 0 {
		r.buffering = true
	}
	return out
}

func (r *tfRing) reset() {
	r.mu.Lock()
	r.buf, r.buffering = r.buf[:0], true
	r.mu.Unlock()
}

var (
	tfAudio     = &tfRing{buffering: true}
	tfAudioWant atomic.Bool // capture only feeds the ring while a stream runs

	tfWheelPos   atomic.Int32 // raw DI steering axis, -32768..32767
	tfWheelPosAt atomic.Int64 // UnixNano of the last tfWheelPos update
	tfCenterPct  atomic.Int32
	tfRumblePct  atomic.Int32
	tfAudioPct   atomic.Int32

	procWriteFile       = kernel32.NewProc("WriteFile")
	procTimeBeginPeriod = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod   = winmm.NewProc("timeEndPeriod")
)

// findTrueforceDevice returns the HID path of a Logitech interface on usage
// page 0xFFFD with 64-byte output reports, or "".
func findTrueforceDevice() string {
	var count uint32
	itemSize := uint32(unsafe.Sizeof(rawInputDeviceList{}))
	if r, _, _ := procGetRawInputDeviceList.Call(0, uintptr(unsafe.Pointer(&count)), uintptr(itemSize)); uint32(r) == 0xFFFFFFFF || count == 0 {
		return ""
	}
	list := make([]rawInputDeviceList, count)
	if r, _, _ := procGetRawInputDeviceList.Call(uintptr(unsafe.Pointer(&list[0])), uintptr(unsafe.Pointer(&count)), uintptr(itemSize)); uint32(r) == 0xFFFFFFFF {
		return ""
	}
	for _, ri := range list[:count] {
		if ri.DwType != RIM_TYPEHID {
			continue
		}
		path := rawInputDevicePath(ri.HDevice)
		if !strings.Contains(strings.ToLower(path), "vid_046d") {
			continue
		}
		// Zero access rights: enough to read caps without touching the device.
		h, _, _ := procCreateFileW.Call(uintptr(unsafe.Pointer(utf16ptr(path))), 0, FILE_SHARE_READ|FILE_SHARE_WRITE, 0, OPEN_EXISTING, 0, 0)
		if h == 0 || h == ^uintptr(0) {
			continue
		}
		var prep uintptr
		var caps hidpCaps
		if ok, _, _ := procHidDGetPreparsedData.Call(h, uintptr(unsafe.Pointer(&prep))); ok != 0 && prep != 0 {
			procHidPGetCaps.Call(prep, uintptr(unsafe.Pointer(&caps)))
			procHidDFreePreparsedData.Call(prep)
		}
		procCloseHandle.Call(h)
		if caps.UsagePage == 0xFFFD && caps.OutputReportByteLength == 64 {
			return path
		}
	}
	return ""
}

// tfStreamer owns the wheel's Trueforce endpoint while Set(true) holds.
type tfStreamer struct {
	mu        sync.Mutex
	stop      chan struct{}
	done      chan struct{}
	failed    bool // stay off after an error until Set(false) clears it
	reportErr func(error)
}

var tfStream = &tfStreamer{}

// Set starts or stops streaming; cheap to call every poll.
func (s *tfStreamer) Set(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done != nil {
		select {
		case <-s.done: // stream ended on its own (error or guard)
			s.stop, s.done, s.failed = nil, nil, true
		default:
		}
	}
	if !on {
		s.failed = false
		if s.stop != nil {
			close(s.stop)
			<-s.done
			s.stop, s.done = nil, nil
		}
		return
	}
	if s.stop != nil || s.failed {
		return
	}
	s.stop, s.done = make(chan struct{}), make(chan struct{})
	go s.run(s.stop, s.done)
}

func (s *tfStreamer) run(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := s.stream(stop); err != nil && s.reportErr != nil {
		s.reportErr(err)
	}
}

// openTFLog appends to trueforce.log next to the config, for diagnosing
// sessions on real hardware (one stats line per second).
func openTFLog() (*log.Logger, func()) {
	f, err := os.OpenFile(filepath.Join(filepath.Dir(configPath()), "trueforce.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return log.New(io.Discard, "", 0), func() {}
	}
	return log.New(f, "", log.Ldate|log.Lmicroseconds), func() { f.Close() }
}

func (s *tfStreamer) stream(stop <-chan struct{}) (err error) {
	lg, closeLog := openTFLog()
	defer closeLog()
	lg.Printf("---- session start")
	defer func() { lg.Printf("session end: %v", err) }()
	path := findTrueforceDevice()
	if path == "" {
		return fmt.Errorf("Trueforce interface not found (wheel in PC mode? open G HUB once, then close it)")
	}
	h, _, _ := procCreateFileW.Call(uintptr(unsafe.Pointer(utf16ptr(path))), GENERIC_READ|genericWrite, FILE_SHARE_READ|FILE_SHARE_WRITE, 0, OPEN_EXISTING, 0, 0)
	if h == 0 || h == ^uintptr(0) {
		return fmt.Errorf("could not open the Trueforce interface; close G HUB and SimHub")
	}
	defer procCloseHandle.Call(h)
	procTimeBeginPeriod.Call(1)
	defer procTimeEndPeriod.Call(1)

	// Wheel replies on a second handle (reads and writes on one synchronous
	// handle would serialize). A healthy session answers at ~1 kHz; replies
	// stopping or changing status shows the wheel ended it, not us.
	var inReports atomic.Int64
	if rh, _, _ := procCreateFileW.Call(uintptr(unsafe.Pointer(utf16ptr(path))), GENERIC_READ, FILE_SHARE_READ|FILE_SHARE_WRITE, 0, OPEN_EXISTING, 0, 0); rh != 0 && rh != ^uintptr(0) {
		defer func() { procCancelIoEx.Call(rh, 0); procCloseHandle.Call(rh) }()
		go func() {
			buf := make([]byte, 1024)
			var last [2]byte
			for {
				var n uint32
				if r, _, e := procReadFile.Call(rh, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n)), 0); r == 0 || n == 0 {
					lg.Printf("reply reader stopped: %v", e)
					return
				}
				inReports.Add(int64(n+63) / 64)
				if n >= 9 && (buf[4] != last[0] || buf[8] != last[1]) {
					last = [2]byte{buf[4], buf[8]}
					lg.Printf("wheel reply type %02x status %02x: % x", buf[4], buf[8], buf[:min(int(n), 32)])
				}
			}
		}()
	} else {
		lg.Printf("could not open reply reader")
	}

	var pkt [64]byte
	var seq byte
	var writeMax time.Duration
	write := func() error {
		var n uint32
		t0 := time.Now()
		// ponytail: synchronous write; a wheel that stops reading would block Stop. Overlapped I/O if that ever happens.
		if ok, _, e := procWriteFile.Call(h, uintptr(unsafe.Pointer(&pkt[0])), 64, uintptr(unsafe.Pointer(&n)), 0); ok == 0 || n != 64 {
			return fmt.Errorf("Trueforce write failed: %v", e)
		}
		writeMax = max(writeMax, time.Since(t0))
		return nil
	}
	for pass := 0; pass < 2; pass++ {
		seq = 1
		for _, hx := range tfInitPackets {
			b, _ := hex.DecodeString(hx)
			pkt = [64]byte{}
			copy(pkt[:], b)
			pkt[5] = seq
			seq++
			if err := write(); err != nil {
				return err
			}
			time.Sleep(2 * time.Millisecond) // G HUB paces init at ~2 ms (it waits for each reply)
		}
	}

	lg.Printf("init sent (%d packets)", 2*len(tfInitPackets))
	tfAudio.reset()
	tfAudioWant.Store(true)
	defer tfAudioWant.Store(false)
	defer func() { // clean session end: neutral, then the captured stop/arm pair, then silence
		var w [tfWindowSlots]uint16
		buildTFPacket(&pkt, seq, 0x8000, &w, 0)
		seq++
		_ = write()
		for _, typ := range []byte{0x04, 0x03} {
			pkt = [64]byte{0: 0x01, 4: typ, 5: seq}
			seq++
			_ = write()
			time.Sleep(2 * time.Millisecond)
		}
	}()

	var window [tfWindowSlots]uint16
	for i := range window {
		window[i] = 0x8000
	}
	start := time.Now()
	sent := 0  // window samples declared so far
	ticks := 0 // packets sent, for cur's 1 kHz rumble phase
	var lastPos float64
	var lastPosAt int64
	var vel float64
	var atLockSince time.Time
	var statPkts, statSamples int
	var statCur, statTex float64
	var statIn int64
	statAt := start
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		now := time.Now()

		// Spring input: stale position (poll stalled) fades the spring out instead of holding it.
		pos := float64(tfWheelPos.Load()) / 32768
		at := tfWheelPosAt.Load()
		fresh := now.Sub(time.Unix(0, at)) < tfStalePos
		if at != lastPosAt && lastPosAt != 0 {
			dt := float64(at-lastPosAt) / 1e9
			vel += 0.2 * ((pos-lastPos)/dt - vel) // smoothed: positions arrive at the poll rate
		}
		lastPos, lastPosAt = pos, at
		var cur float64
		if fresh {
			cur = tfSpring(pos, vel, int(tfCenterPct.Load()))
		}
		// Game rumble stays in cur, at the same frequencies as the DirectInput effects.
		t := float64(ticks) / 1000
		rp := float64(tfRumblePct.Load()) / 100
		cur += float64(rumbleLarge.Load()) / 255 * rp * math.Sin(2*math.Pi*1e6/float64(wheelSinePeriodsUs[0])*t)
		cur += float64(rumbleSmall.Load()) / 255 * rp * math.Sin(2*math.Pi*1e6/float64(wheelSinePeriodsUs[1])*t)
		cur = math.Max(-tfMaxCur, math.Min(tfMaxCur, cur))
		// Runaway guard: our force holding the wheel into lock for a while means
		// the direction is wrong somewhere, stop. (Rumble alone alternates sign.)
		if fresh && tfRunaway(pos, cur) {
			if atLockSince.IsZero() {
				atLockSince = now
			} else if now.Sub(atLockSince) > tfLockGuardFor {
				return fmt.Errorf("Trueforce stopped: force pushing the wheel into full lock (safety guard). Toggle Trueforce to retry")
			}
		} else {
			atLockSince = time.Time{}
		}

		// Audio texture: as many samples as real time says are due.
		due := int(now.Sub(start).Seconds()*tfSampleRate) - sent
		if due > tfCatchupMax {
			sent += due - tfCatchupMax // fell behind: skip ahead rather than burst
			due = tfCatchupMax
		}
		n := 0
		if due > 0 {
			gain := float64(tfAudioPct.Load()) / 100 * tfAudioGain * audioMenuGain(now)
			samples := tfAudio.pop(due)
			silent := true
			for _, v := range samples {
				x := math.Max(-tfMaxTexture, math.Min(tfMaxTexture, v*gain))
				statTex = max(statTex, math.Abs(x))
				copy(window[:], window[1:])
				window[tfWindowSlots-1] = tfU16(x)
				silent = silent && window[tfWindowSlots-1] == 0x8000
			}
			sent += due
			if !silent {
				n = len(samples)
			}
		}
		buildTFPacket(&pkt, seq, tfU16(cur), &window, n)
		seq++
		ticks++
		if err := write(); err != nil {
			return err
		}
		statPkts++
		statSamples += n
		statCur = max(statCur, math.Abs(cur))
		if now.Sub(statAt) >= time.Second {
			in := inReports.Load()
			lg.Printf("pkts %d samples %d replies %d maxwrite %v peak cur %.2f tex %.2f rumble %d/%d pos %.2f fresh %v",
				statPkts, statSamples, in-statIn, writeMax, statCur, statTex, rumbleLarge.Load(), rumbleSmall.Load(), pos, fresh)
			statPkts, statSamples, statCur, statTex, statIn, writeMax, statAt = 0, 0, 0, 0, in, 0, now
		}
		// Pace to ~1 kHz; the due-sample count above absorbs timer jitter.
		if d := time.Until(start.Add(time.Duration(ticks) * time.Millisecond)); d > 0 {
			time.Sleep(d)
		}
	}
}
