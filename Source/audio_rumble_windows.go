package main

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// Captures only Galactic Racer's own audio (WASAPI process loopback, Windows
// 10 build 20348+) and turns low-frequency energy into a 0..255 level that the
// wheel plays as a third, continuous rumble layer. Re-attaches whenever the
// game restarts.

var audioRumbleLevel atomic.Uint32

const (
	defaultWheelAudioStrength = 40
	gameProcessName           = "SWGR-Win64-Shipping.exe"
	captureRate               = 48000
	captureChannels           = 2
)

var (
	ole32                           = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx              = ole32.NewProc("CoInitializeEx")
	mmdevapi                        = syscall.NewLazyDLL("mmdevapi.dll")
	procActivateAudioInterfaceAsync = mmdevapi.NewProc("ActivateAudioInterfaceAsync")
	procCreateEventW                = kernel32.NewProc("CreateEventW")
	procWaitForSingleObject         = kernel32.NewProc("WaitForSingleObject")

	iidIUnknown                                 = diGUID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIAgileObject                             = diGUID{0x94EA2B94, 0xE9CC, 0x49E0, [8]byte{0xC0, 0xFF, 0xEE, 0x64, 0xCA, 0x8F, 0x5B, 0x90}}
	iidIActivateAudioInterfaceCompletionHandler = diGUID{0x41D949AB, 0x9862, 0x444A, [8]byte{0x80, 0xF6, 0xC2, 0x61, 0x33, 0x4D, 0xA5, 0xEB}}
	iidIAudioClient                             = diGUID{0x1CB9AD4C, 0xDBFA, 0x4C32, [8]byte{0xB1, 0x78, 0xC2, 0xF5, 0x68, 0xA7, 0x03, 0xB2}}
	iidIAudioCaptureClient                      = diGUID{0xC8ADBD64, 0xE71E, 0x48A0, [8]byte{0xA4, 0xDE, 0x18, 0x5C, 0x39, 0x5C, 0xD3, 0x17}}
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

// AUDIOCLIENT_ACTIVATION_PARAMS for PROCESS_LOOPBACK, including the target's child processes.
type audioActivationParams struct {
	ActivationType   uint32 // AUDIOCLIENT_ACTIVATION_TYPE_PROCESS_LOOPBACK = 1
	TargetProcessID  uint32
	ProcessLoopbackM uint32 // PROCESS_LOOPBACK_MODE_INCLUDE_TARGET_PROCESS_TREE = 0
}

type propVariantBlob struct {
	Vt        uint16 // VT_BLOB = 65
	_         [3]uint16
	CbSize    uint32
	PBlobData *audioActivationParams
}

// ActivateAudioInterfaceAsync reports completion through a COM callback object.
// One static object is enough: activations run one at a time on the capture
// goroutine. Its only job is to signal activationDone.
var (
	activationDone    = make(chan struct{}, 1)
	activationVtbl    [4]uintptr
	activationHandler struct{ vtbl *[4]uintptr }
)

func init() {
	activationVtbl = [4]uintptr{
		syscall.NewCallback(func(this, riid, ppv uintptr) uintptr { // QueryInterface
			g := *(*diGUID)(unsafe.Pointer(riid))
			if g == iidIUnknown || g == iidIAgileObject || g == iidIActivateAudioInterfaceCompletionHandler {
				*(*uintptr)(unsafe.Pointer(ppv)) = this
				return 0
			}
			*(*uintptr)(unsafe.Pointer(ppv)) = 0
			return 0x80004002 // E_NOINTERFACE
		}),
		syscall.NewCallback(func(this uintptr) uintptr { return 1 }), // AddRef: static object
		syscall.NewCallback(func(this uintptr) uintptr { return 1 }), // Release
		syscall.NewCallback(func(this, op uintptr) uintptr { // ActivateCompleted
			select {
			case activationDone <- struct{}{}:
			default:
			}
			return 0
		}),
	}
	activationHandler.vtbl = &activationVtbl
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

func findProcessID(name string) uint32 {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer syscall.CloseHandle(snap)
	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for err = syscall.Process32First(snap, &pe); err == nil; err = syscall.Process32Next(snap, &pe) {
		if strings.EqualFold(syscall.UTF16ToString(pe.ExeFile[:]), name) {
			return pe.ProcessID
		}
	}
	return 0
}

// runAudioRumble waits for the game, captures it, and re-attaches when it
// restarts, until stop closes. reportErr is called when a new error appears.
func runAudioRumble(stop <-chan struct{}, reportErr func(error)) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer audioRumbleLevel.Store(0)
	procCoInitializeEx.Call(0, 0) // COINIT_MULTITHREADED
	lastErr := ""
	for {
		if pid := findProcessID(gameProcessName); pid != 0 {
			if err := captureProcess(pid, stop); err != nil && err.Error() != lastErr {
				lastErr = err.Error()
				reportErr(err)
			}
		}
		audioRumbleLevel.Store(0)
		select {
		case <-stop:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// captureProcess streams the game's audio until stop closes or the game exits.
func captureProcess(pid uint32, stop <-chan struct{}) error {
	if err := procActivateAudioInterfaceAsync.Find(); err != nil {
		return fmt.Errorf("game audio capture needs Windows 10 build 20348 or newer")
	}
	params := &audioActivationParams{ActivationType: 1, TargetProcessID: pid}
	pv := &propVariantBlob{Vt: 65, CbSize: uint32(unsafe.Sizeof(*params)), PBlobData: params}
	path, _ := syscall.UTF16PtrFromString(`VAD\Process_Loopback`)
	select { // drop a completion left over from a timed-out attempt
	case <-activationDone:
	default:
	}
	var op uintptr
	hr, _, _ := procActivateAudioInterfaceAsync.Call(uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(&iidIAudioClient)),
		uintptr(unsafe.Pointer(pv)), uintptr(unsafe.Pointer(&activationHandler)), uintptr(unsafe.Pointer(&op)))
	runtime.KeepAlive(pv)
	if uint32(hr) != 0 || op == 0 {
		return fmt.Errorf("game audio capture refused (0x%08X)", uint32(hr))
	}
	defer comCall(op, 2)
	select {
	case <-activationDone:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("game audio capture timed out")
	}
	var activateHR, client, capture uintptr
	comCall(op, 3, uintptr(unsafe.Pointer(&activateHR)), uintptr(unsafe.Pointer(&client))) // GetActivateResult
	if uint32(activateHR) != 0 || client == 0 {
		return fmt.Errorf("game audio capture refused (0x%08X)", uint32(activateHR))
	}
	defer func() {
		if capture != 0 {
			comCall(capture, 2)
		}
		comCall(client, 2)
	}()

	// Process loopback has no mix format to query; ask for float32 stereo and let Windows convert.
	wf := &waveFormatEx{WFormatTag: 3, NChannels: captureChannels, NSamplesPerSec: captureRate, WBitsPerSample: 32,
		NBlockAlign: captureChannels * 4, NAvgBytesPerSec: captureRate * captureChannels * 4}
	// Initialize(SHARED, LOOPBACK | EVENTCALLBACK | AUTOCONVERTPCM | SRC_DEFAULT_QUALITY, 100 ms, 0, format, nil)
	hr = comCall(client, 3, 0, 0x00020000|0x00040000|0x80000000|0x08000000, 1000000, 0, uintptr(unsafe.Pointer(wf)), 0)
	runtime.KeepAlive(wf)
	if uint32(hr) != 0 {
		return fmt.Errorf("game audio capture init failed (0x%08X)", uint32(hr))
	}
	event, _, _ := procCreateEventW.Call(0, 0, 0, 0)
	if event == 0 {
		return fmt.Errorf("game audio capture: no event handle")
	}
	defer syscall.CloseHandle(syscall.Handle(event))
	comCall(client, 13, event)                                                                                                           // SetEventHandle
	if hr := uint32(comCall(client, 14, uintptr(unsafe.Pointer(&iidIAudioCaptureClient)), uintptr(unsafe.Pointer(&capture)))); hr != 0 { // GetService
		return fmt.Errorf("game audio capture client unavailable (0x%08X)", hr)
	}
	if hr := uint32(comCall(client, 10)); hr != 0 { // Start
		return fmt.Errorf("game audio capture start failed (0x%08X)", hr)
	}
	defer comCall(client, 11) // Stop

	env := newAudioEnvelope(captureRate)
	lastCheck := time.Now()
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		if time.Since(lastCheck) > 2*time.Second {
			if findProcessID(gameProcessName) != pid {
				return nil // game exited or restarted: re-attach
			}
			lastCheck = time.Now()
		}
		procWaitForSingleObject.Call(event, 20)
		got := false
		for {
			var data *float32 // WASAPI-owned memory, valid until ReleaseBuffer
			var frames, flags uint32
			if hr := uint32(comCall(capture, 3, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&frames)), uintptr(unsafe.Pointer(&flags)), 0, 0)); hr != 0 || frames == 0 {
				break // AUDCLNT_S_BUFFER_EMPTY or error: wait for the next event
			}
			if flags&0x2 == 0 && data != nil { // not AUDCLNT_BUFFERFLAGS_SILENT
				env.feed(unsafe.Slice(data, int(frames)*captureChannels), captureChannels)
			} else {
				env.push(0)
			}
			comCall(capture, 4, uintptr(frames)) // ReleaseBuffer
			got = true
		}
		if !got {
			env.push(0)
		}
		audioRumbleLevel.Store(env.byteLevel())
	}
}
