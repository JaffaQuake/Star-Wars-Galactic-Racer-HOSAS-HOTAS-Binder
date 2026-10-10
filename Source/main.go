//go:build windows

// Galactic Racer HOSAS / HOTAS Bridge
// Copyright (C) 2026 JaffaQuake
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// The application icon is embedded directly into the Go binary for the window/taskbar.
// The same .ico is also shipped beside the EXE so Windows shortcuts can use it.
//
//go:embed GalacticRacerHOSAS.ico
var embeddedAppIcon []byte

const (
	appTitle = "Galactic Racer HOSAS / HOTAS Bridge v1.92 Beta — by JaffaQuake and Mars"

	WS_OVERLAPPED  = 0x00000000
	WS_CAPTION     = 0x00C00000
	WS_SYSMENU     = 0x00080000
	WS_THICKFRAME  = 0x00040000
	WS_MINIMIZEBOX = 0x00020000
	WS_MAXIMIZEBOX = 0x00010000
	WS_CHILD       = 0x40000000
	WS_VISIBLE     = 0x10000000
	WS_TABSTOP     = 0x00010000
	WS_BORDER      = 0x00800000
	WS_VSCROLL     = 0x00200000
	WS_HSCROLL     = 0x00100000
	WS_POPUP       = 0x80000000

	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_NOACTIVATE  = 0x08000000

	TTS_ALWAYSTIP      = 0x01
	TTS_NOPREFIX       = 0x02
	TTF_IDISHWND       = 0x0001
	TTF_SUBCLASS       = 0x0010
	TTM_ADDTOOLW       = 0x0432
	TTM_SETMAXTIPWIDTH = 0x0418

	BS_PUSHBUTTON    = 0x00000000
	BS_AUTOCHECKBOX  = 0x00000003
	CBS_DROPDOWNLIST = 0x00000003
	ES_NUMBER        = 0x00002000
	SS_LEFT          = 0x00000000

	CW_USEDEFAULT     = ^uintptr(0x7fffffff)
	SW_HIDE           = 0
	SW_SHOWNOACTIVATE = 4
	SW_SHOW           = 5

	WM_CREATE  = 0x0001
	WM_DESTROY = 0x0002
	WM_SIZE    = 0x0005
	WM_CLOSE   = 0x0010
	WM_COMMAND = 0x0111
	WM_HSCROLL = 0x0114
	WM_VSCROLL = 0x0115
	WM_TIMER   = 0x0113
	WM_SETFONT = 0x0030
	WM_SETICON = 0x0080
	WM_APP     = 0x8000

	WM_APP_STATUS = WM_APP + 1
	WM_APP_DETECT = WM_APP + 2
	WM_APP_LEARN  = WM_APP + 3
	WM_APP_XBOX   = WM_APP + 4

	BM_GETCHECK = 0x00F0
	BM_SETCHECK = 0x00F1
	BM_SETSTATE = 0x00F3
	BST_CHECKED = 1

	SWP_NOSIZE     = 0x0001
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040
	SWP_HIDEWINDOW = 0x0080

	CB_ADDSTRING    = 0x0143
	CB_RESETCONTENT = 0x014B
	CB_GETCURSEL    = 0x0147
	CB_SETCURSEL    = 0x014E

	BN_CLICKED    = 0
	CBN_SELCHANGE = 1
	EN_CHANGE     = 0x0300

	DEFAULT_GUI_FONT = 17
	COLOR_BTNFACE    = 15
	IDC_ARROW        = 32512
	ICON_SMALL       = 0
	ICON_BIG         = 1

	SB_HORZ          = 0
	SB_VERT          = 1
	SB_LINEUP        = 0
	SB_LINEDOWN      = 1
	SB_PAGEUP        = 2
	SB_PAGEDOWN      = 3
	SB_THUMBPOSITION = 4
	SB_THUMBTRACK    = 5
	SB_TOP           = 6
	SB_BOTTOM        = 7

	JOY_RETURNX       = 0x00000001
	JOY_RETURNY       = 0x00000002
	JOY_RETURNZ       = 0x00000004
	JOY_RETURNR       = 0x00000008
	JOY_RETURNU       = 0x00000010
	JOY_RETURNV       = 0x00000020
	JOY_RETURNPOV     = 0x00000040
	JOY_RETURNBUTTONS = 0x00000080
	JOY_RETURNALL     = 0x000000FF
	JOY_POVCENTERED   = 0xFFFF
	JOYERR_NOERROR    = 0

	RIM_TYPEHID         = 2
	RIDI_DEVICENAME     = 0x20000007
	GENERIC_READ        = 0x80000000
	FILE_SHARE_READ     = 0x00000001
	FILE_SHARE_WRITE    = 0x00000002
	OPEN_EXISTING       = 3
	HIDP_INPUT          = 0
	HIDP_STATUS_SUCCESS = 0x00110000

	XUSB_DPAD_UP        = 0x0001
	XUSB_DPAD_DOWN      = 0x0002
	XUSB_DPAD_LEFT      = 0x0004
	XUSB_DPAD_RIGHT     = 0x0008
	XUSB_START          = 0x0010
	XUSB_BACK           = 0x0020
	XUSB_LEFT_THUMB     = 0x0040
	XUSB_RIGHT_THUMB    = 0x0080
	XUSB_LEFT_SHOULDER  = 0x0100
	XUSB_RIGHT_SHOULDER = 0x0200
	XUSB_A              = 0x1000
	XUSB_B              = 0x2000
	XUSB_X              = 0x4000
	XUSB_Y              = 0x8000

	VK_END          = 0x23
	VK_DELETE       = 0x2E
	VK_NEXT         = 0x22 // Page Down
	KEYEVENTF_KEYUP = 0x0002
)

const (
	idControlMode = 1001 + iota
	idInputBackend
	idHotasThrottleDevice
	idHotasThrottleAxis
	idHotasThrottleInvert
	idHotasStickDevice
	idHotasStickXAxis
	idHotasStickYAxis
	idHotasStickInvertX
	idHotasStickInvertY
	idLeftDevice
	idLeftAxis
	idLeftInvert
	idLeftThrottleMode
	idDetectLeft
	idRightDevice
	idRightAxis
	idRightInvert
	idRightThrottleMode
	idDetectRight
	idPedalDevice
	idPedalAxis
	idPedalInvert
	idPedalMode
	idPedalStrength
	idDeadzone
	idPreserve
	idStart
	idTestA
	idTestCamLeft
	idTestCamRight
	idCameraSensitivity
	idCameraDeadzone
	idPedalLeftKey
	idPedalRightKey
	idPedalKeyThreshold
	idWASDEnable
	idWASDDevice
	idWASDSideAxis
	idWASDForwardAxis
	idWASDInvertSide
	idWASDInvertForward
	idWASDThreshold
	idDetectWASD
	idPollingRate
	idSave
	idRefresh
	idHidHideSetup
	idHidHideStatus
	idMonitor
	idStatus
	idXInputReadback
	idLeftDot
	idRightDot
	idDriveSteerDot
	idLeftVectorText
	idRightVectorText
	idDriveSteerVectorText
	idRawHIDMonitor
	idOnFootMode
	idDIThumbMonitor
	idLearnBase = 1200
)

const (
	idWASDLearnBase     = 1500
	idWASDLabelBase     = 1510
	idWASDIndicatorBase = 1520
)

type point struct{ X, Y int32 }
type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}
type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type joyCaps struct {
	Mid                  uint16
	Pid                  uint16
	Pname                [32]uint16
	Xmin, Xmax           uint32
	Ymin, Ymax           uint32
	Zmin, Zmax           uint32
	NumButtons           uint32
	PeriodMin, PeriodMax uint32
	Rmin, Rmax           uint32
	Umin, Umax           uint32
	Vmin, Vmax           uint32
	Caps                 uint32
	MaxAxes              uint32
	NumAxes              uint32
	MaxButtons           uint32
	RegKey               [32]uint16
	OEMVxD               [260]uint16
}

type joyInfoEx struct {
	Size             uint32
	Flags            uint32
	X, Y, Z, R, U, V uint32
	Buttons          uint32
	ButtonNumber     uint32
	POV              uint32
	Reserved1        uint32
	Reserved2        uint32
}

type deviceInfo struct {
	ID   int
	Name string
	Caps joyCaps
}

type binding struct {
	Kind     string `json:"kind"`
	DeviceID int    `json:"device_id"`
	Button   int    `json:"button,omitempty"`
	POV      int    `json:"pov,omitempty"`
}

type keyOption struct {
	Name string
	VK   int
}

type config struct {
	InputBackend           string             `json:"input_backend"`
	ControlMode            string             `json:"control_mode"`
	HotasThrottleID        int                `json:"hotas_throttle_id"`
	HotasThrottleDIGUID    string             `json:"hotas_throttle_di_guid"`
	HotasThrottleAxis      int                `json:"hotas_throttle_axis"`
	HotasThrottleInvert    bool               `json:"hotas_throttle_invert"`
	HotasStickID           int                `json:"hotas_stick_id"`
	HotasStickDIGUID       string             `json:"hotas_stick_di_guid"`
	HotasStickXAxis        int                `json:"hotas_stick_x_axis"`
	HotasStickYAxis        int                `json:"hotas_stick_y_axis"`
	HotasStickInvertX      bool               `json:"hotas_stick_invert_x"`
	HotasStickInvertY      bool               `json:"hotas_stick_invert_y"`
	LeftID                 int                `json:"left_id"`
	LeftDIGUID             string             `json:"left_di_guid"`
	RightID                int                `json:"right_id"`
	RightDIGUID            string             `json:"right_di_guid"`
	PedalID                int                `json:"pedal_id"`
	PedalDIGUID            string             `json:"pedal_di_guid"`
	LeftAxis               int                `json:"left_axis"`
	RightAxis              int                `json:"right_axis"`
	PedalAxis              int                `json:"pedal_axis"`
	LeftInvert             bool               `json:"left_invert"`
	RightInvert            bool               `json:"right_invert"`
	LeftThrottleMode       bool               `json:"left_throttle_mode"`
	RightThrottleMode      bool               `json:"right_throttle_mode"`
	PedalInvert            bool               `json:"pedal_invert"`
	PedalMode              string             `json:"pedal_mode"`
	Deadzone               int                `json:"deadzone"`
	Bindings               map[string]binding `json:"bindings"`
	PreserveThrust         bool               `json:"preserve_thrust"`
	PedalSteerStrength     int                `json:"pedal_steer_strength"`
	CameraSensitivity      int                `json:"camera_sensitivity"`
	CameraDeadzone         int                `json:"camera_deadzone"`
	PedalLeftKey           int                `json:"pedal_left_key"`
	PedalRightKey          int                `json:"pedal_right_key"`
	PedalKeyThreshold      int                `json:"pedal_key_threshold"`
	ThumbWASDEnabled       bool               `json:"thumb_wasd_enabled"`
	ThumbWASDDeviceID      int                `json:"thumb_wasd_device_id"`
	ThumbWASDSideAxis      int                `json:"thumb_wasd_side_axis"`
	ThumbWASDForwardAxis   int                `json:"thumb_wasd_forward_axis"`
	ThumbWASDInvertSide    bool               `json:"thumb_wasd_invert_side"`
	ThumbWASDInvertForward bool               `json:"thumb_wasd_invert_forward"`
	ThumbWASDThreshold     int                `json:"thumb_wasd_threshold"`
	ThumbWASDUseRawHID     bool               `json:"thumb_wasd_use_raw_hid"`
	ThumbHIDSidePath       string             `json:"thumb_hid_side_path"`
	ThumbHIDForwardPath    string             `json:"thumb_hid_forward_path"`
	ThumbHIDSideUsage      int                `json:"thumb_hid_side_usage"`
	ThumbHIDForwardUsage   int                `json:"thumb_hid_forward_usage"`
	ThumbHIDSideCenter     int32              `json:"thumb_hid_side_center"`
	ThumbHIDForwardCenter  int32              `json:"thumb_hid_forward_center"`
	ThumbHIDSideSpan       int32              `json:"thumb_hid_side_span"`
	ThumbHIDForwardSpan    int32              `json:"thumb_hid_forward_span"`
	OnFootMode             string             `json:"on_foot_mode"`
	ThumbDISideGUID        string             `json:"thumb_di_side_guid"`
	ThumbDIForwardGUID     string             `json:"thumb_di_forward_guid"`
	ThumbDIDeviceName      string             `json:"thumb_di_device_name"`
	ThumbDISideAxis        int                `json:"thumb_di_side_axis"`
	ThumbDIForwardAxis     int                `json:"thumb_di_forward_axis"`
	ThumbDISideCenter      int32              `json:"thumb_di_side_center"`
	ThumbDIForwardCenter   int32              `json:"thumb_di_forward_center"`
	ThumbDISideSpan        int32              `json:"thumb_di_side_span"`
	ThumbDIForwardSpan     int32              `json:"thumb_di_forward_span"`
	PollingHz              int                `json:"polling_hz"`
}

type xusbReport struct {
	Buttons      uint16
	LeftTrigger  byte
	RightTrigger byte
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

type xinputGamepad struct {
	Buttons      uint16
	LeftTrigger  byte
	RightTrigger byte
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

type xinputState struct {
	PacketNumber uint32
	Gamepad      xinputGamepad
}

type monitorState struct {
	Left, Right, Pedal, Drive, Steer float64
	LeftX, LeftY, RightX, RightY     float64
	CameraX                          float64
	OutputButtons                    uint16
	OutputHighlights                 uint32
	OnFootHighlights                 uint8
	LastUpdate                       time.Time
}

type detectResult struct {
	Side             string
	DeviceID         int
	Axis             int
	Axis2            int
	Invert           bool
	Invert2          bool
	DeviceName       string
	OK               bool
	Message          string
	UseRawHID        bool
	HIDSidePath      string
	HIDForwardPath   string
	HIDSideUsage     int
	HIDForwardUsage  int
	HIDSideCenter    int32
	HIDForwardCenter int32
	HIDSideSpan      int32
	HIDForwardSpan   int32
	UseDirectInput   bool
	DISideGUID       string
	DIForwardGUID    string
	DISideCenter     int32
	DIForwardCenter  int32
	DISideSpan       int32
	DIForwardSpan    int32
}

type learnResult struct {
	Key     string
	Binding binding
	OK      bool
	Message string
}

type rawInputDeviceList struct {
	HDevice uintptr
	DwType  uint32
	_       uint32
}

type hidpCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

type hidAxisSnapshot struct {
	Name         string
	Path         string
	Values       map[uint16]int32
	Updated      time.Time
	LastActivity time.Time
}

type hidInputDevice struct {
	Path      string
	Name      string
	Handle    uintptr
	Preparsed uintptr
	ReportLen int
}

type hidManager struct {
	mu       sync.RWMutex
	states   map[string]*hidAxisSnapshot
	devices  []*hidInputDevice
	wg       sync.WaitGroup
	stopOnce sync.Once
}

var hidAxisUsages = []uint16{0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38}

func hidUsageName(u uint16) string {
	switch u {
	case 0x30:
		return "X"
	case 0x31:
		return "Y"
	case 0x32:
		return "Z"
	case 0x33:
		return "X Rotation (Rx)"
	case 0x34:
		return "Y Rotation (Ry)"
	case 0x35:
		return "Z Rotation (Rz)"
	case 0x36:
		return "Slider"
	case 0x37:
		return "Dial"
	case 0x38:
		return "Wheel"
	default:
		return fmt.Sprintf("Usage 0x%02X", u)
	}
}

func normalizeHIDAxis(current, center, positiveSpan int32) float64 {
	if positiveSpan == 0 {
		return 0
	}
	return clamp(float64(current-center)/float64(positiveSpan), -1, 1)
}

// DirectInput8 is used specifically for the on-foot thumbstick path in v1.6.
// joy.cpl sees VKB X Rotation/Y Rotation through DirectInput even when legacy
// WinMM/Raw HID paths do not expose those axes correctly.
type diGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type diDeviceInstanceW struct {
	DwSize          uint32
	GuidInstance    diGUID
	GuidProduct     diGUID
	DwDevType       uint32
	TszInstanceName [260]uint16
	TszProductName  [260]uint16
	GuidFFDriver    diGUID
	WUsagePage      uint16
	WUsage          uint16
}

type diJoyState struct {
	X, Y, Z, Rx, Ry, Rz int32
	Sliders             [2]int32
	POV                 [4]uint32
	Buttons             [32]byte
}

type diObjectDataFormat struct {
	Pguid   uintptr
	DwOfs   uint32
	DwType  uint32
	DwFlags uint32
}

type diDataFormat struct {
	DwSize     uint32
	DwObjSize  uint32
	DwFlags    uint32
	DwDataSize uint32
	DwNumObjs  uint32
	Rgodf      *diObjectDataFormat
}

type diPropHeader struct {
	DwSize       uint32
	DwHeaderSize uint32
	DwObj        uint32
	DwHow        uint32
}

type diPropRange struct {
	Header diPropHeader
	LMin   int32
	LMax   int32
}

type diDeviceDesc struct {
	GUID diGUID
	Name string
}

type diDevice struct {
	GUID         diGUID
	GUIDString   string
	Name         string
	Ptr          uintptr
	LastAxes     [8]int32
	HaveLast     bool
	LastActivity time.Time
}

type diSnapshot struct {
	GUID         string
	Name         string
	Axes         [8]int32
	LastActivity time.Time
}

type directInputManager struct {
	mu      sync.Mutex
	di      uintptr
	devices []*diDevice
	hwnd    uintptr
}

var diEnumMu sync.Mutex
var diEnumSink *[]diDeviceDesc
var diEnumCallback = syscall.NewCallback(func(pInst, _ uintptr) uintptr {
	if pInst == 0 {
		return 1
	}
	inst := (*diDeviceInstanceW)(unsafe.Pointer(pInst))
	d := diDeviceDesc{GUID: inst.GuidInstance, Name: strings.TrimSpace(syscall.UTF16ToString(inst.TszProductName[:]))}
	if d.Name == "" {
		d.Name = strings.TrimSpace(syscall.UTF16ToString(inst.TszInstanceName[:]))
	}
	diEnumMu.Lock()
	if diEnumSink != nil {
		*diEnumSink = append(*diEnumSink, d)
	}
	diEnumMu.Unlock()
	return 1 // DIENUM_CONTINUE
})

var diAxisNames = []string{"X", "Y", "Z", "X Rotation (Rx)", "Y Rotation (Ry)", "Z Rotation (Rz)", "Slider 1", "Slider 2"}

func diGUIDString(g diGUID) string {
	return fmt.Sprintf("%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X",
		g.Data1, g.Data2, g.Data3, g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

func comMethod(iface uintptr, slot int) uintptr {
	if iface == 0 {
		return 0
	}
	vtbl := *(*uintptr)(unsafe.Pointer(iface))
	if vtbl == 0 {
		return 0
	}
	return *(*uintptr)(unsafe.Pointer(vtbl + uintptr(slot)*unsafe.Sizeof(uintptr(0))))
}

func comCall(iface uintptr, slot int, args ...uintptr) uintptr {
	fn := comMethod(iface, slot)
	if fn == 0 {
		return ^uintptr(0)
	}
	callArgs := make([]uintptr, 0, len(args)+1)
	callArgs = append(callArgs, iface)
	callArgs = append(callArgs, args...)
	r1, _, _ := syscall.SyscallN(fn, callArgs...)
	return r1
}

func buildDIDataFormat() ([]diObjectDataFormat, diDataFormat) {
	const (
		didftAnyInstance = 0x00FFFF00
		didftOptional    = 0x80000000
		didftAxis        = 0x00000003
		didftButton      = 0x0000000C
		didftPOV         = 0x00000010
		didfAbsAxis      = 0x00000001
	)
	objs := make([]diObjectDataFormat, 0, 44)
	for _, ofs := range []uint32{0, 4, 8, 12, 16, 20, 24, 28} {
		objs = append(objs, diObjectDataFormat{DwOfs: ofs, DwType: didftAxis | didftAnyInstance | didftOptional})
	}
	for _, ofs := range []uint32{32, 36, 40, 44} {
		objs = append(objs, diObjectDataFormat{DwOfs: ofs, DwType: didftPOV | didftAnyInstance | didftOptional})
	}
	for i := 0; i < 32; i++ {
		objs = append(objs, diObjectDataFormat{DwOfs: uint32(48 + i), DwType: didftButton | didftAnyInstance | didftOptional})
	}
	fmtv := diDataFormat{
		DwSize: uint32(unsafe.Sizeof(diDataFormat{})), DwObjSize: uint32(unsafe.Sizeof(diObjectDataFormat{})),
		DwFlags: didfAbsAxis, DwDataSize: uint32(unsafe.Sizeof(diJoyState{})), DwNumObjs: uint32(len(objs)), Rgodf: &objs[0],
	}
	return objs, fmtv
}

func (m *directInputManager) Start(hwnd uintptr) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
	m.hwnd = hwnd
	if err := procDirectInput8Create.Find(); err != nil {
		return fmt.Errorf("dinput8.dll unavailable: %v", err)
	}
	hinst, _, _ := procGetModuleHandleW.Call(0)
	iid := diGUID{Data1: 0xBF798031, Data2: 0x483A, Data3: 0x4DA2, Data4: [8]byte{0xAA, 0x99, 0x5D, 0x64, 0xED, 0x36, 0x97, 0x00}}
	var di uintptr
	hr, _, _ := procDirectInput8Create.Call(hinst, 0x0800, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&di)), 0)
	if uint32(hr) != 0 || di == 0 {
		return fmt.Errorf("DirectInput8Create failed (0x%08X)", uint32(hr))
	}
	m.di = di

	descs := []diDeviceDesc{}
	diEnumMu.Lock()
	diEnumSink = &descs
	diEnumMu.Unlock()
	// IDirectInput8::EnumDevices slot 4; GAMECTRL=4, ATTACHEDONLY=1.
	comCall(m.di, 4, 4, diEnumCallback, 0, 1)
	diEnumMu.Lock()
	diEnumSink = nil
	diEnumMu.Unlock()

	for _, desc := range descs {
		var dev uintptr
		hr = comCall(m.di, 3, uintptr(unsafe.Pointer(&desc.GUID)), uintptr(unsafe.Pointer(&dev)), 0)
		if uint32(hr) != 0 || dev == 0 {
			continue
		}
		objs, fmtv := buildDIDataFormat()
		hrFmt := comCall(dev, 11, uintptr(unsafe.Pointer(&fmtv)))
		runtime.KeepAlive(objs) // SetDataFormat references the object-format backing array.
		if uint32(hrFmt) != 0 {
			comCall(dev, 2)
			continue
		}
		// Ask DirectInput to normalize every available analog axis to a predictable signed range.
		// DIPROP_RANGE is the special DirectInput property token 4; DIPH_BYOFFSET is 1.
		for _, ofs := range []uint32{0, 4, 8, 12, 16, 20, 24, 28} {
			pr := diPropRange{Header: diPropHeader{DwSize: uint32(unsafe.Sizeof(diPropRange{})), DwHeaderSize: uint32(unsafe.Sizeof(diPropHeader{})), DwObj: ofs, DwHow: 1}, LMin: -32768, LMax: 32767}
			_ = comCall(dev, 6, 4, uintptr(unsafe.Pointer(&pr)))
		}
		// DISCL_BACKGROUND | DISCL_NONEXCLUSIVE.
		if uint32(comCall(dev, 13, hwnd, 0x00000008|0x00000002)) != 0 {
			comCall(dev, 2)
			continue
		}
		_ = comCall(dev, 7) // Acquire. Retry also happens when polling.
		m.devices = append(m.devices, &diDevice{GUID: desc.GUID, GUIDString: diGUIDString(desc.GUID), Name: desc.Name, Ptr: dev})
	}
	return nil
}

func (m *directInputManager) Restart(hwnd uintptr) error { return m.Start(hwnd) }

func (m *directInputManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

func (m *directInputManager) stopLocked() {
	for _, d := range m.devices {
		if d.Ptr != 0 {
			comCall(d.Ptr, 8) // Unacquire
			comCall(d.Ptr, 2) // Release
		}
	}
	m.devices = nil
	if m.di != 0 {
		comCall(m.di, 2)
		m.di = 0
	}
}

func diAxes(st diJoyState) [8]int32 {
	return [8]int32{st.X, st.Y, st.Z, st.Rx, st.Ry, st.Rz, st.Sliders[0], st.Sliders[1]}
}

func (m *directInputManager) stateLocked(d *diDevice) (diJoyState, bool) {
	var st diJoyState
	if d == nil || d.Ptr == 0 {
		return st, false
	}
	_ = comCall(d.Ptr, 25) // Poll
	hr := comCall(d.Ptr, 9, uintptr(unsafe.Sizeof(st)), uintptr(unsafe.Pointer(&st)))
	if uint32(hr) != 0 {
		_ = comCall(d.Ptr, 7) // Acquire again after focus/device changes.
		_ = comCall(d.Ptr, 25)
		hr = comCall(d.Ptr, 9, uintptr(unsafe.Sizeof(st)), uintptr(unsafe.Pointer(&st)))
	}
	return st, uint32(hr) == 0
}

func (m *directInputManager) State(guid string) ([8]int32, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.devices {
		if strings.EqualFold(d.GUIDString, guid) {
			st, ok := m.stateLocked(d)
			if !ok {
				return [8]int32{}, false
			}
			axes := diAxes(st)
			m.noteActivityLocked(d, axes)
			return axes, true
		}
	}
	return [8]int32{}, false
}

func (m *directInputManager) noteActivityLocked(d *diDevice, axes [8]int32) {
	if d.HaveLast {
		changed := false
		for i := range axes {
			if axes[i] != d.LastAxes[i] {
				changed = true
				break
			}
		}
		if changed {
			d.LastActivity = time.Now()
		}
	} else {
		d.LastActivity = time.Now()
		d.HaveLast = true
	}
	d.LastAxes = axes
}

func (m *directInputManager) Snapshots() []diSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]diSnapshot, 0, len(m.devices))
	for _, d := range m.devices {
		st, ok := m.stateLocked(d)
		if !ok {
			continue
		}
		axes := diAxes(st)
		m.noteActivityLocked(d, axes)
		out = append(out, diSnapshot{GUID: d.GUIDString, Name: d.Name, Axes: axes, LastActivity: d.LastActivity})
	}
	return out
}

func (m *directInputManager) MostActive() (diSnapshot, bool) {
	snaps := m.Snapshots()
	if len(snaps) == 0 {
		return diSnapshot{}, false
	}
	best := snaps[0]
	for _, s := range snaps[1:] {
		if s.LastActivity.After(best.LastActivity) {
			best = s
		}
	}
	return best, true
}

type diChoice struct {
	GUID string
	Name string
}

func (m *directInputManager) DeviceList() []diChoice {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]diChoice, 0, len(m.devices))
	for _, d := range m.devices {
		out = append(out, diChoice{GUID: d.GUIDString, Name: d.Name})
	}
	return out
}

func (m *directInputManager) StateByGUID(guid string) ([8]int32, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.devices {
		if strings.EqualFold(d.GUIDString, guid) {
			st, ok := m.stateLocked(d)
			if !ok {
				return [8]int32{}, d.Name, false
			}
			axes := diAxes(st)
			m.noteActivityLocked(d, axes)
			return axes, d.Name, true
		}
	}
	return [8]int32{}, "", false
}

func normalizeDIAbsoluteAxis(v int32, invert bool) float64 {
	// v1.92 Beta asks DirectInput to expose analog axes as -32768..32767.
	var n float64
	if v < 0 {
		n = float64(v) / 32768.0
	} else {
		n = float64(v) / 32767.0
	}
	n = clamp(n, -1, 1)
	if invert {
		n = -n
	}
	return n
}

func normalizeDIAxis(current, center, positiveSpan int32) float64 {
	if positiveSpan == 0 {
		return 0
	}
	return clamp(float64(current-center)/float64(positiveSpan), -1, 1)
}

func (a *appState) detectDirectInputAxis(prompt, excludeGUID string, excludeAxis int) (guid, name string, axis int, center, span int32, score float64, ok bool) {
	a.postStatus(prompt)
	time.Sleep(180 * time.Millisecond)
	bases := a.di.Snapshots()
	if len(bases) == 0 {
		return "", "", 0, 0, 0, 0, false
	}
	baseMap := make(map[string][8]int32)
	nameMap := make(map[string]string)
	for _, s := range bases {
		baseMap[s.GUID] = s.Axes
		nameMap[s.GUID] = s.Name
	}
	deadline := time.Now().Add(6 * time.Second)
	bestScore := 0.0
	var bestGUID, bestName string
	bestAxis := 0
	var bestCenter, bestSpan int32
	for time.Now().Before(deadline) {
		for _, st := range a.di.Snapshots() {
			base, exists := baseMap[st.GUID]
			if !exists {
				baseMap[st.GUID] = st.Axes
				nameMap[st.GUID] = st.Name
				continue
			}
			for i, cur := range st.Axes {
				if strings.EqualFold(st.GUID, excludeGUID) && i == excludeAxis {
					continue
				}
				delta := int64(cur) - int64(base[i])
				// DI joystick axes are normally 0..65535. Use a fixed half-range score.
				sc := math.Abs(float64(delta)) / 32768.0
				if sc > bestScore {
					bestScore = sc
					bestGUID = st.GUID
					bestName = st.Name
					bestAxis = i
					bestCenter = base[i]
					bestSpan = int32(delta)
				}
			}
		}
		if bestScore >= 0.45 {
			break
		}
		time.Sleep(12 * time.Millisecond)
	}
	return bestGUID, bestName, bestAxis, bestCenter, bestSpan, bestScore, bestGUID != "" && bestScore >= 0.08 && bestSpan != 0
}

func (a *appState) refreshDirectInputMonitor() {
	h := a.controls[idDIThumbMonitor]
	if h == 0 {
		return
	}
	if vis, _, _ := procIsWindowVisible.Call(h); vis == 0 {
		return
	}
	if a.mappingActive.Load() {
		setText(h, "DirectInput axis monitor paused while game mapping is active.")
		return
	}
	st, ok := a.di.MostActive()
	if !ok {
		setText(h, "DirectInput axis monitor: no game-controller devices available.")
		return
	}
	parts := make([]string, 0, 8)
	for i, v := range st.Axes {
		parts = append(parts, fmt.Sprintf("%s %d", diAxisNames[i], v))
	}
	setText(h, fmt.Sprintf("DirectInput: %s | %s", st.Name, strings.Join(parts, "   ")))
}

type xboxBridge struct {
	mu                                                             sync.Mutex
	dll                                                            *syscall.DLL
	alloc, free, connect, disconnect                               *syscall.Proc
	targetAlloc, targetFree, targetAdd, targetRemove, targetUpdate *syscall.Proc
	client                                                         uintptr
	target                                                         uintptr
	ready                                                          bool
}

type appState struct {
	hwnd      uintptr
	font      uintptr
	cfgMu     sync.RWMutex
	cfg       config
	devicesMu sync.RWMutex
	devices   []deviceInfo

	controls         map[int]uintptr
	deviceComboIDs   map[int][]int
	deviceComboGUIDs map[int][]string
	axisNames        []string

	uiReady           bool
	visualsPaused     bool
	mappingActive     atomic.Bool
	mappingStarting   atomic.Bool
	xboxReady         atomic.Bool
	xboxSlot          atomic.Int32
	testUntilMu       sync.RWMutex
	testUntil         time.Time
	cameraTestMu      sync.RWMutex
	cameraTestUntil   time.Time
	cameraTestValue   float64
	monitorMu         sync.RWMutex
	monitor           monitorState
	keyboardMu        sync.Mutex
	keyboardDown      map[uint16]bool
	tooltipHwnd       uintptr
	tooltipTexts      []*uint16
	tooltipHelp       map[uintptr]string
	tooltipPopup      uintptr
	tooltipHover      uintptr
	baseRects         map[uintptr]rect
	scrollX           int
	scrollY           int
	contentWidth      int
	contentHeight     int
	controlHOSASGroup []uintptr
	controlHOTASGroup []uintptr
	hosasDrivingGroup []uintptr
	pedalFineGroup    []uintptr
	pedalCameraGroup  []uintptr
	pedalKeyGroup     []uintptr
	onFootThumbGroup  []uintptr
	onFootButtonGroup []uintptr

	xbox     xboxBridge
	hid      hidManager
	di       directInputManager
	stopCh   chan struct{}
	stopOnce sync.Once
	pollWG   sync.WaitGroup
	winmmMu  sync.Mutex

	pendingMu   sync.Mutex
	statusQueue []string
	detectQueue []detectResult
	learnQueue  []learnResult
}

var app *appState

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")
	hiddll   = syscall.NewLazyDLL("hid.dll")
	dinput8  = syscall.NewLazyDLL("dinput8.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procShowWindow               = user32.NewProc("ShowWindow")
	procUpdateWindow             = user32.NewProc("UpdateWindow")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procSendMessageW             = user32.NewProc("SendMessageW")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procSetWindowTextW           = user32.NewProc("SetWindowTextW")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procSetTimer                 = user32.NewProc("SetTimer")
	procKillTimer                = user32.NewProc("KillTimer")
	procKeybdEvent               = user32.NewProc("keybd_event")
	procLoadCursorW              = user32.NewProc("LoadCursorW")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetClientRect            = user32.NewProc("GetClientRect")
	procSetScrollRange           = user32.NewProc("SetScrollRange")
	procSetScrollPos             = user32.NewProc("SetScrollPos")
	procShowScrollBar            = user32.NewProc("ShowScrollBar")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject           = gdi32.NewProc("GetStockObject")
	procInitCommonControlsEx     = comctl32.NewProc("InitCommonControlsEx")

	procJoyGetNumDevs  = winmm.NewProc("joyGetNumDevs")
	procJoyGetDevCapsW = winmm.NewProc("joyGetDevCapsW")
	procJoyGetPosEx    = winmm.NewProc("joyGetPosEx")

	procGetRawInputDeviceList  = user32.NewProc("GetRawInputDeviceList")
	procGetRawInputDeviceInfoW = user32.NewProc("GetRawInputDeviceInfoW")
	procCreateFileW            = kernel32.NewProc("CreateFileW")
	procReadFile               = kernel32.NewProc("ReadFile")
	procCloseHandle            = kernel32.NewProc("CloseHandle")
	procCancelIoEx             = kernel32.NewProc("CancelIoEx")
	procHidDGetPreparsedData   = hiddll.NewProc("HidD_GetPreparsedData")
	procHidDFreePreparsedData  = hiddll.NewProc("HidD_FreePreparsedData")
	procHidDGetProductString   = hiddll.NewProc("HidD_GetProductString")
	procHidPGetCaps            = hiddll.NewProc("HidP_GetCaps")
	procHidPGetUsageValue      = hiddll.NewProc("HidP_GetUsageValue")
	procDirectInput8Create     = dinput8.NewProc("DirectInput8Create")
	procShellExecuteW          = shell32.NewProc("ShellExecuteW")
)

var wndProcCallback = syscall.NewCallback(wndProc)

var buttonOutputs = []struct {
	Key, Label string
	Mask       uint16 // Xbox button mask; zero for keyboard-only outputs.
	VK         uint16 // Windows virtual key; zero for Xbox-only outputs.
}{
	{"A", "Xbox A", XUSB_A, 0}, {"B", "Xbox B", XUSB_B, 0},
	{"X", "Xbox X", XUSB_X, 0}, {"Y", "Xbox Y", XUSB_Y, 0},
	{"LB", "Left bumper", XUSB_LEFT_SHOULDER, 0}, {"RB", "Right bumper", XUSB_RIGHT_SHOULDER, 0},
	{"BACK", "Back / View", XUSB_BACK, 0}, {"START", "Start / Menu", XUSB_START, 0},
	{"LTHUMB", "Left-stick click", XUSB_LEFT_THUMB, 0}, {"RTHUMB", "Right-stick click", XUSB_RIGHT_THUMB, 0},
	{"DUP", "D-pad Up", XUSB_DPAD_UP, 0}, {"DDOWN", "D-pad Down", XUSB_DPAD_DOWN, 0},
	{"DLEFT", "D-pad Left", XUSB_DPAD_LEFT, 0}, {"DRIGHT", "D-pad Right", XUSB_DPAD_RIGHT, 0},
	{"LOOKLEFT", "Look Left", 0, VK_DELETE},
	{"LOOKRIGHT", "Look Right", 0, VK_NEXT},
	{"LOOKBACK", "Look Back", 0, VK_END},
}

var wasdButtonDefs = []struct {
	Key, Label string
	VK         uint16
}{
	{"WASD_W", "W", uint16('W')},
	{"WASD_A", "A", uint16('A')},
	{"WASD_S", "S", uint16('S')},
	{"WASD_D", "D", uint16('D')},
}

var keyboardKeyOptions = func() []keyOption {
	out := []keyOption{
		{"Delete", 0x2E}, {"Page Down", 0x22}, {"End", 0x23},
		{"Left Arrow", 0x25}, {"Right Arrow", 0x27}, {"Up Arrow", 0x26}, {"Down Arrow", 0x28},
		{"Page Up", 0x21}, {"Home", 0x24}, {"Insert", 0x2D}, {"Space", 0x20},
		{"Tab", 0x09}, {"Enter", 0x0D}, {"Backspace", 0x08},
	}
	for ch := 'A'; ch <= 'Z'; ch++ {
		out = append(out, keyOption{Name: string(ch), VK: int(ch)})
	}
	for ch := '0'; ch <= '9'; ch++ {
		out = append(out, keyOption{Name: string(ch), VK: int(ch)})
	}
	for i := 1; i <= 12; i++ {
		out = append(out, keyOption{Name: fmt.Sprintf("F%d", i), VK: 0x6F + i})
	}
	for i := 0; i <= 9; i++ {
		out = append(out, keyOption{Name: fmt.Sprintf("Numpad %d", i), VK: 0x60 + i})
	}
	return out
}()

func utf16ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func loword(v uintptr) int { return int(v & 0xffff) }
func hiword(v uintptr) int { return int((v >> 16) & 0xffff) }
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

type initCommonControlsEx struct {
	DwSize uint32
	DwICC  uint32
}

type rect struct{ Left, Top, Right, Bottom int32 }
type toolInfo struct {
	CbSize     uint32
	UFlags     uint32
	Hwnd       uintptr
	UId        uintptr
	Rect       rect
	Hinst      uintptr
	LpszText   *uint16
	LParam     uintptr
	LpReserved uintptr
}

var pollingRateOptions = []int{100, 200, 250, 500, 1000}

func validPollingHz(hz int) bool {
	for _, v := range pollingRateOptions {
		if hz == v {
			return true
		}
	}
	return false
}
func pollingInterval(hz int) time.Duration {
	if !validPollingHz(hz) {
		hz = 500
	}
	return time.Second / time.Duration(hz)
}

func iconFromICO(data []byte, desired int) uintptr {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return 0
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	bestOff, bestSize, bestDiff := 0, 0, int(^uint(0)>>1)
	for i := 0; i < count; i++ {
		pos := 6 + i*16
		if pos+16 > len(data) {
			break
		}
		w := int(data[pos])
		if w == 0 {
			w = 256
		}
		h := int(data[pos+1])
		if h == 0 {
			h = 256
		}
		sz := int(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
		off := int(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))
		if off < 0 || sz <= 0 || off+sz > len(data) {
			continue
		}
		diff := int(math.Abs(float64(w-desired)) + math.Abs(float64(h-desired)))
		if diff < bestDiff {
			bestDiff, bestOff, bestSize = diff, off, sz
		}
	}
	if bestSize == 0 {
		return 0
	}
	img := data[bestOff : bestOff+bestSize]
	h, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&img[0])), uintptr(len(img)), 1, 0x00030000,
		uintptr(desired), uintptr(desired), 0,
	)
	return h
}

func (h *hidManager) Start() {
	h.mu.Lock()
	if h.states == nil {
		h.states = make(map[string]*hidAxisSnapshot)
	}
	h.mu.Unlock()

	var count uint32
	itemSize := uint32(unsafe.Sizeof(rawInputDeviceList{}))
	r, _, _ := procGetRawInputDeviceList.Call(0, uintptr(unsafe.Pointer(&count)), uintptr(itemSize))
	if uint32(r) == 0xFFFFFFFF || count == 0 {
		return
	}
	list := make([]rawInputDeviceList, count)
	r, _, _ = procGetRawInputDeviceList.Call(uintptr(unsafe.Pointer(&list[0])), uintptr(unsafe.Pointer(&count)), uintptr(itemSize))
	if uint32(r) == 0xFFFFFFFF {
		return
	}
	for _, ri := range list[:count] {
		if ri.DwType != RIM_TYPEHID || ri.HDevice == 0 {
			continue
		}
		path := rawInputDevicePath(ri.HDevice)
		if path == "" {
			continue
		}
		handle, _, _ := procCreateFileW.Call(
			uintptr(unsafe.Pointer(utf16ptr(path))),
			GENERIC_READ,
			FILE_SHARE_READ|FILE_SHARE_WRITE,
			0,
			OPEN_EXISTING,
			0,
			0,
		)
		if handle == 0 || handle == ^uintptr(0) {
			continue
		}
		var prep uintptr
		ok, _, _ := procHidDGetPreparsedData.Call(handle, uintptr(unsafe.Pointer(&prep)))
		if ok == 0 || prep == 0 {
			procCloseHandle.Call(handle)
			continue
		}
		var caps hidpCaps
		status, _, _ := procHidPGetCaps.Call(prep, uintptr(unsafe.Pointer(&caps)))
		if uint32(status) != HIDP_STATUS_SUCCESS || caps.UsagePage != 0x01 || (caps.Usage != 0x04 && caps.Usage != 0x05 && caps.Usage != 0x08) || caps.InputReportByteLength == 0 {
			procHidDFreePreparsedData.Call(prep)
			procCloseHandle.Call(handle)
			continue
		}
		name := hidProductName(handle)
		if name == "" {
			name = rawHIDShortName(path)
		}
		dev := &hidInputDevice{Path: path, Name: name, Handle: handle, Preparsed: prep, ReportLen: int(caps.InputReportByteLength)}
		h.mu.Lock()
		h.devices = append(h.devices, dev)
		h.states[path] = &hidAxisSnapshot{Name: name, Path: path, Values: make(map[uint16]int32)}
		h.mu.Unlock()
		h.wg.Add(1)
		go h.readLoop(dev)
	}
}

func (h *hidManager) Stop() {
	h.stopOnce.Do(func() {
		h.mu.RLock()
		devs := append([]*hidInputDevice(nil), h.devices...)
		h.mu.RUnlock()
		for _, d := range devs {
			if d.Handle != 0 && d.Handle != ^uintptr(0) {
				procCancelIoEx.Call(d.Handle, 0)
				procCloseHandle.Call(d.Handle)
			}
		}
		h.wg.Wait()
		for _, d := range devs {
			if d.Preparsed != 0 {
				procHidDFreePreparsedData.Call(d.Preparsed)
			}
		}
	})
}

func rawInputDevicePath(hDevice uintptr) string {
	var chars uint32
	procGetRawInputDeviceInfoW.Call(hDevice, RIDI_DEVICENAME, 0, uintptr(unsafe.Pointer(&chars)))
	if chars == 0 {
		return ""
	}
	buf := make([]uint16, int(chars)+2)
	r, _, _ := procGetRawInputDeviceInfoW.Call(hDevice, RIDI_DEVICENAME, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&chars)))
	if int32(r) < 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func hidProductName(handle uintptr) string {
	buf := make([]uint16, 256)
	r, _, _ := procHidDGetProductString.Call(handle, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2))
	if r == 0 {
		return ""
	}
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}

func rawHIDShortName(path string) string {
	up := strings.ToUpper(path)
	vid, pid := "", ""
	if i := strings.Index(up, "VID_"); i >= 0 && i+8 <= len(up) {
		vid = up[i+4 : i+8]
	}
	if i := strings.Index(up, "PID_"); i >= 0 && i+8 <= len(up) {
		pid = up[i+4 : i+8]
	}
	if vid != "" || pid != "" {
		return fmt.Sprintf("HID VID_%s PID_%s", vid, pid)
	}
	return "Raw HID joystick"
}

func (h *hidManager) readLoop(d *hidInputDevice) {
	defer h.wg.Done()
	buf := make([]byte, d.ReportLen)
	for {
		var n uint32
		r, _, _ := procReadFile.Call(d.Handle, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n)), 0)
		if r == 0 || n == 0 {
			return
		}
		values := make(map[uint16]int32)
		for _, usage := range hidAxisUsages {
			var v uint32
			status, _, _ := procHidPGetUsageValue.Call(
				HIDP_INPUT,
				0x01,
				0,
				uintptr(usage),
				uintptr(unsafe.Pointer(&v)),
				d.Preparsed,
				uintptr(unsafe.Pointer(&buf[0])),
				uintptr(n),
			)
			if uint32(status) == HIDP_STATUS_SUCCESS {
				values[usage] = int32(v)
			}
		}
		if len(values) == 0 {
			continue
		}
		now := time.Now()
		h.mu.Lock()
		st := h.states[d.Path]
		if st == nil {
			st = &hidAxisSnapshot{Name: d.Name, Path: d.Path, Values: make(map[uint16]int32)}
			h.states[d.Path] = st
		}
		changed := false
		for usage, v := range values {
			if old, ok := st.Values[usage]; !ok || old != v {
				changed = true
			}
			st.Values[usage] = v
		}
		st.Updated = now
		if changed {
			st.LastActivity = now
		}
		h.mu.Unlock()
	}
}

func (h *hidManager) snapshot() map[string]hidAxisSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]hidAxisSnapshot, len(h.states))
	for path, st := range h.states {
		cp := hidAxisSnapshot{Name: st.Name, Path: st.Path, Updated: st.Updated, LastActivity: st.LastActivity, Values: make(map[uint16]int32, len(st.Values))}
		for u, v := range st.Values {
			cp.Values[u] = v
		}
		out[path] = cp
	}
	return out
}

func (h *hidManager) axisValue(path string, usage uint16) (int32, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	st := h.states[path]
	if st == nil {
		return 0, false
	}
	v, ok := st.Values[usage]
	return v, ok
}

func (h *hidManager) mostActive() (hidAxisSnapshot, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var best *hidAxisSnapshot
	for _, st := range h.states {
		if len(st.Values) == 0 {
			continue
		}
		if best == nil || st.LastActivity.After(best.LastActivity) {
			best = st
		}
	}
	if best == nil {
		return hidAxisSnapshot{}, false
	}
	cp := hidAxisSnapshot{Name: best.Name, Path: best.Path, Updated: best.Updated, LastActivity: best.LastActivity, Values: make(map[uint16]int32, len(best.Values))}
	for u, v := range best.Values {
		cp.Values[u] = v
	}
	return cp, true
}

func (a *appState) detectRawHIDAxis(prompt, excludePath string, excludeUsage uint16) (path, name string, usage uint16, center, span int32, score float64, ok bool) {
	a.postStatus(prompt)
	time.Sleep(180 * time.Millisecond)
	bases := a.hid.snapshot()
	if len(bases) == 0 {
		return "", "", 0, 0, 0, 0, false
	}
	deadline := time.Now().Add(6 * time.Second)
	bestScore := 0.0
	var bestPath, bestName string
	var bestUsage uint16
	var bestCenter, bestSpan int32
	for time.Now().Before(deadline) {
		snaps := a.hid.snapshot()
		for p, st := range snaps {
			base, exists := bases[p]
			if !exists {
				bases[p] = st
				continue
			}
			for u, cur := range st.Values {
				if p == excludePath && u == excludeUsage {
					continue
				}
				b, exists := base.Values[u]
				if !exists {
					base.Values[u] = cur
					bases[p] = base
					continue
				}
				delta := int64(cur) - int64(b)
				denom := math.Max(math.Abs(float64(b)), 256.0)
				sc := math.Abs(float64(delta)) / denom
				if sc > bestScore {
					bestScore = sc
					bestPath, bestName, bestUsage = p, st.Name, u
					bestCenter, bestSpan = b, int32(delta)
				}
			}
		}
		if bestScore >= 0.45 {
			break
		}
		time.Sleep(12 * time.Millisecond)
	}
	return bestPath, bestName, bestUsage, bestCenter, bestSpan, bestScore, bestScore >= 0.08 && bestSpan != 0
}

func (a *appState) refreshRawHIDMonitor() {
	h := a.controls[idRawHIDMonitor]
	if h == 0 {
		return
	}
	if a.mappingActive.Load() {
		setText(h, "Raw HID axis monitor paused while game mapping is active.")
		return
	}
	st, ok := a.hid.mostActive()
	if !ok {
		setText(h, "Raw HID axis monitor: waiting for joystick reports...")
		return
	}
	parts := []string{}
	for _, u := range hidAxisUsages {
		if v, exists := st.Values[u]; exists {
			parts = append(parts, fmt.Sprintf("%s %d", hidUsageName(u), v))
		}
	}
	if len(parts) > 6 {
		parts = parts[:6]
	}
	setText(h, fmt.Sprintf("Raw HID: %s | %s", st.Name, strings.Join(parts, "   ")))
}

func main() {
	runtime.LockOSThread()
	icc := initCommonControlsEx{DwSize: uint32(unsafe.Sizeof(initCommonControlsEx{})), DwICC: 0x000000FF}
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	app = &appState{
		controls:         make(map[int]uintptr),
		deviceComboIDs:   make(map[int][]int),
		deviceComboGUIDs: make(map[int][]string),
		baseRects:        make(map[uintptr]rect),
		contentWidth:     1130,
		contentHeight:    1175,
		axisNames:        []string{"X", "Y", "Z", "X Rotation (R)", "Y Rotation (U)", "Z Rotation (V)"},
		tooltipHelp:      make(map[uintptr]string),
		keyboardDown:     make(map[uint16]bool),
		stopCh:           make(chan struct{}),
	}
	app.xboxSlot.Store(-1)
	app.cfg = defaultConfig()
	loadConfig(&app.cfg)
	app.devices = enumerateDevices()

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	bigIcon := iconFromICO(embeddedAppIcon, 256)
	smallIcon := iconFromICO(embeddedAppIcon, 32)
	className := utf16ptr("GalacticRacerHOSASBridgeClass")
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   wndProcCallback,
		HInstance:     hInstance,
		HIcon:         bigIcon,
		HCursor:       cursor,
		HbrBackground: uintptr(COLOR_BTNFACE + 1),
		LpszClassName: className,
		HIconSm:       smallIcon,
	}
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		panic(fmt.Sprintf("RegisterClassExW failed: %v", e))
	}
	style := uintptr(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_THICKFRAME | WS_VSCROLL | WS_HSCROLL)
	hwnd, _, e := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16ptr(appTitle))), style,
		CW_USEDEFAULT, CW_USEDEFAULT, 1160, 900, 0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		panic(fmt.Sprintf("CreateWindowExW failed: %v", e))
	}
	app.hwnd = hwnd
	if bigIcon != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, bigIcon)
	}
	if smallIcon != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, smallIcon)
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
	if err := app.di.Start(hwnd); err != nil {
		app.postStatus("DirectInput unavailable: " + err.Error())
	} else {
		app.populateCombos()
		app.applyConfigToUI()
		app.updateContextVisibility()
	}
	app.updateScrollbars()
	app.refreshHidHideUI()

	app.pollWG.Add(1)
	go app.pollLoop()

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	app.stopOnce.Do(func() { close(app.stopCh) })
	app.pollWG.Wait()
	app.releaseAllKeyboardOutputs()
	app.xbox.Stop()
	app.di.Stop()
}

func defaultConfig() config {
	return config{
		InputBackend:    "Auto",
		ControlMode:     "HOSAS",
		HotasThrottleID: -1, HotasThrottleAxis: 1,
		HotasStickID: -1, HotasStickXAxis: 0, HotasStickYAxis: 1,
		LeftID: 0, RightID: 1, PedalID: -1,
		LeftAxis: 1, RightAxis: 1, PedalAxis: 0,
		PedalMode: "Off", Deadzone: 6,
		PreserveThrust:         false,
		PedalSteerStrength:     30,
		CameraSensitivity:      100,
		CameraDeadzone:         6,
		PedalLeftKey:           VK_DELETE,
		PedalRightKey:          VK_NEXT,
		PedalKeyThreshold:      20,
		ThumbWASDEnabled:       false,
		ThumbWASDDeviceID:      -1,
		ThumbWASDSideAxis:      0,
		ThumbWASDForwardAxis:   1,
		ThumbWASDInvertSide:    false,
		ThumbWASDInvertForward: false,
		ThumbWASDThreshold:     20,
		ThumbWASDUseRawHID:     false,
		OnFootMode:             "Off",
		ThumbDISideAxis:        3,
		ThumbDIForwardAxis:     4,
		PollingHz:              500,
		Bindings:               make(map[string]binding),
	}
}

func configPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		if b, err := os.UserConfigDir(); err == nil {
			base = b
		} else {
			base = "."
		}
	}
	dir := filepath.Join(base, "GalacticRacerHOSAS")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "hosas_config.json")
}

func loadConfig(c *config) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return
	}
	// First try normal decoding. The schema intentionally matches v0.6 field names.
	var tmp config
	if err := json.Unmarshal(data, &tmp); err == nil {
		if tmp.Bindings == nil {
			tmp.Bindings = make(map[string]binding)
		}
		if tmp.Deadzone < 0 || tmp.Deadzone > 40 {
			tmp.Deadzone = 6
		}
		if tmp.PedalSteerStrength <= 0 || tmp.PedalSteerStrength > 100 {
			tmp.PedalSteerStrength = 30
		}
		if tmp.CameraSensitivity <= 0 || tmp.CameraSensitivity > 250 {
			tmp.CameraSensitivity = 100
		}
		if tmp.CameraDeadzone < 0 || tmp.CameraDeadzone > 40 {
			tmp.CameraDeadzone = 6
		}
		if !validKeyboardVK(tmp.PedalLeftKey) {
			tmp.PedalLeftKey = VK_DELETE
		}
		if !validKeyboardVK(tmp.PedalRightKey) {
			tmp.PedalRightKey = VK_NEXT
		}
		if tmp.PedalKeyThreshold < 5 || tmp.PedalKeyThreshold > 95 {
			tmp.PedalKeyThreshold = 20
		}
		// New v1.2 fields need explicit defaults when loading an older config,
		// because missing integer JSON fields otherwise decode as zero.
		var rawKeys map[string]json.RawMessage
		_ = json.Unmarshal(data, &rawKeys)
		if _, ok := rawKeys["input_backend"]; !ok {
			// Preserve the known-good behavior for existing users. New installs default to Auto.
			tmp.InputBackend = "WinMM"
		}
		tmp.InputBackend = normalizeInputBackend(tmp.InputBackend)
		if _, ok := rawKeys["control_mode"]; !ok {
			tmp.ControlMode = "HOSAS"
		}
		tmp.ControlMode = normalizeControlMode(tmp.ControlMode)
		if _, ok := rawKeys["hotas_throttle_id"]; !ok {
			tmp.HotasThrottleID = -1
		}
		if _, ok := rawKeys["hotas_stick_id"]; !ok {
			tmp.HotasStickID = -1
		}
		maxAxis := 7
		if normalizeInputBackend(tmp.InputBackend) == "WinMM" {
			maxAxis = 5
		}
		if tmp.HotasThrottleAxis < 0 || tmp.HotasThrottleAxis > maxAxis {
			tmp.HotasThrottleAxis = 1
		}
		if tmp.HotasStickXAxis < 0 || tmp.HotasStickXAxis > maxAxis {
			tmp.HotasStickXAxis = 0
		}
		if tmp.HotasStickYAxis < 0 || tmp.HotasStickYAxis > maxAxis {
			tmp.HotasStickYAxis = 1
		}
		if tmp.LeftAxis < 0 || tmp.LeftAxis > maxAxis {
			tmp.LeftAxis = 1
		}
		if tmp.RightAxis < 0 || tmp.RightAxis > maxAxis {
			tmp.RightAxis = 1
		}
		if tmp.PedalAxis < 0 || tmp.PedalAxis > maxAxis {
			tmp.PedalAxis = 0
		}
		if _, ok := rawKeys["thumb_wasd_device_id"]; !ok {
			tmp.ThumbWASDDeviceID = -1
		}
		if _, ok := rawKeys["thumb_wasd_side_axis"]; !ok {
			tmp.ThumbWASDSideAxis = 0
		}
		if _, ok := rawKeys["thumb_wasd_forward_axis"]; !ok {
			tmp.ThumbWASDForwardAxis = 1
		}
		if _, ok := rawKeys["on_foot_mode"]; !ok {
			if tmp.ThumbWASDEnabled {
				tmp.OnFootMode = "Thumbstick -> WASD"
			} else {
				tmp.OnFootMode = "Off"
			}
		}
		tmp.OnFootMode = normalizeOnFootMode(tmp.OnFootMode)
		if tmp.ThumbDISideAxis < 0 || tmp.ThumbDISideAxis > 7 {
			tmp.ThumbDISideAxis = 3
		}
		if tmp.ThumbDIForwardAxis < 0 || tmp.ThumbDIForwardAxis > 7 {
			tmp.ThumbDIForwardAxis = 4
		}
		if tmp.ThumbWASDThreshold < 5 || tmp.ThumbWASDThreshold > 95 {
			tmp.ThumbWASDThreshold = 20
		}
		if _, ok := rawKeys["polling_hz"]; !ok {
			tmp.PollingHz = 500
		}
		if !validPollingHz(tmp.PollingHz) {
			tmp.PollingHz = 500
		}
		if tmp.ThumbWASDSideAxis < 0 || tmp.ThumbWASDSideAxis > 5 {
			tmp.ThumbWASDSideAxis = 0
		}
		if tmp.ThumbWASDForwardAxis < 0 || tmp.ThumbWASDForwardAxis > 5 {
			tmp.ThumbWASDForwardAxis = 1
		}
		tmp.PedalMode = normalizePedalMode(tmp.PedalMode)
		tmp.Bindings = normalizeBindingMap(tmp.Bindings)
		*c = tmp
		return
	}
	// Migration fallback for any old axis values that were saved as names instead of indices.
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return
	}
	d := defaultConfig()
	if b, ok := raw["input_backend"]; ok {
		_ = json.Unmarshal(b, &d.InputBackend)
	} else {
		d.InputBackend = "WinMM"
	}
	d.InputBackend = normalizeInputBackend(d.InputBackend)
	readInt(raw, "left_id", &d.LeftID)
	readInt(raw, "right_id", &d.RightID)
	readInt(raw, "pedal_id", &d.PedalID)
	readAxis(raw, "left_axis", &d.LeftAxis)
	readAxis(raw, "right_axis", &d.RightAxis)
	readAxis(raw, "pedal_axis", &d.PedalAxis)
	readBool(raw, "left_invert", &d.LeftInvert)
	readBool(raw, "right_invert", &d.RightInvert)
	readBool(raw, "left_throttle_mode", &d.LeftThrottleMode)
	readBool(raw, "right_throttle_mode", &d.RightThrottleMode)
	readBool(raw, "pedal_invert", &d.PedalInvert)
	readPedalMode(raw, "pedal_mode", &d.PedalMode)
	readDeadzone(raw, "deadzone", &d.Deadzone)
	readBool(raw, "preserve_thrust", &d.PreserveThrust)
	readInt(raw, "pedal_steer_strength", &d.PedalSteerStrength)
	readInt(raw, "camera_sensitivity", &d.CameraSensitivity)
	readInt(raw, "camera_deadzone", &d.CameraDeadzone)
	readInt(raw, "pedal_left_key", &d.PedalLeftKey)
	readInt(raw, "pedal_right_key", &d.PedalRightKey)
	readInt(raw, "pedal_key_threshold", &d.PedalKeyThreshold)
	readBool(raw, "thumb_wasd_enabled", &d.ThumbWASDEnabled)
	if b, ok := raw["on_foot_mode"]; ok {
		_ = json.Unmarshal(b, &d.OnFootMode)
	} else if d.ThumbWASDEnabled {
		d.OnFootMode = "Thumbstick -> WASD"
	}
	readInt(raw, "thumb_wasd_device_id", &d.ThumbWASDDeviceID)
	readAxis(raw, "thumb_wasd_side_axis", &d.ThumbWASDSideAxis)
	readAxis(raw, "thumb_wasd_forward_axis", &d.ThumbWASDForwardAxis)
	readBool(raw, "thumb_wasd_invert_side", &d.ThumbWASDInvertSide)
	readBool(raw, "thumb_wasd_invert_forward", &d.ThumbWASDInvertForward)
	readInt(raw, "thumb_wasd_threshold", &d.ThumbWASDThreshold)
	if b, ok := raw["thumb_di_side_guid"]; ok {
		_ = json.Unmarshal(b, &d.ThumbDISideGUID)
	}
	if b, ok := raw["thumb_di_forward_guid"]; ok {
		_ = json.Unmarshal(b, &d.ThumbDIForwardGUID)
	}
	if b, ok := raw["thumb_di_device_name"]; ok {
		_ = json.Unmarshal(b, &d.ThumbDIDeviceName)
	}
	readInt(raw, "thumb_di_side_axis", &d.ThumbDISideAxis)
	readInt(raw, "thumb_di_forward_axis", &d.ThumbDIForwardAxis)
	readInt(raw, "polling_hz", &d.PollingHz)
	if d.CameraSensitivity <= 0 || d.CameraSensitivity > 250 {
		d.CameraSensitivity = 100
	}
	if d.CameraDeadzone < 0 || d.CameraDeadzone > 40 {
		d.CameraDeadzone = 6
	}
	if !validKeyboardVK(d.PedalLeftKey) {
		d.PedalLeftKey = VK_DELETE
	}
	if !validKeyboardVK(d.PedalRightKey) {
		d.PedalRightKey = VK_NEXT
	}
	if d.PedalKeyThreshold < 5 || d.PedalKeyThreshold > 95 {
		d.PedalKeyThreshold = 20
	}
	if d.ThumbWASDThreshold < 5 || d.ThumbWASDThreshold > 95 {
		d.ThumbWASDThreshold = 20
	}
	if !validPollingHz(d.PollingHz) {
		d.PollingHz = 500
	}
	if d.ThumbWASDSideAxis < 0 || d.ThumbWASDSideAxis > 5 {
		d.ThumbWASDSideAxis = 0
	}
	if d.ThumbWASDForwardAxis < 0 || d.ThumbWASDForwardAxis > 5 {
		d.ThumbWASDForwardAxis = 1
	}
	d.PedalMode = normalizePedalMode(d.PedalMode)
	d.OnFootMode = normalizeOnFootMode(d.OnFootMode)
	if b, ok := raw["bindings"]; ok {
		_ = json.Unmarshal(b, &d.Bindings)
		d.Bindings = normalizeBindingMap(d.Bindings)
	}
	*c = d
}

func readInt(raw map[string]json.RawMessage, k string, dst *int) {
	b, ok := raw[k]
	if !ok {
		return
	}
	var n int
	if json.Unmarshal(b, &n) == nil {
		*dst = n
		return
	}
	var f float64
	if json.Unmarshal(b, &f) == nil {
		*dst = int(math.Round(f))
		return
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		if n, e := strconv.Atoi(s); e == nil {
			*dst = n
		}
	}
}
func readBool(raw map[string]json.RawMessage, k string, dst *bool) {
	if b, ok := raw[k]; ok {
		_ = json.Unmarshal(b, dst)
	}
}
func readString(raw map[string]json.RawMessage, k string, dst *string) {
	if b, ok := raw[k]; ok {
		_ = json.Unmarshal(b, dst)
	}
}

func normalizeInputBackend(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "directinput", "direct input", "di":
		return "DirectInput"
	case "winmm", "win mm", "legacy":
		return "WinMM"
	default:
		return "Auto"
	}
}

func normalizeControlMode(s string) string {
	u := strings.ToUpper(strings.TrimSpace(s))
	if strings.Contains(u, "HOTAS") {
		return "HOTAS"
	}
	return "HOSAS"
}

func normalizePedalMode(s string) string {
	ls := strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.Contains(ls, "keyboard") || strings.Contains(ls, "key"):
		return "Keyboard keys"
	case strings.Contains(ls, "fine") || strings.Contains(ls, "steer"):
		return "Fine steering"
	case strings.Contains(ls, "camera") || strings.Contains(ls, "right") || strings.Contains(ls, "rx"):
		return "Move Camera"
	default:
		return "Off"
	}
}

func normalizeOnFootMode(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "thumbstick -> wasd", "thumbstick", "analog", "thumbstick to wasd":
		return "Thumbstick -> WASD"
	case "buttons -> wasd", "buttons", "button", "buttons to wasd":
		return "Buttons -> WASD"
	default:
		return "Off"
	}
}

func readPedalMode(raw map[string]json.RawMessage, k string, dst *string) {
	b, ok := raw[k]
	if !ok {
		return
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		*dst = normalizePedalMode(s)
		return
	}
	var n int
	if json.Unmarshal(b, &n) == nil {
		if n == 1 {
			*dst = "Fine steering"
		} else if n == 2 {
			*dst = "Move Camera"
		} else if n == 3 {
			*dst = "Keyboard keys"
		} else {
			*dst = "Off"
		}
	}
}

func readDeadzone(raw map[string]json.RawMessage, k string, dst *int) {
	b, ok := raw[k]
	if !ok {
		return
	}
	var f float64
	if json.Unmarshal(b, &f) == nil {
		if f > 0 && f <= 1 {
			f *= 100
		}
		*dst = int(math.Round(f))
	}
}

func readAxis(raw map[string]json.RawMessage, k string, dst *int) {
	b, ok := raw[k]
	if !ok {
		return
	}
	var n int
	if json.Unmarshal(b, &n) == nil {
		*dst = n
		return
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		for i, a := range []string{"X", "Y", "Z", "R", "U", "V"} {
			if strings.EqualFold(s, a) {
				*dst = i
				return
			}
		}
	}
}

func normalizeBindingMap(in map[string]binding) map[string]binding {
	out := make(map[string]binding)
	for k, v := range in {
		out[normalizeBindingKey(k)] = v
	}
	return out
}
func normalizeBindingKey(k string) string {
	s := strings.ToUpper(strings.TrimSpace(k))
	r := strings.NewReplacer(" ", "", "-", "", "_", "", "/", "")
	s = r.Replace(s)
	aliases := map[string]string{
		"XBOXA": "A", "XBOXB": "B", "XBOXX": "X", "XBOXY": "Y",
		"LEFTBUMPER": "LB", "RIGHTBUMPER": "RB", "L1": "LB", "R1": "RB",
		"BACKVIEW": "BACK", "VIEW": "BACK", "STARTMENU": "START", "MENU": "START",
		"LEFTSTICKCLICK": "LTHUMB", "RIGHTSTICKCLICK": "RTHUMB", "LTHUMB": "LTHUMB", "RTHUMB": "RTHUMB",
		"DPADUP": "DUP", "DPADDOWN": "DDOWN", "DPADLEFT": "DLEFT", "DPADRIGHT": "DRIGHT",
	}
	if a, ok := aliases[s]; ok {
		return a
	}
	return s
}

func validKeyboardVK(v int) bool {
	for _, k := range keyboardKeyOptions {
		if k.VK == v {
			return true
		}
	}
	return false
}

func keyOptionIndex(vk int) int {
	for i, k := range keyboardKeyOptions {
		if k.VK == vk {
			return i
		}
	}
	return 0
}

func selectedKeyVK(id int) int {
	i := comboIndex(id)
	if i >= 0 && i < len(keyboardKeyOptions) {
		return keyboardKeyOptions[i].VK
	}
	return keyboardKeyOptions[0].VK
}

func keyName(vk int) string {
	for _, k := range keyboardKeyOptions {
		if k.VK == vk {
			return k.Name
		}
	}
	return fmt.Sprintf("VK 0x%02X", vk)
}

func saveConfig() {
	app.cfgMu.RLock()
	c := app.cfg
	c.Bindings = make(map[string]binding, len(app.cfg.Bindings))
	for k, v := range app.cfg.Bindings {
		c.Bindings[k] = v
	}
	app.cfgMu.RUnlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	p := configPath()
	tmp := p + ".tmp"
	if os.WriteFile(tmp, data, 0644) == nil {
		_ = os.Remove(p)
		_ = os.Rename(tmp, p)
	}
}

func enumerateDevices() []deviceInfo {
	n, _, _ := procJoyGetNumDevs.Call()
	out := make([]deviceInfo, 0)
	for i := 0; i < int(n); i++ {
		var caps joyCaps
		r, _, _ := procJoyGetDevCapsW.Call(uintptr(i), uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps))
		if uint32(r) != JOYERR_NOERROR {
			continue
		}
		name := utf16SliceToString(caps.Pname[:])
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("Joystick %d", i)
		}
		out = append(out, deviceInfo{ID: i, Name: name, Caps: caps})
	}
	return out
}

func utf16SliceToString(a []uint16) string {
	n := 0
	for n < len(a) && a[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(a[:n])
}

func readJoy(id int) (joyInfoEx, bool) {
	var j joyInfoEx
	j.Size = uint32(unsafe.Sizeof(j))
	j.Flags = JOY_RETURNALL
	app.winmmMu.Lock()
	r, _, _ := procJoyGetPosEx.Call(uintptr(id), uintptr(unsafe.Pointer(&j)))
	app.winmmMu.Unlock()
	return j, uint32(r) == JOYERR_NOERROR
}

func axisRaw(j joyInfoEx, axis int) uint32 {
	switch axis {
	case 0:
		return j.X
	case 1:
		return j.Y
	case 2:
		return j.Z
	case 3:
		return j.R
	case 4:
		return j.U
	case 5:
		return j.V
	}
	return j.Y
}
func axisRange(c joyCaps, axis int) (uint32, uint32) {
	switch axis {
	case 0:
		return c.Xmin, c.Xmax
	case 1:
		return c.Ymin, c.Ymax
	case 2:
		return c.Zmin, c.Zmax
	case 3:
		return c.Rmin, c.Rmax
	case 4:
		return c.Umin, c.Umax
	case 5:
		return c.Vmin, c.Vmax
	}
	return 0, 65535
}
func axisNormalized(deviceID, axis int, invert bool, j joyInfoEx) float64 {
	app.devicesMu.RLock()
	var caps *joyCaps
	for i := range app.devices {
		if app.devices[i].ID == deviceID {
			cp := app.devices[i].Caps
			caps = &cp
			break
		}
	}
	app.devicesMu.RUnlock()
	minv, maxv := uint32(0), uint32(65535)
	if caps != nil {
		minv, maxv = axisRange(*caps, axis)
	}
	if maxv <= minv {
		minv, maxv = 0, 65535
	}
	raw := axisRaw(j, axis)
	v := ((float64(raw)-float64(minv))/float64(maxv-minv))*2.0 - 1.0
	if invert {
		v = -v
	}
	return clamp(v, -1, 1)
}

// axisThrottleCenteredNormalized treats an absolute throttle axis as a bidirectional
// HOSAS input: 0% travel = -1, 50% travel = 0, 100% travel = +1.
// This is intentionally explicit even though many symmetric WinMM axes normalize
// similarly, because throttle devices are not spring-centered and users need a
// predictable midpoint-neutral mode.
func axisThrottleCenteredNormalized(deviceID, axis int, invert bool, j joyInfoEx) float64 {
	app.devicesMu.RLock()
	var caps *joyCaps
	for i := range app.devices {
		if app.devices[i].ID == deviceID {
			cp := app.devices[i].Caps
			caps = &cp
			break
		}
	}
	app.devicesMu.RUnlock()
	minv, maxv := uint32(0), uint32(65535)
	if caps != nil {
		minv, maxv = axisRange(*caps, axis)
	}
	if maxv <= minv {
		minv, maxv = 0, 65535
	}
	raw := float64(axisRaw(j, axis))
	minf, maxf := float64(minv), float64(maxv)
	mid := minf + (maxf-minf)*0.5
	var v float64
	if raw >= mid {
		span := maxf - mid
		if span > 0 {
			v = (raw - mid) / span
		}
	} else {
		span := mid - minf
		if span > 0 {
			v = (raw - mid) / span
		}
	}
	if invert {
		v = -v
	}
	return clamp(v, -1, 1)
}

func deviceNameByID(id int) string {
	app.devicesMu.RLock()
	defer app.devicesMu.RUnlock()
	for _, d := range app.devices {
		if d.ID == id {
			return d.Name
		}
	}
	return ""
}

func normalizedDeviceKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (a *appState) matchWinMMIDByName(name string) int {
	key := normalizedDeviceKey(name)
	if key == "" {
		return -1
	}
	a.devicesMu.RLock()
	defer a.devicesMu.RUnlock()
	best := -1
	for _, d := range a.devices {
		dk := normalizedDeviceKey(d.Name)
		if dk == key {
			return d.ID
		}
		if dk != "" && (strings.Contains(dk, key) || strings.Contains(key, dk)) {
			if best < 0 {
				best = d.ID
			}
		}
	}
	return best
}

func (a *appState) readConfiguredAxis(backend, guid string, deviceID, axis int, invert bool, winStates map[int]joyInfoEx, winOK map[int]bool) (float64, bool) {
	backend = normalizeInputBackend(backend)
	if backend != "WinMM" && guid != "" && axis >= 0 && axis < 8 {
		if axes, _, ok := a.di.StateByGUID(guid); ok {
			return normalizeDIAbsoluteAxis(axes[axis], invert), true
		}
		if backend == "DirectInput" {
			return 0, false
		}
	}
	if deviceID >= 0 && axis >= 0 && axis < 6 && winOK[deviceID] {
		return axisNormalized(deviceID, axis, invert, winStates[deviceID]), true
	}
	return 0, false
}

func (a *appState) readConfiguredThrottleCentered(backend, guid string, deviceID, axis int, invert bool, winStates map[int]joyInfoEx, winOK map[int]bool) (float64, bool) {
	backend = normalizeInputBackend(backend)
	if backend != "WinMM" && guid != "" && axis >= 0 && axis < 8 {
		if axes, _, ok := a.di.StateByGUID(guid); ok {
			// DirectInput absolute axes already normalize naturally to -1..+1 around midpoint.
			return normalizeDIAbsoluteAxis(axes[axis], invert), true
		}
		if backend == "DirectInput" {
			return 0, false
		}
	}
	if deviceID >= 0 && axis >= 0 && axis < 6 && winOK[deviceID] {
		return axisThrottleCenteredNormalized(deviceID, axis, invert, winStates[deviceID]), true
	}
	return 0, false
}

func applyDeadzone(v float64, pct int) float64 {
	dz := clamp(float64(pct)/100.0, 0, 0.40)
	a := math.Abs(v)
	if a <= dz {
		return 0
	}
	out := (a - dz) / (1 - dz)
	if v < 0 {
		out = -out
	}
	return clamp(out, -1, 1)
}

func calculateDrive(left, right float64, preserve bool) float64 {
	if !preserve {
		return clamp((left+right)/2.0, -1, 1)
	}
	// Hybrid preserve-thrust mode:
	// 1) Both forward: strongest forward stick sets thrust, so differential steering costs no thrust.
	// 2) One forward + one backward: the backward stick progressively scrubs that forward thrust.
	//    Example R=+1: L=0 => 100%, L=-.25 => 75%, L=-.5 => 50%, L=-1 => 0%.
	// 3) Both backward: retain the original averaging behavior, so both at -1 remains full brake/reverse.
	if left >= 0 && right >= 0 {
		return math.Max(left, right)
	}
	if left <= 0 && right <= 0 {
		return clamp((left+right)/2.0, -1, 0)
	}
	if left > 0 && right < 0 {
		return clamp(left*(1-math.Abs(right)), 0, 1)
	}
	if right > 0 && left < 0 {
		return clamp(right*(1-math.Abs(left)), 0, 1)
	}
	return 0
}

func monitorHorizontalAxis(forwardAxis int) int {
	if forwardAxis == 0 {
		return 1
	}
	return 0
}

func (a *appState) pollLoop() {
	defer a.pollWG.Done()
	a.cfgMu.RLock()
	currentInterval := pollingInterval(a.cfg.PollingHz)
	a.cfgMu.RUnlock()
	ticker := time.NewTicker(currentInterval)
	defer func() { ticker.Stop() }()
	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.cfgMu.RLock()
			c := a.cfg
			a.cfgMu.RUnlock()
			desiredInterval := pollingInterval(c.PollingHz)
			if desiredInterval != currentInterval {
				ticker.Stop()
				ticker = time.NewTicker(desiredInterval)
				currentInterval = desiredInterval
			}

			ids := []int{}
			if strings.EqualFold(normalizeControlMode(c.ControlMode), "HOTAS") {
				ids = append(ids, c.HotasThrottleID, c.HotasStickID)
			} else {
				ids = append(ids, c.LeftID, c.RightID)
			}
			if c.PedalID >= 0 {
				ids = append(ids, c.PedalID)
			}
			for _, b := range c.Bindings {
				if b.DeviceID >= 0 {
					ids = append(ids, b.DeviceID)
				}
			}

			states := make(map[int]joyInfoEx)
			okmap := make(map[int]bool)
			for _, id := range ids {
				if id < 0 {
					continue
				}
				if _, exists := states[id]; exists {
					continue
				}
				st, ok := readJoy(id)
				states[id] = st
				okmap[id] = ok
			}

			l, r := 0.0, 0.0
			hotasThrottle, hotasX, hotasY := 0.0, 0.0, 0.0
			pedalRaw, p, cameraX := 0.0, 0.0, 0.0
			isHOTAS := strings.EqualFold(normalizeControlMode(c.ControlMode), "HOTAS")
			backend := normalizeInputBackend(c.InputBackend)
			if isHOTAS {
				if raw, ok := a.readConfiguredAxis(backend, c.HotasThrottleDIGUID, c.HotasThrottleID, c.HotasThrottleAxis, c.HotasThrottleInvert, states, okmap); ok {
					hotasThrottle = clamp((raw+1.0)/2.0, 0, 1)
				}
				if raw, ok := a.readConfiguredAxis(backend, c.HotasStickDIGUID, c.HotasStickID, c.HotasStickXAxis, c.HotasStickInvertX, states, okmap); ok {
					hotasX = applyDeadzone(raw, c.Deadzone)
				}
				if raw, ok := a.readConfiguredAxis(backend, c.HotasStickDIGUID, c.HotasStickID, c.HotasStickYAxis, c.HotasStickInvertY, states, okmap); ok {
					hotasY = applyDeadzone(raw, c.Deadzone)
				}
			} else {
				if c.LeftThrottleMode {
					if raw, ok := a.readConfiguredThrottleCentered(backend, c.LeftDIGUID, c.LeftID, c.LeftAxis, c.LeftInvert, states, okmap); ok {
						l = applyDeadzone(raw, c.Deadzone)
					}
				} else if raw, ok := a.readConfiguredAxis(backend, c.LeftDIGUID, c.LeftID, c.LeftAxis, c.LeftInvert, states, okmap); ok {
					l = applyDeadzone(raw, c.Deadzone)
				}
				if c.RightThrottleMode {
					if raw, ok := a.readConfiguredThrottleCentered(backend, c.RightDIGUID, c.RightID, c.RightAxis, c.RightInvert, states, okmap); ok {
						r = applyDeadzone(raw, c.Deadzone)
					}
				} else if raw, ok := a.readConfiguredAxis(backend, c.RightDIGUID, c.RightID, c.RightAxis, c.RightInvert, states, okmap); ok {
					r = applyDeadzone(raw, c.Deadzone)
				}
			}
			if raw, ok := a.readConfiguredAxis(backend, c.PedalDIGUID, c.PedalID, c.PedalAxis, c.PedalInvert, states, okmap); ok {
				pedalRaw = raw
				p = applyDeadzone(pedalRaw, c.Deadzone)
				cameraX = clamp(
					applyDeadzone(pedalRaw, c.CameraDeadzone)*float64(c.CameraSensitivity)/100.0,
					-1, 1,
				)
			}

			drive := calculateDrive(l, r, c.PreserveThrust)
			steer := clamp(l-r, -1, 1)
			if isHOTAS {
				drive = hotasThrottle
				steer = hotasX
			}
			if strings.EqualFold(c.PedalMode, "Fine steering") {
				steer = clamp(steer+p*float64(c.PedalSteerStrength)/100.0, -1, 1)
			}

			var buttonMask uint16
			var outputHighlights uint32
			keyboardWanted := make(map[uint16]bool)
			for i, bo := range buttonOutputs {
				pressed := false
				if b, ok := c.Bindings[bo.Key]; ok {
					pressed = bindingPressed(b, states, okmap)
				}
				if pressed {
					outputHighlights |= uint32(1) << uint(i)
					if bo.Mask != 0 {
						buttonMask |= bo.Mask
					}
				}
				if bo.VK != 0 {
					keyboardWanted[bo.VK] = keyboardWanted[bo.VK] || (pressed && a.mappingActive.Load())
				}
			}
			if strings.EqualFold(c.PedalMode, "Keyboard keys") && a.mappingActive.Load() && (c.PedalID >= 0 || c.PedalDIGUID != "") {
				threshold := clamp(float64(c.PedalKeyThreshold)/100.0, 0.05, 0.95)
				if pedalRaw <= -threshold {
					keyboardWanted[uint16(c.PedalLeftKey)] = true
				} else if pedalRaw >= threshold {
					keyboardWanted[uint16(c.PedalRightKey)] = true
				}
			}

			// On-foot controls are selectable: DirectInput thumbstick axes or learned buttons.
			var onFootHighlights uint8
			onFootMode := normalizeOnFootMode(c.OnFootMode)
			if strings.EqualFold(onFootMode, "Thumbstick -> WASD") && a.mappingActive.Load() {
				var sideAxes, forwardAxes [8]int32
				gotSide, gotForward := false, false
				if c.ThumbDISideGUID != "" {
					sideAxes, gotSide = a.di.State(c.ThumbDISideGUID)
				}
				if c.ThumbDIForwardGUID != "" {
					if strings.EqualFold(c.ThumbDIForwardGUID, c.ThumbDISideGUID) && gotSide {
						forwardAxes, gotForward = sideAxes, true
					} else {
						forwardAxes, gotForward = a.di.State(c.ThumbDIForwardGUID)
					}
				}
				if gotSide && gotForward && c.ThumbDISideAxis >= 0 && c.ThumbDISideAxis < 8 && c.ThumbDIForwardAxis >= 0 && c.ThumbDIForwardAxis < 8 {
					side := normalizeDIAxis(sideAxes[c.ThumbDISideAxis], c.ThumbDISideCenter, c.ThumbDISideSpan)
					forward := normalizeDIAxis(forwardAxes[c.ThumbDIForwardAxis], c.ThumbDIForwardCenter, c.ThumbDIForwardSpan)
					if c.ThumbWASDInvertSide {
						side = -side
					}
					if c.ThumbWASDInvertForward {
						forward = -forward
					}
					threshold := clamp(float64(c.ThumbWASDThreshold)/100.0, 0.05, 0.95)
					if forward >= threshold {
						keyboardWanted[uint16('W')] = true
					} else if forward <= -threshold {
						keyboardWanted[uint16('S')] = true
					}
					if side <= -threshold {
						keyboardWanted[uint16('A')] = true
					} else if side >= threshold {
						keyboardWanted[uint16('D')] = true
					}
				}
			} else if strings.EqualFold(onFootMode, "Buttons -> WASD") {
				for i, def := range wasdButtonDefs {
					pressed := false
					if b, ok := c.Bindings[def.Key]; ok {
						pressed = bindingPressed(b, states, okmap)
					}
					if pressed {
						onFootHighlights |= uint8(1 << uint(i))
						if a.mappingActive.Load() {
							keyboardWanted[def.VK] = true
						}
					}
				}
			}
			a.applyKeyboardOutputs(keyboardWanted)

			// Live monitor visuals intentionally do not do the extra axis work while
			// game mapping is active. The controller path remains at the selected polling rate.
			leftX, leftY, rightX, rightY := 0.0, l, 0.0, r
			active := a.mappingActive.Load()
			if isHOTAS {
				leftX, leftY = hotasX, hotasY
				rightX, rightY = 0, 0
			} else if !active {
				hax := monitorHorizontalAxis(c.LeftAxis)
				if raw, ok := a.readConfiguredAxis(backend, c.LeftDIGUID, c.LeftID, hax, false, states, okmap); ok {
					leftX = applyDeadzone(raw, c.Deadzone)
				}
				hax = monitorHorizontalAxis(c.RightAxis)
				if raw, ok := a.readConfiguredAxis(backend, c.RightDIGUID, c.RightID, hax, false, states, okmap); ok {
					rightX = applyDeadzone(raw, c.Deadzone)
				}
			}

			a.monitorMu.Lock()
			a.monitor = monitorState{
				Left:             l,
				Right:            r,
				Pedal:            p,
				Drive:            drive,
				Steer:            steer,
				LeftX:            leftX,
				LeftY:            leftY,
				RightX:           rightX,
				RightY:           rightY,
				CameraX:          cameraX,
				OutputButtons:    buttonMask,
				OutputHighlights: outputHighlights,
				OnFootHighlights: onFootHighlights,
				LastUpdate:       time.Now(),
			}
			a.monitorMu.Unlock()

			if a.xboxReady.Load() {
				var rep xusbReport

				a.testUntilMu.RLock()
				testA := time.Now().Before(a.testUntil)
				a.testUntilMu.RUnlock()

				a.cameraTestMu.RLock()
				cameraTest := time.Now().Before(a.cameraTestUntil)
				cameraTestValue := a.cameraTestValue
				a.cameraTestMu.RUnlock()

				if active {
					if isHOTAS {
						rep.RightTrigger = byte(math.Round(clamp(hotasThrottle, 0, 1) * 255))
						rep.ThumbLX = floatToStick(steer)
						rep.ThumbLY = floatToStick(hotasY)
					} else {
						if drive >= 0 {
							rep.RightTrigger = byte(math.Round(clamp(drive, 0, 1) * 255))
						} else {
							rep.LeftTrigger = byte(math.Round(clamp(-drive, 0, 1) * 255))
						}
						rep.ThumbLX = floatToStick(steer)
					}
					if strings.EqualFold(c.PedalMode, "Move Camera") {
						rep.ThumbRX = floatToStick(cameraX)
					}
					rep.Buttons = buttonMask
				}

				if cameraTest {
					rep.ThumbRX = floatToStick(cameraTestValue)
				}
				if testA {
					rep.Buttons |= XUSB_A
				}
				_ = a.xbox.Update(rep)
			}
		}
	}
}

func (a *appState) applyKeyboardOutputs(wanted map[uint16]bool) {
	a.keyboardMu.Lock()
	defer a.keyboardMu.Unlock()
	if a.keyboardDown == nil {
		a.keyboardDown = make(map[uint16]bool)
	}
	all := make(map[uint16]bool, len(wanted)+len(a.keyboardDown))
	for vk := range wanted {
		if vk != 0 {
			all[vk] = true
		}
	}
	for vk := range a.keyboardDown {
		if vk != 0 {
			all[vk] = true
		}
	}
	for vk := range all {
		want := wanted[vk]
		isDown := a.keyboardDown[vk]
		if want == isDown {
			continue
		}
		flags := uintptr(0)
		if !want {
			flags = KEYEVENTF_KEYUP
		}
		procKeybdEvent.Call(uintptr(byte(vk)), 0, flags, 0)
		a.keyboardDown[vk] = want
	}
}

func (a *appState) releaseAllKeyboardOutputs() {
	a.keyboardMu.Lock()
	defer a.keyboardMu.Unlock()
	for vk, down := range a.keyboardDown {
		if down {
			procKeybdEvent.Call(uintptr(byte(vk)), 0, KEYEVENTF_KEYUP, 0)
		}
		a.keyboardDown[vk] = false
	}
}

func floatToStick(v float64) int16 {
	v = clamp(v, -1, 1)
	if v < 0 {
		return int16(math.Round(v * 32768))
	}
	return int16(math.Round(v * 32767))
}

func bindingPressed(b binding, states map[int]joyInfoEx, oks map[int]bool) bool {
	if !oks[b.DeviceID] {
		return false
	}
	st := states[b.DeviceID]
	if strings.EqualFold(b.Kind, "button") {
		if b.Button < 0 || b.Button > 31 {
			return false
		}
		return st.Buttons&(uint32(1)<<uint(b.Button)) != 0
	}
	if strings.EqualFold(b.Kind, "pov") {
		if st.POV == JOY_POVCENTERED {
			return false
		}
		d := angleDistance(int(st.POV), b.POV)
		return d <= 2300
	}
	return false
}
func angleDistance(a, b int) int {
	d := int(math.Abs(float64(a - b)))
	if d > 18000 {
		d = 36000 - d
	}
	return d
}

func (a *appState) startMappingAsync() {
	a.postStatus("Creating virtual Xbox controller...")
	before := xinputConnectedSlots()
	if err := a.xbox.Start(); err != nil {
		a.mappingStarting.Store(false)
		a.mappingActive.Store(false)
		a.xboxReady.Store(false)
		a.xboxSlot.Store(-1)
		a.postStatus("Xbox output ERROR: " + err.Error() + "  Run Install.cmd and restart Windows if ViGEmBus was just installed.")
		procPostMessageW.Call(a.hwnd, WM_APP_XBOX, 0, 0)
		return
	}

	// The target now exists. Enable output immediately; XInput slot detection can finish in parallel.
	a.xboxReady.Store(true)
	a.mappingActive.Store(true)

	slot := -1
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		after := xinputConnectedSlots()
		for _, s := range after {
			if !containsInt(before, s) {
				slot = s
				break
			}
		}
		if slot >= 0 {
			break
		}
	}
	if slot < 0 {
		after := xinputConnectedSlots()
		if len(after) > 0 && len(before) == 0 {
			slot = after[0]
		}
	}
	a.xboxSlot.Store(int32(slot))
	a.mappingStarting.Store(false)
	if slot >= 0 {
		a.postStatus(fmt.Sprintf("Mapping ACTIVE. Virtual Xbox created in Windows XInput slot %d.", slot+1))
	} else {
		a.postStatus("Mapping ACTIVE. Virtual Xbox created; XInput slot verification is inconclusive.")
	}
	procPostMessageW.Call(a.hwnd, WM_APP_XBOX, 0, 0)
}

func (a *appState) stopMapping() {
	// Stop output first so the polling loop no longer submits reports while the target is removed.
	a.mappingActive.Store(false)
	a.releaseAllKeyboardOutputs()
	a.xboxReady.Store(false)
	a.xboxSlot.Store(-1)

	a.testUntilMu.Lock()
	a.testUntil = time.Time{}
	a.testUntilMu.Unlock()
	a.cameraTestMu.Lock()
	a.cameraTestUntil = time.Time{}
	a.cameraTestValue = 0
	a.cameraTestMu.Unlock()

	a.xbox.Stop()
	a.postStatus("Mapping stopped. Virtual Xbox controller disconnected and released.")
	procPostMessageW.Call(a.hwnd, WM_APP_XBOX, 0, 0)
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

var (
	xinputOnce     sync.Once
	xinputDLL      *syscall.DLL
	xinputGetState *syscall.Proc
)

func getXInputGetState() *syscall.Proc {
	xinputOnce.Do(func() {
		for _, n := range []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"} {
			dll, err := syscall.LoadDLL(n)
			if err != nil {
				continue
			}
			p, err := dll.FindProc("XInputGetState")
			if err == nil {
				xinputDLL = dll // Keep loaded for the lifetime of the app.
				xinputGetState = p
				return
			}
			_ = dll.Release()
		}
	})
	return xinputGetState
}

func xinputConnectedSlots() []int {
	p := getXInputGetState()
	if p == nil {
		return nil
	}
	out := []int{}
	for i := 0; i < 4; i++ {
		var st xinputState
		r, _, _ := p.Call(uintptr(i), uintptr(unsafe.Pointer(&st)))
		if uint32(r) == 0 {
			out = append(out, i)
		}
	}
	return out
}

func xinputReadState(slot int) (xinputState, bool) {
	var st xinputState
	if slot < 0 || slot > 3 {
		return st, false
	}
	p := getXInputGetState()
	if p == nil {
		return st, false
	}
	r, _, _ := p.Call(uintptr(slot), uintptr(unsafe.Pointer(&st)))
	return st, uint32(r) == 0
}

func stickReadback(v int16) float64 {
	if v < 0 {
		return float64(v) / 32768.0
	}
	return float64(v) / 32767.0
}

func (x *xboxBridge) Start() error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.ready {
		return nil
	}

	// Prefer the DLL beside the running EXE, but do not assume a local/dev build
	// always copied it there. Fall back to the normal installed app directory.
	candidates := make([]string, 0, 3)
	if exe, err := os.Executable(); err == nil && exe != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "ViGEmClient.dll"))
	}
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		candidates = append(candidates, filepath.Join(base, "GalacticRacerHOSAS", "ViGEmClient.dll"))
	} else if base, err := os.UserConfigDir(); err == nil && base != "" {
		candidates = append(candidates, filepath.Join(base, "GalacticRacerHOSAS", "ViGEmClient.dll"))
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		candidates = append(candidates, filepath.Join(wd, "ViGEmClient.dll"))
	}

	seen := make(map[string]bool)
	checked := make([]string, 0, len(candidates))
	var dll *syscall.DLL
	var err error
	for _, path := range candidates {
		clean := filepath.Clean(path)
		key := strings.ToLower(clean)
		if seen[key] {
			continue
		}
		seen[key] = true
		checked = append(checked, clean)
		if _, statErr := os.Stat(clean); statErr != nil {
			continue
		}
		dll, err = syscall.LoadDLL(clean)
		if err == nil {
			break
		}
	}
	if dll == nil {
		if len(checked) == 0 {
			return fmt.Errorf("ViGEmClient.dll could not be located")
		}
		return fmt.Errorf("ViGEmClient.dll could not be loaded; checked: %s", strings.Join(checked, "; "))
	}
	req := func(n string) (*syscall.Proc, error) { return dll.FindProc(n) }
	if x.alloc, err = req("vigem_alloc"); err != nil {
		return err
	}
	if x.free, err = req("vigem_free"); err != nil {
		return err
	}
	if x.connect, err = req("vigem_connect"); err != nil {
		return err
	}
	if x.disconnect, err = req("vigem_disconnect"); err != nil {
		return err
	}
	if x.targetAlloc, err = req("vigem_target_x360_alloc"); err != nil {
		return err
	}
	if x.targetFree, err = req("vigem_target_free"); err != nil {
		return err
	}
	if x.targetAdd, err = req("vigem_target_add"); err != nil {
		return err
	}
	if x.targetRemove, err = req("vigem_target_remove"); err != nil {
		return err
	}
	if x.targetUpdate, err = req("vigem_target_x360_update"); err != nil {
		return err
	}
	x.dll = dll
	client, _, _ := x.alloc.Call()
	if client == 0 {
		return fmt.Errorf("vigem_alloc failed")
	}
	x.client = client
	r, _, _ := x.connect.Call(client)
	if uint32(r) != 0x20000000 {
		x.cleanupLocked()
		return fmt.Errorf("ViGEmBus connect failed (0x%X)", uint32(r))
	}
	target, _, _ := x.targetAlloc.Call()
	if target == 0 {
		x.cleanupLocked()
		return fmt.Errorf("could not create virtual Xbox controller")
	}
	x.target = target
	r, _, _ = x.targetAdd.Call(client, target)
	if uint32(r) != 0x20000000 {
		x.cleanupLocked()
		return fmt.Errorf("could not add virtual Xbox controller (0x%X)", uint32(r))
	}
	x.ready = true
	neutral := xusbReport{}
	_, _, _ = x.targetUpdate.Call(x.client, x.target, uintptr(unsafe.Pointer(&neutral)))
	return nil
}
func (x *xboxBridge) Update(r xusbReport) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if !x.ready {
		return fmt.Errorf("Xbox not ready")
	}
	rc, _, _ := x.targetUpdate.Call(x.client, x.target, uintptr(unsafe.Pointer(&r)))
	if uint32(rc) != 0x20000000 {
		return fmt.Errorf("update failed 0x%X", uint32(rc))
	}
	return nil
}
func (x *xboxBridge) Stop() { x.mu.Lock(); defer x.mu.Unlock(); x.cleanupLocked() }
func (x *xboxBridge) cleanupLocked() {
	if x.target != 0 && x.client != 0 && x.targetRemove != nil {
		x.targetRemove.Call(x.client, x.target)
	}
	if x.target != 0 && x.targetFree != nil {
		x.targetFree.Call(x.target)
	}
	if x.client != 0 && x.disconnect != nil {
		x.disconnect.Call(x.client)
	}
	if x.client != 0 && x.free != nil {
		x.free.Call(x.client)
	}
	if x.dll != nil {
		_ = x.dll.Release()
	}
	x.client = 0
	x.target = 0
	x.ready = false
	x.dll = nil
}

func wndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case WM_CREATE:
		app.hwnd = hwnd
		app.createUI()
		return 0
	case WM_COMMAND:
		app.onCommand(loword(wParam), hiword(wParam))
		return 0
	case WM_SIZE:
		app.updateScrollbars()
		return 0
	case WM_VSCROLL:
		app.handleScroll(false, wParam)
		return 0
	case WM_HSCROLL:
		app.handleScroll(true, wParam)
		return 0
	case WM_TIMER:
		app.refreshMonitor()
		app.refreshDirectInputMonitor()
		app.updateHoverTooltip()
		return 0
	case WM_APP_STATUS:
		app.drainStatus()
		return 0
	case WM_APP_DETECT:
		app.drainDetect()
		return 0
	case WM_APP_LEARN:
		app.drainLearn()
		return 0
	case WM_APP_XBOX:
		app.refreshXboxUI()
		return 0
	case WM_CLOSE:
		procDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		procKillTimer.Call(hwnd, 1)
		app.stopOnce.Do(func() { close(app.stopCh) })
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func (a *appState) createControl(class, text string, style uintptr, x, y, w, h, id int) uintptr {
	ctrl, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(utf16ptr(class))), uintptr(unsafe.Pointer(utf16ptr(text))), style, uintptr(x), uintptr(y), uintptr(w), uintptr(h), a.hwnd, uintptr(id), 0, 0)
	if ctrl != 0 && a.font != 0 {
		procSendMessageW.Call(ctrl, WM_SETFONT, a.font, 1)
	}
	if id != 0 {
		a.controls[id] = ctrl
	}
	if ctrl != 0 && a.baseRects != nil {
		a.baseRects[ctrl] = rect{Left: int32(x), Top: int32(y), Right: int32(x + w), Bottom: int32(y + h)}
	}
	return ctrl
}
func (a *appState) createLabel(text string, x, y, w, h int) uintptr {
	return a.createControl("STATIC", text, WS_CHILD|WS_VISIBLE|SS_LEFT, x, y, w, h, 0)
}
func (a *appState) createButton(text string, x, y, w, h, id int) uintptr {
	return a.createControl("BUTTON", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, x, y, w, h, id)
}
func (a *appState) createCheck(text string, x, y, w, h, id int) uintptr {
	return a.createControl("BUTTON", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, x, y, w, h, id)
}
func (a *appState) createCombo(x, y, w, h, id int) uintptr {
	return a.createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, x, y, w, h, id)
}
func (a *appState) createEdit(text string, x, y, w, h, id int) uintptr {
	return a.createControl("EDIT", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_NUMBER, x, y, w, h, id)
}

func (a *appState) initTooltips() {
	if a.tooltipHwnd != 0 {
		return
	}
	h, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(utf16ptr("tooltips_class32"))), 0, WS_POPUP|TTS_ALWAYSTIP|TTS_NOPREFIX, CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, a.hwnd, 0, 0, 0)
	a.tooltipHwnd = h
	if h != 0 {
		procSendMessageW.Call(h, TTM_SETMAXTIPWIDTH, 0, 420)
	}
}

func (a *appState) addTooltip(control uintptr, text string) {
	if control == 0 || text == "" {
		return
	}
	if a.tooltipHelp == nil {
		a.tooltipHelp = make(map[uintptr]string)
	}
	a.tooltipHelp[control] = text
	a.initTooltips()
	if a.tooltipHwnd == 0 {
		return
	}
	p := utf16ptr(text)
	a.tooltipTexts = append(a.tooltipTexts, p)
	ti := toolInfo{CbSize: uint32(unsafe.Sizeof(toolInfo{})), UFlags: TTF_IDISHWND | TTF_SUBCLASS, Hwnd: a.hwnd, UId: control, LpszText: p}
	procSendMessageW.Call(a.tooltipHwnd, TTM_ADDTOOLW, 0, uintptr(unsafe.Pointer(&ti)))
}

func (a *appState) tooltipForID(id int, text string) { a.addTooltip(a.controls[id], text) }

// updateHoverTooltip is a reliable fallback for systems where the native common-controls
// tooltip bubble does not render. It checks the mouse against the controls that have help text
// and shows a small non-activating popup beside the pointer.
func (a *appState) updateHoverTooltip() {
	if len(a.tooltipHelp) == 0 || a.hwnd == 0 {
		return
	}
	var pt point
	if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		return
	}
	var hovered uintptr
	var help string
	for h, text := range a.tooltipHelp {
		if vis, _, _ := procIsWindowVisible.Call(h); vis == 0 {
			continue
		}
		var rc rect
		if r, _, _ := procGetWindowRect.Call(h, uintptr(unsafe.Pointer(&rc))); r == 0 {
			continue
		}
		if pt.X >= rc.Left && pt.X < rc.Right && pt.Y >= rc.Top && pt.Y < rc.Bottom {
			hovered, help = h, text
			break
		}
	}
	if hovered == 0 {
		if a.tooltipPopup != 0 {
			procShowWindow.Call(a.tooltipPopup, SW_HIDE)
		}
		a.tooltipHover = 0
		return
	}
	if a.tooltipPopup == 0 {
		ex := uintptr(WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE | WS_EX_TRANSPARENT)
		popup, _, _ := procCreateWindowExW.Call(
			ex,
			uintptr(unsafe.Pointer(utf16ptr("STATIC"))),
			uintptr(unsafe.Pointer(utf16ptr(""))),
			WS_POPUP|WS_BORDER|SS_LEFT,
			0, 0, 430, 58,
			a.hwnd, 0, 0, 0,
		)
		a.tooltipPopup = popup
		if popup != 0 && a.font != 0 {
			procSendMessageW.Call(popup, WM_SETFONT, a.font, 1)
		}
	}
	if a.tooltipPopup == 0 {
		return
	}
	if hovered != a.tooltipHover {
		procSetWindowTextW.Call(a.tooltipPopup, uintptr(unsafe.Pointer(utf16ptr("  "+help))))
		a.tooltipHover = hovered
	}
	x, y := int(pt.X)+18, int(pt.Y)+22
	// Keep the tooltip away from the far right/bottom edge on common desktop sizes.
	if x > 1450 {
		x = int(pt.X) - 448
	}
	if y > 900 {
		y = int(pt.Y) - 80
	}
	procSetWindowPos.Call(a.tooltipPopup, ^uintptr(0), uintptr(x), uintptr(y), 430, 58, SWP_NOACTIVATE|SWP_SHOWWINDOW)
	procShowWindow.Call(a.tooltipPopup, SW_SHOWNOACTIVATE)
}

func (a *appState) updateScrollbars() {
	if a.hwnd == 0 {
		return
	}
	var rc rect
	if ok, _, _ := procGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&rc))); ok == 0 {
		return
	}
	cw := int(rc.Right - rc.Left)
	ch := int(rc.Bottom - rc.Top)
	maxX := a.contentWidth - cw
	maxY := a.contentHeight - ch
	if maxX < 0 {
		maxX = 0
	}
	if maxY < 0 {
		maxY = 0
	}
	if a.scrollX > maxX {
		a.scrollX = maxX
	}
	if a.scrollY > maxY {
		a.scrollY = maxY
	}
	procSetScrollRange.Call(a.hwnd, SB_HORZ, 0, uintptr(maxX), 1)
	procSetScrollRange.Call(a.hwnd, SB_VERT, 0, uintptr(maxY), 1)
	procSetScrollPos.Call(a.hwnd, SB_HORZ, uintptr(a.scrollX), 1)
	procSetScrollPos.Call(a.hwnd, SB_VERT, uintptr(a.scrollY), 1)
	showH, showV := uintptr(0), uintptr(0)
	if maxX > 0 {
		showH = 1
	}
	if maxY > 0 {
		showV = 1
	}
	procShowScrollBar.Call(a.hwnd, SB_HORZ, showH)
	procShowScrollBar.Call(a.hwnd, SB_VERT, showV)
	a.repositionScrolledChildren()
}

func (a *appState) repositionScrolledChildren() {
	for h, r := range a.baseRects {
		if h == 0 {
			continue
		}
		x := int(r.Left) - a.scrollX
		y := int(r.Top) - a.scrollY
		procSetWindowPos.Call(h, 0, uintptr(x), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
	}
}

func (a *appState) handleScroll(horizontal bool, wParam uintptr) {
	code := loword(wParam)
	thumb := hiword(wParam)
	var rc rect
	procGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&rc)))
	page := int(rc.Bottom-rc.Top) - 40
	pos := a.scrollY
	limit := a.contentHeight - int(rc.Bottom-rc.Top)
	bar := uintptr(SB_VERT)
	if horizontal {
		page = int(rc.Right-rc.Left) - 40
		pos = a.scrollX
		limit = a.contentWidth - int(rc.Right-rc.Left)
		bar = SB_HORZ
	}
	if page < 40 {
		page = 40
	}
	if limit < 0 {
		limit = 0
	}
	switch code {
	case SB_LINEUP:
		pos -= 30
	case SB_LINEDOWN:
		pos += 30
	case SB_PAGEUP:
		pos -= page
	case SB_PAGEDOWN:
		pos += page
	case SB_THUMBPOSITION, SB_THUMBTRACK:
		pos = thumb
	case SB_TOP:
		pos = 0
	case SB_BOTTOM:
		pos = limit
	}
	if pos < 0 {
		pos = 0
	}
	if pos > limit {
		pos = limit
	}
	if horizontal {
		a.scrollX = pos
	} else {
		a.scrollY = pos
	}
	procSetScrollPos.Call(a.hwnd, bar, uintptr(pos), 1)
	a.repositionScrolledChildren()
}

func hidhidePaths() (cli, client string, installed bool) {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		return "", "", false
	}
	root := filepath.Join(pf, "Nefarius Software Solutions", "HidHide")
	candidatesCLI := []string{filepath.Join(root, "HidHideCLI.exe"), filepath.Join(root, "x64", "HidHideCLI.exe")}
	candidatesClient := []string{filepath.Join(root, "HidHideClient.exe"), filepath.Join(root, "x64", "HidHideClient.exe")}
	for _, p := range candidatesCLI {
		if _, err := os.Stat(p); err == nil {
			cli = p
			break
		}
	}
	for _, p := range candidatesClient {
		if _, err := os.Stat(p); err == nil {
			client = p
			break
		}
	}
	return cli, client, cli != "" && client != ""
}

func (a *appState) refreshHidHideUI() {
	h := a.controls[idHidHideStatus]
	if h == 0 {
		return
	}
	_, _, ok := hidhidePaths()
	if ok {
		setText(h, "HidHide: installed (optional)")
	} else {
		setText(h, "HidHide: not installed (optional)")
	}
}

func (a *appState) openHidHideSetup() {
	cli, client, ok := hidhidePaths()
	if !ok {
		a.postStatus("HidHide is not installed. Opening the official HidHide releases page. It is optional and used to prevent double input.")
		procShellExecuteW.Call(0, uintptr(unsafe.Pointer(utf16ptr("open"))), uintptr(unsafe.Pointer(utf16ptr("https://github.com/nefarius/HidHide/releases"))), 0, 0, SW_SHOW)
		return
	}
	exePath, err := os.Executable()
	if err != nil {
		exePath = ""
	}
	go func() {
		registered := false
		if exePath != "" {
			cmd := exec.Command(cli, "--inv-off", "--app-reg", exePath)
			if out, err := cmd.CombinedOutput(); err == nil {
				registered = true
			} else if len(out) > 0 {
				a.postStatus("HidHide is installed, but automatic app registration was busy/blocked. The configuration client will still open; add this EXE on the Applications tab if needed.")
			}
		}
		if err := exec.Command(client).Start(); err != nil {
			a.postStatus("Could not open HidHide Configuration Client: " + err.Error())
			return
		}
		if registered {
			a.postStatus("HidHide opened and this mapper was added to its application allowlist. Choose the physical flight devices to hide, enable device hiding, then reconnect them.")
		} else {
			a.postStatus("HidHide opened. Add this mapper on Applications, select the physical flight devices on Devices, enable hiding, then reconnect them.")
		}
	}()
}

func (a *appState) setGroupVisible(group []uintptr, visible bool) {
	cmd := uintptr(SW_HIDE)
	if visible {
		cmd = SW_SHOW
	}
	for _, h := range group {
		if h != 0 {
			procShowWindow.Call(h, cmd)
		}
	}
}

func (a *appState) updateContextVisibility() {
	if !a.uiReady {
		return
	}
	controlModes := []string{"HOSAS", "HOTAS"}
	ci := comboIndex(idControlMode)
	if ci < 0 || ci >= len(controlModes) {
		ci = 0
	}
	cm := controlModes[ci]
	a.setGroupVisible(a.controlHOSASGroup, strings.EqualFold(cm, "HOSAS"))
	a.setGroupVisible(a.controlHOTASGroup, strings.EqualFold(cm, "HOTAS"))
	a.setGroupVisible(a.hosasDrivingGroup, strings.EqualFold(cm, "HOSAS"))
	pedalModes := []string{"Off", "Fine steering", "Move Camera", "Keyboard keys"}
	pi := comboIndex(idPedalMode)
	if pi < 0 || pi >= len(pedalModes) {
		pi = 0
	}
	pm := pedalModes[pi]
	a.setGroupVisible(a.pedalFineGroup, strings.EqualFold(pm, "Fine steering"))
	a.setGroupVisible(a.pedalCameraGroup, strings.EqualFold(pm, "Move Camera"))
	a.setGroupVisible(a.pedalKeyGroup, strings.EqualFold(pm, "Keyboard keys"))

	onFootModes := []string{"Off", "Thumbstick -> WASD", "Buttons -> WASD"}
	oi := comboIndex(idOnFootMode)
	if oi < 0 || oi >= len(onFootModes) {
		oi = 0
	}
	om := onFootModes[oi]
	a.setGroupVisible(a.onFootThumbGroup, strings.EqualFold(om, "Thumbstick -> WASD"))
	a.setGroupVisible(a.onFootButtonGroup, strings.EqualFold(om, "Buttons -> WASD"))
}

func (a *appState) createUI() {
	f, _, _ := procGetStockObject.Call(DEFAULT_GUI_FONT)
	a.font = f

	a.createLabel("HOSAS / HOTAS -> Xbox 360 bridge for Galactic Racer", 20, 15, 650, 24)
	a.createLabel("by JaffaQuake and Mars", 820, 15, 260, 24)
	a.createLabel("Control mode", 830, 50, 95, 22)
	a.createCombo(925, 44, 185, 160, idControlMode)
	a.createLabel("Input API", 830, 80, 90, 22)
	a.createCombo(925, 74, 185, 140, idInputBackend)
	a.createButton("Refresh devices", 830, 104, 120, 28, idRefresh)
	a.createButton("HidHide Setup", 960, 104, 150, 28, idHidHideSetup)
	a.createControl("STATIC", "HidHide: checking...", WS_CHILD|WS_VISIBLE|SS_LEFT, 830, 134, 280, 18, idHidHideStatus)

	// HOSAS controls. Hidden automatically when HOTAS is selected.
	a.controlHOSASGroup = append(a.controlHOSASGroup,
		a.createLabel("HOSAS sticks", 20, 48, 160, 20),
		a.createLabel("Device", 155, 48, 240, 20),
		a.createLabel("Forward axis", 405, 48, 110, 20),
		a.createLabel("Left stick", 20, 76, 120, 24),
		a.createCombo(150, 72, 240, 260, idLeftDevice),
		a.createCombo(405, 72, 85, 220, idLeftAxis),
		a.createCheck("Invert", 500, 73, 75, 24, idLeftInvert),
		a.createCheck("Throttle axis", 580, 73, 115, 24, idLeftThrottleMode),
		a.createButton("Detect Left", 705, 70, 110, 28, idDetectLeft),
		a.createLabel("Right stick", 20, 110, 120, 24),
		a.createCombo(150, 106, 240, 260, idRightDevice),
		a.createCombo(405, 106, 85, 220, idRightAxis),
		a.createCheck("Invert", 500, 107, 75, 24, idRightInvert),
		a.createCheck("Throttle axis", 580, 107, 115, 24, idRightThrottleMode),
		a.createButton("Detect Right", 705, 104, 110, 28, idDetectRight))

	// HOTAS controls occupy the same space as the HOSAS rows. Throttle is RT-only;
	// the selected flight stick maps directly to the virtual Xbox left thumbstick.
	a.controlHOTASGroup = append(a.controlHOTASGroup,
		a.createLabel("HOTAS", 20, 48, 100, 20),
		a.createLabel("Device", 155, 48, 220, 20),
		a.createLabel("Throttle / X axis", 405, 48, 100, 20),
		a.createLabel("Y axis", 500, 48, 70, 20),
		a.createLabel("Throttle", 20, 76, 120, 24),
		a.createCombo(150, 72, 240, 260, idHotasThrottleDevice),
		a.createCombo(405, 72, 85, 220, idHotasThrottleAxis),
		a.createCheck("Invert", 500, 73, 75, 24, idHotasThrottleInvert),
		a.createLabel("RT / forward only", 585, 76, 115, 22),
		a.createLabel("Flight stick", 20, 110, 120, 24),
		a.createCombo(150, 106, 240, 260, idHotasStickDevice),
		a.createCombo(405, 106, 85, 220, idHotasStickXAxis),
		a.createCombo(500, 106, 85, 220, idHotasStickYAxis),
		a.createCheck("Invert X", 590, 107, 78, 24, idHotasStickInvertX),
		a.createCheck("Invert Y", 670, 107, 78, 24, idHotasStickInvertY))

	a.createLabel("Rudder pedals", 20, 154, 130, 20)
	a.createLabel("Pedal device", 20, 182, 115, 24)
	a.createCombo(150, 178, 240, 260, idPedalDevice)
	a.createCombo(405, 178, 85, 220, idPedalAxis)
	a.createCheck("Invert", 500, 179, 75, 24, idPedalInvert)
	a.createLabel("Pedal use", 585, 182, 80, 20)
	a.createCombo(665, 178, 170, 180, idPedalMode)

	// Pedal-specific settings share one row and are shown only for the selected pedal mode.
	a.pedalFineGroup = append(a.pedalFineGroup,
		a.createLabel("Fine-steer authority", 20, 216, 130, 24),
		a.createEdit("30", 150, 212, 55, 25, idPedalStrength),
		a.createLabel("%", 210, 216, 25, 22))
	a.pedalCameraGroup = append(a.pedalCameraGroup,
		a.createLabel("Camera sensitivity", 20, 216, 120, 24),
		a.createEdit("100", 145, 212, 55, 25, idCameraSensitivity),
		a.createLabel("%", 205, 216, 25, 22),
		a.createLabel("Camera deadzone", 285, 216, 115, 24),
		a.createEdit("6", 405, 212, 55, 25, idCameraDeadzone),
		a.createLabel("%", 465, 216, 25, 22))
	a.pedalKeyGroup = append(a.pedalKeyGroup,
		a.createLabel("Pedal left key", 20, 216, 110, 24),
		a.createCombo(130, 212, 145, 260, idPedalLeftKey),
		a.createLabel("Pedal right key", 295, 216, 115, 24),
		a.createCombo(410, 212, 145, 260, idPedalRightKey),
		a.createLabel("Threshold", 575, 216, 75, 24),
		a.createEdit("20", 650, 212, 55, 25, idPedalKeyThreshold),
		a.createLabel("%", 710, 216, 25, 22))

	// On-foot controls can use either an analog thumbstick (DirectInput) or learned buttons/hats.
	a.createLabel("On-foot controls", 20, 264, 120, 22)
	a.createLabel("Mode", 145, 264, 45, 22)
	a.createCombo(190, 258, 190, 180, idOnFootMode)

	// Thumbstick -> WASD contextual controls.
	a.onFootThumbGroup = append(a.onFootThumbGroup,
		a.createLabel("Threshold", 400, 264, 70, 20),
		a.createEdit("20", 472, 258, 45, 25, idWASDThreshold),
		a.createLabel("%", 520, 264, 20, 20),
		a.createCheck("Invert side", 555, 260, 92, 24, idWASDInvertSide),
		a.createCheck("Invert fwd", 655, 260, 90, 24, idWASDInvertForward),
		a.createButton("Detect Thumbstick", 765, 256, 145, 30, idDetectWASD))
	diMon := a.createControl("STATIC", "DirectInput axis monitor: waiting for game-controller data...", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 300, 1100, 52, idDIThumbMonitor)
	a.onFootThumbGroup = append(a.onFootThumbGroup, diMon)

	// Buttons -> WASD contextual controls. Each direction learns a normal VKB button/hat/switch.
	for i, def := range wasdButtonDefs {
		x := 20 + i*275
		indicator := a.createControl("BUTTON", def.Label, WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, x, 300, 55, 28, idWASDIndicatorBase+i)
		label := a.createControl("STATIC", "Not assigned", WS_CHILD|WS_VISIBLE|SS_LEFT, x+62, 305, 138, 22, idWASDLabelBase+i)
		learn := a.createButton("Learn", x+205, 300, 60, 28, idWASDLearnBase+i)
		a.onFootButtonGroup = append(a.onFootButtonGroup, indicator, label, learn)
	}

	a.createLabel("Driving", 20, 392, 100, 20)
	a.createLabel("Deadzone", 20, 420, 80, 22)
	a.createEdit("6", 100, 416, 50, 25, idDeadzone)
	a.createLabel("%", 155, 420, 25, 22)
	a.hosasDrivingGroup = append(a.hosasDrivingGroup,
		a.createCheck("Hybrid preserve thrust", 200, 416, 190, 26, idPreserve),
		a.createLabel("Forward steering keeps thrust; pulling one stick behind center progressively scrubs it.", 400, 420, 555, 30))

	a.createButton("Start mapping", 20, 460, 125, 32, idStart)
	a.createButton("Test Xbox A", 155, 460, 110, 32, idTestA)
	a.createButton("Camera Left", 275, 460, 110, 32, idTestCamLeft)
	a.createButton("Camera Right", 395, 460, 110, 32, idTestCamRight)
	a.createButton("Save settings", 515, 460, 115, 32, idSave)
	a.createLabel("Polling rate", 660, 467, 85, 22)
	a.createCombo(745, 460, 105, 180, idPollingRate)
	a.createLabel("Hz", 855, 467, 30, 22)

	a.createLabel("LIVE INPUT MONITOR", 20, 508, 170, 20)
	a.createControl("STATIC", "Waiting for devices...", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 533, 545, 56, idMonitor)
	a.createControl("STATIC", "Virtual Xbox disconnected. Click Start mapping to create it.", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 593, 545, 45, idStatus)
	a.createControl("STATIC", "XInput readback: disconnected", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 643, 545, 28, idXInputReadback)

	const boxSize = 126
	leftBoxX, boxY := 600, 542
	rightBoxX := 770
	outputBoxX := 940
	a.createLabel("LEFT STICK", leftBoxX+25, 506, 95, 20)
	a.createLabel("RIGHT STICK", rightBoxX+18, 506, 105, 20)
	a.createLabel("DRIVE / STEER", outputBoxX+9, 506, 120, 20)
	for _, x := range []int{leftBoxX, rightBoxX, outputBoxX} {
		a.createControl("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, x, boxY, boxSize, boxSize, 0)
		a.createControl("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, x+boxSize/2, boxY, 1, boxSize, 0)
		a.createControl("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, x, boxY+boxSize/2, boxSize, 1, 0)
		a.createLabel("F", x+boxSize/2-4, boxY-18, 15, 18)
		a.createLabel("B", x+boxSize/2-4, boxY+boxSize+2, 15, 18)
		a.createLabel("L", x-14, boxY+boxSize/2-8, 14, 18)
		a.createLabel("R", x+boxSize+3, boxY+boxSize/2-8, 14, 18)
	}
	a.createControl("BUTTON", "", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, leftBoxX+boxSize/2-6, boxY+boxSize/2-6, 12, 12, idLeftDot)
	a.createControl("BUTTON", "", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, rightBoxX+boxSize/2-6, boxY+boxSize/2-6, 12, 12, idRightDot)
	a.createControl("BUTTON", "", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, outputBoxX+boxSize/2-6, boxY+boxSize/2-6, 12, 12, idDriveSteerDot)
	a.createControl("STATIC", "Side +0.00  F +0.00", WS_CHILD|WS_VISIBLE|SS_LEFT, leftBoxX, boxY+boxSize+22, 155, 20, idLeftVectorText)
	a.createControl("STATIC", "Side +0.00  F +0.00", WS_CHILD|WS_VISIBLE|SS_LEFT, rightBoxX, boxY+boxSize+22, 155, 20, idRightVectorText)
	a.createControl("STATIC", "Drive +0.00 Neutral\r\nSteer +0.00 Straight", WS_CHILD|WS_VISIBLE|SS_LEFT, outputBoxX-5, boxY+boxSize+20, 185, 38, idDriveSteerVectorText)

	a.createLabel("Button / switch mapping - Xbox outputs plus native keyboard camera views", 20, 736, 760, 22)
	startY := 766
	rowsPerCol := (len(buttonOutputs) + 1) / 2
	for i, bo := range buttonOutputs {
		col := i / rowsPerCol
		row := i % rowsPerCol
		x := 20 + col*535
		y := startY + row*37
		a.createControl("BUTTON", bo.Label, WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, x, y, 118, 28, idLearnBase+200+i)
		a.createControl("STATIC", "Not assigned", WS_CHILD|WS_VISIBLE|SS_LEFT, x+125, y+5, 255, 22, idLearnBase+100+i)
		a.createButton("Learn", x+390, y, 80, 28, idLearnBase+i)
	}

	// Hover help.
	a.tooltipForID(idControlMode, "HOSAS / Podracer Style combines two forward/back stick axes into thrust and differential steering. HOTAS / Traditional Steering uses a dedicated throttle for RT/forward and a flight stick as the Xbox left thumbstick.")
	a.tooltipForID(idInputBackend, "Axis input backend. Auto uses DirectInput first and falls back to WinMM when possible. DirectInput exposes X/Y/Z, Rx/Ry/Rz and Slider 1/2. WinMM keeps the known-good legacy six-axis path. Learned buttons/hats remain on the stable WinMM path in this beta.")
	a.tooltipForID(idHidHideSetup, "Optional double-input protection. If HidHide is installed, this registers this mapper on HidHide's allowlist and opens the official configuration client so you can hide the physical controllers from games. v1.92 Beta does not automatically choose devices to hide.")
	a.tooltipForID(idHotasThrottleDevice, "HOTAS throttle device. Its selected axis is converted to Xbox Right Trigger from 0% to 100% forward thrust only.")
	a.tooltipForID(idHotasThrottleAxis, "Physical throttle axis. Full aft maps to RT 0%; full forward maps to RT 100%. Use Invert if the direction is backwards.")
	a.tooltipForID(idHotasStickDevice, "HOTAS flight stick device. Its selected X/Y axes map directly to the Xbox left thumbstick.")
	a.tooltipForID(idHotasStickXAxis, "Physical flight-stick horizontal axis mapped to Xbox Left Stick X.")
	a.tooltipForID(idHotasStickYAxis, "Physical flight-stick vertical axis mapped to Xbox Left Stick Y.")
	a.tooltipForID(idDetectLeft, "Move the LEFT flight stick forward after clicking. The app identifies the device, forward/back axis, and inversion automatically.")
	a.tooltipForID(idDetectRight, "Move the RIGHT flight stick forward after clicking. The app identifies the device, forward/back axis, and inversion automatically.")
	a.tooltipForID(idLeftThrottleMode, "Treat the selected LEFT HOSAS axis as a throttle lever: 0% = -1, 50% = neutral (0), 100% = +1. Useful for non-spring-centered throttle hardware.")
	a.tooltipForID(idRightThrottleMode, "Treat the selected RIGHT HOSAS axis as a throttle lever: 0% = -1, 50% = neutral (0), 100% = +1. Useful for non-spring-centered throttle hardware.")
	a.tooltipForID(idRefresh, "Rescan Windows for connected controller devices and restart the DirectInput device list.")
	a.tooltipForID(idDetectWASD, "DirectInput thumbstick detection: push the small stick RIGHT, return to center, then push it FORWARD. This reads the same X/Y Rotation path that joy.cpl uses.")
	a.tooltipForID(idOnFootMode, "Choose how on-foot W/A/S/D is produced: analog thumbstick, learned stick buttons/hats, or off.")
	a.tooltipForID(idStart, "Create the virtual Xbox controller and begin sending HOSAS, keyboard, pedal, and learned-button outputs to the game. Stop Mapping removes the virtual controller.")
	a.tooltipForID(idTestA, "Send a short Xbox A-button pulse through the virtual controller. Start Mapping must be active.")
	a.tooltipForID(idTestCamLeft, "Force virtual Xbox Right Stick X fully left briefly. Useful for camera-output diagnostics.")
	a.tooltipForID(idTestCamRight, "Force virtual Xbox Right Stick X fully right briefly. Useful for camera-output diagnostics.")
	a.tooltipForID(idSave, "Save the current device, axis, deadzone, polling-rate, pedal, on-foot, and learned-button settings.")
	a.tooltipForID(idPreserve, "Hybrid thrust: forward-stick steering keeps full thrust; pulling one stick behind center progressively scrubs thrust.")
	a.tooltipForID(idPollingRate, "How often the controller loop requests fresh input/output updates. Higher rates reduce interval but use more CPU; hardware/Windows update rates may still be lower.")
	a.tooltipForID(idPedalMode, "Choose how the rudder axis is used. Only settings relevant to the selected mode are shown.")
	a.tooltipForID(idPedalLeftKey, "Keyboard key held when the pedal axis moves left past the configured threshold.")
	a.tooltipForID(idPedalRightKey, "Keyboard key held when the pedal axis moves right past the configured threshold.")
	for i, bo := range buttonOutputs {
		a.tooltipForID(idLearnBase+i, "Click Learn, then press the physical VKB button/hat/switch you want assigned to "+bo.Label+".")
		a.tooltipForID(idLearnBase+200+i, "Live indicator for "+bo.Label+". It highlights when the learned physical input is pressed while the visual monitor is active.")
	}
	for i, def := range wasdButtonDefs {
		a.tooltipForID(idWASDLearnBase+i, "Click Learn, then press the physical VKB button/hat/switch that should generate "+def.Label+" for on-foot movement.")
		a.tooltipForID(idWASDIndicatorBase+i, "Live indicator for the learned "+def.Label+" movement input.")
	}

	a.populateCombos()
	a.applyConfigToUI()
	a.updateBindingLabels()
	a.uiReady = true
	a.updateContextVisibility()
	procSetTimer.Call(a.hwnd, 1, 100, 0)
}

func (a *appState) populateCombos() {
	a.cfgMu.RLock()
	backend := normalizeInputBackend(a.cfg.InputBackend)
	a.cfgMu.RUnlock()

	deviceIDs := []int{idLeftDevice, idRightDevice, idPedalDevice, idHotasThrottleDevice, idHotasStickDevice}
	if backend == "WinMM" {
		a.devicesMu.RLock()
		devs := append([]deviceInfo(nil), a.devices...)
		a.devicesMu.RUnlock()
		for _, id := range deviceIDs {
			h := a.controls[id]
			procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
			ids := []int{}
			guids := []string{}
			if id == idPedalDevice || id == idHotasThrottleDevice || id == idHotasStickDevice {
				procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr("None"))))
				ids = append(ids, -1)
				guids = append(guids, "")
			}
			for _, d := range devs {
				txt := fmt.Sprintf("%s [WinMM ID %d]", d.Name, d.ID)
				procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(txt))))
				ids = append(ids, d.ID)
				guids = append(guids, "")
			}
			a.deviceComboIDs[id] = ids
			a.deviceComboGUIDs[id] = guids
		}
	} else {
		devs := a.di.DeviceList()
		for _, id := range deviceIDs {
			h := a.controls[id]
			procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
			ids := []int{-1}
			guids := []string{""}
			procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr("None"))))
			for _, d := range devs {
				txt := fmt.Sprintf("%s [DirectInput]", d.Name)
				procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(txt))))
				ids = append(ids, a.matchWinMMIDByName(d.Name))
				guids = append(guids, d.GUID)
			}
			a.deviceComboIDs[id] = ids
			a.deviceComboGUIDs[id] = guids
		}
	}

	axisList := a.axisNames
	if backend != "WinMM" {
		axisList = diAxisNames
	}
	for _, id := range []int{idLeftAxis, idRightAxis, idPedalAxis, idHotasThrottleAxis, idHotasStickXAxis, idHotasStickYAxis} {
		h := a.controls[id]
		procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
		for _, name := range axisList {
			procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(name))))
		}
	}

	h := a.controls[idInputBackend]
	procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
	for _, mode := range []string{"Auto (DirectInput first)", "DirectInput", "WinMM (compatibility)"} {
		procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(mode))))
	}

	h = a.controls[idControlMode]
	procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
	for _, mode := range []string{"HOSAS / Podracer Style", "HOTAS / Traditional Steering"} {
		procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(mode))))
	}

	h = a.controls[idPedalMode]
	procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
	for _, mode := range []string{"Off", "Fine steering", "Move Camera", "Keyboard keys"} {
		procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(mode))))
	}

	h = a.controls[idOnFootMode]
	procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
	for _, mode := range []string{"Off", "Thumbstick -> WASD", "Buttons -> WASD"} {
		procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(mode))))
	}

	hp := a.controls[idPollingRate]
	procSendMessageW.Call(hp, CB_RESETCONTENT, 0, 0)
	for _, hz := range pollingRateOptions {
		procSendMessageW.Call(hp, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(fmt.Sprintf("%d", hz)))))
	}
	for _, id := range []int{idPedalLeftKey, idPedalRightKey} {
		hk := a.controls[id]
		procSendMessageW.Call(hk, CB_RESETCONTENT, 0, 0)
		for _, k := range keyboardKeyOptions {
			procSendMessageW.Call(hk, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(k.Name))))
		}
	}
}

func (a *appState) applyConfigToUI() {
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()
	backendIdx := 0
	switch normalizeInputBackend(c.InputBackend) {
	case "DirectInput":
		backendIdx = 1
	case "WinMM":
		backendIdx = 2
	}
	comboSetIndex(idInputBackend, backendIdx)
	controlIdx := 0
	if strings.EqualFold(normalizeControlMode(c.ControlMode), "HOTAS") {
		controlIdx = 1
	}
	comboSetIndex(idControlMode, controlIdx)
	if normalizeInputBackend(c.InputBackend) == "WinMM" {
		comboSetDevice(idHotasThrottleDevice, c.HotasThrottleID)
		comboSetDevice(idHotasStickDevice, c.HotasStickID)
		comboSetDevice(idLeftDevice, c.LeftID)
		comboSetDevice(idRightDevice, c.RightID)
		comboSetDevice(idPedalDevice, c.PedalID)
	} else {
		comboSetGUID(idHotasThrottleDevice, c.HotasThrottleDIGUID)
		comboSetGUID(idHotasStickDevice, c.HotasStickDIGUID)
		comboSetGUID(idLeftDevice, c.LeftDIGUID)
		comboSetGUID(idRightDevice, c.RightDIGUID)
		comboSetGUID(idPedalDevice, c.PedalDIGUID)
	}
	comboSetIndex(idHotasThrottleAxis, c.HotasThrottleAxis)
	setChecked(a.controls[idHotasThrottleInvert], c.HotasThrottleInvert)
	comboSetIndex(idHotasStickXAxis, c.HotasStickXAxis)
	comboSetIndex(idHotasStickYAxis, c.HotasStickYAxis)
	setChecked(a.controls[idHotasStickInvertX], c.HotasStickInvertX)
	setChecked(a.controls[idHotasStickInvertY], c.HotasStickInvertY)
	comboSetIndex(idLeftAxis, c.LeftAxis)
	comboSetIndex(idRightAxis, c.RightAxis)
	comboSetIndex(idPedalAxis, c.PedalAxis)
	setChecked(a.controls[idLeftInvert], c.LeftInvert)
	setChecked(a.controls[idRightInvert], c.RightInvert)
	setChecked(a.controls[idLeftThrottleMode], c.LeftThrottleMode)
	setChecked(a.controls[idRightThrottleMode], c.RightThrottleMode)
	setChecked(a.controls[idPedalInvert], c.PedalInvert)
	setChecked(a.controls[idWASDInvertSide], c.ThumbWASDInvertSide)
	setChecked(a.controls[idWASDInvertForward], c.ThumbWASDInvertForward)
	modeIdx := 0
	if strings.EqualFold(c.PedalMode, "Fine steering") {
		modeIdx = 1
	} else if strings.EqualFold(c.PedalMode, "Move Camera") {
		modeIdx = 2
	} else if strings.EqualFold(c.PedalMode, "Keyboard keys") {
		modeIdx = 3
	}
	comboSetIndex(idPedalMode, modeIdx)
	onFootIdx := 0
	switch normalizeOnFootMode(c.OnFootMode) {
	case "Thumbstick -> WASD":
		onFootIdx = 1
	case "Buttons -> WASD":
		onFootIdx = 2
	}
	comboSetIndex(idOnFootMode, onFootIdx)
	comboSetIndex(idPedalLeftKey, keyOptionIndex(c.PedalLeftKey))
	comboSetIndex(idPedalRightKey, keyOptionIndex(c.PedalRightKey))
	setText(a.controls[idDeadzone], strconv.Itoa(c.Deadzone))
	setText(a.controls[idPedalStrength], strconv.Itoa(c.PedalSteerStrength))
	setText(a.controls[idCameraSensitivity], strconv.Itoa(c.CameraSensitivity))
	setText(a.controls[idCameraDeadzone], strconv.Itoa(c.CameraDeadzone))
	setText(a.controls[idPedalKeyThreshold], strconv.Itoa(c.PedalKeyThreshold))
	setText(a.controls[idWASDThreshold], strconv.Itoa(c.ThumbWASDThreshold))
	pidx := 0
	for i, hz := range pollingRateOptions {
		if hz == c.PollingHz {
			pidx = i
			break
		}
	}
	comboSetIndex(idPollingRate, pidx)
	setChecked(a.controls[idPreserve], c.PreserveThrust)
	if a.uiReady {
		a.updateContextVisibility()
	}
}

func comboSetIndex(id, idx int) {
	if idx < 0 {
		idx = 0
	}
	procSendMessageW.Call(app.controls[id], CB_SETCURSEL, uintptr(idx), 0)
}
func comboSetDevice(id, deviceID int) {
	ids := app.deviceComboIDs[id]
	idx := 0
	for i, v := range ids {
		if v == deviceID {
			idx = i
			break
		}
	}
	procSendMessageW.Call(app.controls[id], CB_SETCURSEL, uintptr(idx), 0)
}
func comboSetGUID(id int, guid string) {
	guids := app.deviceComboGUIDs[id]
	idx := 0
	for i, v := range guids {
		if strings.EqualFold(v, guid) {
			idx = i
			break
		}
	}
	procSendMessageW.Call(app.controls[id], CB_SETCURSEL, uintptr(idx), 0)
}

func comboGUID(id int) string {
	idx := comboIndex(id)
	guids := app.deviceComboGUIDs[id]
	if idx >= 0 && idx < len(guids) {
		return guids[idx]
	}
	return ""
}

func setChecked(h uintptr, v bool) {
	n := uintptr(0)
	if v {
		n = BST_CHECKED
	}
	procSendMessageW.Call(h, BM_SETCHECK, n, 0)
}
func isChecked(h uintptr) bool {
	r, _, _ := procSendMessageW.Call(h, BM_GETCHECK, 0, 0)
	return r == BST_CHECKED
}
func setText(h uintptr, s string) { procSetWindowTextW.Call(h, uintptr(unsafe.Pointer(utf16ptr(s)))) }
func getText(h uintptr) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:int(n)])
}
func comboIndex(id int) int {
	r, _, _ := procSendMessageW.Call(app.controls[id], CB_GETCURSEL, 0, 0)
	if int64(r) < 0 {
		return 0
	}
	return int(r)
}
func comboDevice(id int) int {
	idx := comboIndex(id)
	ids := app.deviceComboIDs[id]
	if idx >= 0 && idx < len(ids) {
		return ids[idx]
	}
	return -1
}

func (a *appState) syncConfigFromUI() {
	if !a.uiReady {
		return
	}
	a.cfgMu.Lock()
	c := a.cfg
	backendModes := []string{"Auto", "DirectInput", "WinMM"}
	bi := comboIndex(idInputBackend)
	if bi < 0 || bi >= len(backendModes) {
		bi = 0
	}
	c.InputBackend = backendModes[bi]
	controlModes := []string{"HOSAS", "HOTAS"}
	ci := comboIndex(idControlMode)
	if ci < 0 || ci >= len(controlModes) {
		ci = 0
	}
	c.ControlMode = controlModes[ci]
	if normalizeInputBackend(c.InputBackend) == "WinMM" {
		c.HotasThrottleID = comboDevice(idHotasThrottleDevice)
		c.HotasStickID = comboDevice(idHotasStickDevice)
		c.LeftID = comboDevice(idLeftDevice)
		c.RightID = comboDevice(idRightDevice)
		c.PedalID = comboDevice(idPedalDevice)
	} else {
		c.HotasThrottleDIGUID = comboGUID(idHotasThrottleDevice)
		c.HotasStickDIGUID = comboGUID(idHotasStickDevice)
		c.LeftDIGUID = comboGUID(idLeftDevice)
		c.RightDIGUID = comboGUID(idRightDevice)
		c.PedalDIGUID = comboGUID(idPedalDevice)
		// Keep a matching WinMM ID when names line up so Auto has a safe legacy fallback.
		c.HotasThrottleID = comboDevice(idHotasThrottleDevice)
		c.HotasStickID = comboDevice(idHotasStickDevice)
		c.LeftID = comboDevice(idLeftDevice)
		c.RightID = comboDevice(idRightDevice)
		c.PedalID = comboDevice(idPedalDevice)
	}
	c.HotasThrottleAxis = comboIndex(idHotasThrottleAxis)
	c.HotasThrottleInvert = isChecked(a.controls[idHotasThrottleInvert])
	c.HotasStickXAxis = comboIndex(idHotasStickXAxis)
	c.HotasStickYAxis = comboIndex(idHotasStickYAxis)
	c.HotasStickInvertX = isChecked(a.controls[idHotasStickInvertX])
	c.HotasStickInvertY = isChecked(a.controls[idHotasStickInvertY])
	c.LeftAxis = comboIndex(idLeftAxis)
	c.RightAxis = comboIndex(idRightAxis)
	c.PedalAxis = comboIndex(idPedalAxis)
	pi := comboIndex(idPollingRate)
	if pi >= 0 && pi < len(pollingRateOptions) {
		c.PollingHz = pollingRateOptions[pi]
	}
	c.LeftInvert = isChecked(a.controls[idLeftInvert])
	c.RightInvert = isChecked(a.controls[idRightInvert])
	c.LeftThrottleMode = isChecked(a.controls[idLeftThrottleMode])
	c.RightThrottleMode = isChecked(a.controls[idRightThrottleMode])
	c.PedalInvert = isChecked(a.controls[idPedalInvert])
	c.ThumbWASDInvertSide = isChecked(a.controls[idWASDInvertSide])
	c.ThumbWASDInvertForward = isChecked(a.controls[idWASDInvertForward])
	modes := []string{"Off", "Fine steering", "Move Camera", "Keyboard keys"}
	mi := comboIndex(idPedalMode)
	if mi < 0 || mi >= len(modes) {
		mi = 0
	}
	c.PedalMode = modes[mi]
	onFootModes := []string{"Off", "Thumbstick -> WASD", "Buttons -> WASD"}
	oi := comboIndex(idOnFootMode)
	if oi < 0 || oi >= len(onFootModes) {
		oi = 0
	}
	c.OnFootMode = onFootModes[oi]
	// Maintain the old boolean field so older builds can still interpret a thumbstick-enabled config.
	c.ThumbWASDEnabled = strings.EqualFold(c.OnFootMode, "Thumbstick -> WASD")
	c.PedalLeftKey = selectedKeyVK(idPedalLeftKey)
	c.PedalRightKey = selectedKeyVK(idPedalRightKey)
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idWASDThreshold]))); e == nil {
		if n < 5 {
			n = 5
		}
		if n > 95 {
			n = 95
		}
		c.ThumbWASDThreshold = n
	}
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idPedalKeyThreshold]))); e == nil {
		if n < 5 {
			n = 5
		}
		if n > 95 {
			n = 95
		}
		c.PedalKeyThreshold = n
	}
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idDeadzone]))); e == nil {
		if n < 0 {
			n = 0
		}
		if n > 40 {
			n = 40
		}
		c.Deadzone = n
	}
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idPedalStrength]))); e == nil {
		if n < 0 {
			n = 0
		}
		if n > 100 {
			n = 100
		}
		c.PedalSteerStrength = n
	}
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idCameraSensitivity]))); e == nil {
		if n < 10 {
			n = 10
		}
		if n > 250 {
			n = 250
		}
		c.CameraSensitivity = n
	}
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idCameraDeadzone]))); e == nil {
		if n < 0 {
			n = 0
		}
		if n > 40 {
			n = 40
		}
		c.CameraDeadzone = n
	}
	c.PreserveThrust = isChecked(a.controls[idPreserve])
	a.cfg = c
	a.cfgMu.Unlock()
}

func (a *appState) onCommand(id, notify int) {
	switch {
	case id == idStart && notify == BN_CLICKED:
		if a.mappingStarting.Load() {
			a.postStatus("Virtual Xbox controller is still starting...")
			return
		}
		if a.mappingActive.Load() || a.xboxReady.Load() {
			a.stopMapping()
			return
		}
		a.syncConfigFromUI()
		saveConfig()
		a.mappingStarting.Store(true)
		setText(a.controls[idStart], "Starting...")
		a.postStatus("Creating virtual Xbox controller for active mapping...")
		go a.startMappingAsync()
		return
	case id == idTestA && notify == BN_CLICKED:
		if !a.xboxReady.Load() {
			a.postStatus("Start mapping first; the virtual Xbox controller is intentionally disconnected while mapping is stopped.")
			return
		}
		a.testUntilMu.Lock()
		a.testUntil = time.Now().Add(350 * time.Millisecond)
		a.testUntilMu.Unlock()
		a.postStatus("Sent Xbox A test pulse.")
		return
	case id == idTestCamLeft && notify == BN_CLICKED:
		if !a.xboxReady.Load() {
			a.postStatus("Start mapping first; the virtual Xbox controller is intentionally disconnected while mapping is stopped.")
			return
		}
		a.cameraTestMu.Lock()
		a.cameraTestValue = -1.0
		a.cameraTestUntil = time.Now().Add(800 * time.Millisecond)
		a.cameraTestMu.Unlock()
		a.postStatus("Camera test: virtual Xbox Right Stick X = full LEFT for 0.8 seconds.")
		return
	case id == idTestCamRight && notify == BN_CLICKED:
		if !a.xboxReady.Load() {
			a.postStatus("Start mapping first; the virtual Xbox controller is intentionally disconnected while mapping is stopped.")
			return
		}
		a.cameraTestMu.Lock()
		a.cameraTestValue = 1.0
		a.cameraTestUntil = time.Now().Add(800 * time.Millisecond)
		a.cameraTestMu.Unlock()
		a.postStatus("Camera test: virtual Xbox Right Stick X = full RIGHT for 0.8 seconds.")
		return
	case id == idHidHideSetup && notify == BN_CLICKED:
		a.openHidHideSetup()
		return
	case id == idSave && notify == BN_CLICKED:
		a.syncConfigFromUI()
		saveConfig()
		a.postStatus("Settings saved.")
		return
	case id == idRefresh && notify == BN_CLICKED:
		a.syncConfigFromUI()
		a.devicesMu.Lock()
		a.devices = enumerateDevices()
		a.devicesMu.Unlock()
		_ = a.di.Restart(a.hwnd)
		a.populateCombos()
		a.applyConfigToUI()
		a.postStatus("Device list refreshed.")
		return
	case id == idDetectLeft && notify == BN_CLICKED:
		a.startDetect("Left")
		return
	case id == idDetectRight && notify == BN_CLICKED:
		a.startDetect("Right")
		return
	case id == idDetectWASD && notify == BN_CLICKED:
		a.startDetectThumbstick()
		return
	case id >= idWASDLearnBase && id < idWASDLearnBase+len(wasdButtonDefs) && notify == BN_CLICKED:
		def := wasdButtonDefs[id-idWASDLearnBase]
		a.startLearn(def.Key, def.Label+" (on-foot)")
		return
	case id >= idLearnBase && id < idLearnBase+len(buttonOutputs) && notify == BN_CLICKED:
		a.startLearn(buttonOutputs[id-idLearnBase].Key, buttonOutputs[id-idLearnBase].Label)
		return
	}
	if !a.uiReady {
		return
	}
	if id == idInputBackend && notify == CBN_SELCHANGE {
		backends := []string{"Auto", "DirectInput", "WinMM"}
		idx := comboIndex(idInputBackend)
		if idx < 0 || idx >= len(backends) {
			idx = 0
		}
		a.cfgMu.Lock()
		a.cfg.InputBackend = backends[idx]
		a.cfgMu.Unlock()
		a.populateCombos()
		a.applyConfigToUI()
		a.updateContextVisibility()
		saveConfig()
		a.postStatus("Input API changed to " + backends[idx] + ". DirectInput exposes additional rotation/slider axes; WinMM remains available as the compatibility fallback.")
		return
	}
	if notify == CBN_SELCHANGE || notify == EN_CHANGE || notify == BN_CLICKED {
		a.syncConfigFromUI()
		saveConfig()
		if notify == CBN_SELCHANGE {
			a.updateContextVisibility()
		}
	}
}

func (a *appState) mappingStatus() string {
	if a.xboxReady.Load() {
		s := a.xboxSlot.Load()
		if s >= 0 {
			return fmt.Sprintf("Mapping ACTIVE. Windows sees Xbox controller in XInput slot %d.", s+1)
		}
		return "Mapping ACTIVE. Virtual Xbox controller is running."
	}
	return "Mapping ACTIVE, but Xbox output is not ready yet."
}

func (a *appState) refreshXboxUI() {
	if a.controls[idStart] == 0 {
		return
	}
	if a.mappingStarting.Load() {
		setText(a.controls[idStart], "Starting...")
		return
	}
	if a.mappingActive.Load() && a.xboxReady.Load() {
		setText(a.controls[idStart], "Stop mapping")
		return
	}
	setText(a.controls[idStart], "Start mapping")
}

func (a *appState) refreshXInputReadback() {
	h := a.controls[idXInputReadback]
	if h == 0 {
		return
	}
	if !a.xboxReady.Load() {
		setText(h, "XInput readback: disconnected (virtual Xbox exists only during mapping)")
		return
	}
	slot := int(a.xboxSlot.Load())
	if slot < 0 {
		setText(h, "XInput readback: virtual Xbox connected; slot not identified yet")
		return
	}
	st, ok := xinputReadState(slot)
	if !ok {
		setText(h, fmt.Sprintf("XInput readback: slot %d not readable", slot+1))
		return
	}
	g := st.Gamepad
	setText(h, fmt.Sprintf(
		"XInput slot %d  LX %+.2f  LY %+.2f  RX %+.2f  RY %+.2f  LT %3d  RT %3d  Btn 0x%04X",
		slot+1,
		stickReadback(g.ThumbLX), stickReadback(g.ThumbLY),
		stickReadback(g.ThumbRX), stickReadback(g.ThumbRY),
		g.LeftTrigger, g.RightTrigger, g.Buttons,
	))
}

func (a *appState) setButtonTileStates(highlights uint32) {
	for i := range buttonOutputs {
		h := a.controls[idLearnBase+200+i]
		if h == 0 {
			continue
		}
		state := uintptr(0)
		if highlights&(uint32(1)<<uint(i)) != 0 {
			state = 1
		}
		procSendMessageW.Call(h, BM_SETSTATE, state, 0)
	}
}

func (a *appState) setOnFootButtonStates(highlights uint8) {
	for i := range wasdButtonDefs {
		h := a.controls[idWASDIndicatorBase+i]
		if h == 0 {
			continue
		}
		state := uintptr(0)
		if highlights&(uint8(1)<<uint(i)) != 0 {
			state = 1
		}
		procSendMessageW.Call(h, BM_SETSTATE, state, 0)
	}
}

func (a *appState) moveStickDot(id, boxX, boxY, boxSize int, x, y float64) {
	const dot = 12
	x = clamp(x, -1, 1)
	y = clamp(y, -1, 1)
	rangePx := float64(boxSize-dot) / 2.0
	cx := float64(boxX+boxSize/2-dot/2-a.scrollX) + x*rangePx
	cy := float64(boxY+boxSize/2-dot/2-a.scrollY) - y*rangePx
	procSetWindowPos.Call(
		a.controls[id],
		0,
		uintptr(int(cx)),
		uintptr(int(cy)),
		uintptr(dot),
		uintptr(dot),
		SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE,
	)
}

func (a *appState) refreshMonitor() {
	a.refreshXInputReadback()
	a.monitorMu.RLock()
	m := a.monitor
	a.monitorMu.RUnlock()
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()

	const boxSize = 126
	const leftBoxX = 600
	const rightBoxX = 770
	const outputBoxX = 940
	const boxY = 542

	// The live visual layer is intentionally suspended while controller mapping is active.
	// This leaves the fast 100 Hz input/output loop focused on the game.
	if a.mappingActive.Load() {
		if !a.visualsPaused {
			setText(a.controls[idMonitor], "Live visual monitor PAUSED while game mapping is active.\r\nController polling/output remains active at the selected polling rate.")
			a.setButtonTileStates(0)
			a.setOnFootButtonStates(0)
			a.moveStickDot(idLeftDot, leftBoxX, boxY, boxSize, 0, 0)
			a.moveStickDot(idRightDot, rightBoxX, boxY, boxSize, 0, 0)
			a.moveStickDot(idDriveSteerDot, outputBoxX, boxY, boxSize, 0, 0)
			setText(a.controls[idLeftVectorText], "Monitor paused")
			setText(a.controls[idRightVectorText], "Monitor paused")
			setText(a.controls[idDriveSteerVectorText], "Monitor paused")
			a.visualsPaused = true
		}
		return
	}
	a.visualsPaused = false

	mode := "Average thrust"
	if strings.EqualFold(normalizeControlMode(c.ControlMode), "HOTAS") {
		mode = "HOTAS: throttle -> RT, flight stick -> Xbox LS"
	} else if c.PreserveThrust {
		mode = "Hybrid preserve thrust"
	}
	pedal := "Off"
	if c.PedalID >= 0 {
		pedal = fmt.Sprintf("%+.2f", m.Pedal)
	}
	camera := "Off"
	if strings.EqualFold(c.PedalMode, "Move Camera") {
		camera = fmt.Sprintf("RX %+.2f", m.CameraX)
	} else if strings.EqualFold(c.PedalMode, "Keyboard keys") {
		camera = fmt.Sprintf("Keys %s / %s @ %d%%", keyName(c.PedalLeftKey), keyName(c.PedalRightKey), c.PedalKeyThreshold)
	}
	var s string
	if strings.EqualFold(normalizeControlMode(c.ControlMode), "HOTAS") {
		s = fmt.Sprintf(
			"Throttle %+0.2f   Stick X %+0.2f   Stick Y %+0.2f   Pedal %s   Pedal output %s\r\n%s",
			m.Drive, m.LeftX, m.LeftY, pedal, camera, mode,
		)
	} else {
		s = fmt.Sprintf(
			"Left %+0.2f   Right %+0.2f   Pedal %s   Drive %+0.2f   Steer %+0.2f   Pedal output %s\r\n%s",
			m.Left, m.Right, pedal, m.Drive, m.Steer, camera, mode,
		)
	}
	setText(a.controls[idMonitor], s)

	a.moveStickDot(idLeftDot, leftBoxX, boxY, boxSize, m.LeftX, m.LeftY)
	a.moveStickDot(idRightDot, rightBoxX, boxY, boxSize, m.RightX, m.RightY)
	a.moveStickDot(idDriveSteerDot, outputBoxX, boxY, boxSize, m.Steer, m.Drive)
	setText(a.controls[idLeftVectorText], fmt.Sprintf("Side %+0.2f  F %+0.2f", m.LeftX, m.LeftY))
	setText(a.controls[idRightVectorText], fmt.Sprintf("Side %+0.2f  F %+0.2f", m.RightX, m.RightY))
	driveDir := "Neutral"
	if m.Drive > 0.01 {
		driveDir = "Forward"
	} else if m.Drive < -0.01 {
		driveDir = "Brake/Reverse"
	}
	steerDir := "Straight"
	if m.Steer > 0.01 {
		steerDir = "Right"
	} else if m.Steer < -0.01 {
		steerDir = "Left"
	}
	setText(a.controls[idDriveSteerVectorText], fmt.Sprintf("Drive %+0.2f  %s\r\nSteer %+0.2f  %s", m.Drive, driveDir, m.Steer, steerDir))
	a.setButtonTileStates(m.OutputHighlights)
	a.setOnFootButtonStates(m.OnFootHighlights)
}

func (a *appState) postStatus(s string) {
	a.pendingMu.Lock()
	a.statusQueue = append(a.statusQueue, s)
	a.pendingMu.Unlock()
	procPostMessageW.Call(a.hwnd, WM_APP_STATUS, 0, 0)
}
func (a *appState) drainStatus() {
	a.pendingMu.Lock()
	q := append([]string(nil), a.statusQueue...)
	a.statusQueue = nil
	a.pendingMu.Unlock()
	if len(q) > 0 {
		setText(a.controls[idStatus], q[len(q)-1])
	}
}

func (a *appState) startDetect(side string) {
	a.cfgMu.RLock()
	backend := normalizeInputBackend(a.cfg.InputBackend)
	a.cfgMu.RUnlock()
	if backend != "WinMM" {
		go func() {
			guid, name, axis, _, span, score, ok := a.detectDirectInputAxis(
				fmt.Sprintf("Detecting %s stick through DirectInput: PUSH THAT STICK FORWARD now...", side), "", -1,
			)
			if !ok {
				if backend == "Auto" {
					a.postStatus(fmt.Sprintf("DirectInput did not see a strong %s movement (score %.2f); Auto is retrying with WinMM...", side, score))
					a.startDetectWinMM(side)
					return
				}
				a.queueDetect(detectResult{Side: side, Message: fmt.Sprintf("DirectInput did not see a strong axis movement (score %.2f). Try Refresh devices or switch Input API to WinMM compatibility.", score)})
				return
			}
			a.queueDetect(detectResult{Side: side, DeviceID: a.matchWinMMIDByName(name), Axis: axis, Invert: span < 0, DeviceName: name, OK: true, UseDirectInput: true, DISideGUID: guid})
		}()
		return
	}
	a.startDetectWinMM(side)
}

func (a *appState) startDetectWinMM(side string) {
	a.postStatus(fmt.Sprintf("Detecting %s stick through WinMM: PUSH THAT STICK FORWARD now...", side))
	go func() {
		devs := enumerateDevices()
		if len(devs) == 0 {
			a.queueDetect(detectResult{Side: side, Message: "No joystick devices detected."})
			return
		}
		type baseKey struct{ dev, axis int }
		bases := map[baseKey]float64{}
		for _, d := range devs {
			if st, ok := readJoy(d.ID); ok {
				for ax := 0; ax < 6; ax++ {
					bases[baseKey{d.ID, ax}] = axisNormalized(d.ID, ax, false, st)
				}
			}
		}
		bestDelta := 0.0
		bestDev := -1
		bestAxis := 0
		bestVal := 0.0
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			for _, d := range devs {
				if st, ok := readJoy(d.ID); ok {
					for ax := 0; ax < 6; ax++ {
						v := axisNormalized(d.ID, ax, false, st)
						delta := math.Abs(v - bases[baseKey{d.ID, ax}])
						if delta > bestDelta {
							bestDelta = delta
							bestDev = d.ID
							bestAxis = ax
							bestVal = v - bases[baseKey{d.ID, ax}]
						}
					}
				}
			}
			if bestDelta > 0.65 {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if bestDev < 0 || bestDelta < 0.30 {
			a.queueDetect(detectResult{Side: side, Message: "No strong axis movement detected. Click Detect and make one large forward movement."})
			return
		}
		name := fmt.Sprintf("Joystick %d", bestDev)
		for _, d := range devs {
			if d.ID == bestDev {
				name = d.Name
				break
			}
		}
		a.queueDetect(detectResult{Side: side, DeviceID: bestDev, Axis: bestAxis, Invert: bestVal < 0, DeviceName: name, OK: true})
	}()
}

func (a *appState) detectStrongAxis(devs []deviceInfo, restrictDev, excludeDev, excludeAxis int, prompt string, minDelta float64) (int, int, bool, float64, bool) {
	a.postStatus(prompt)
	type baseKey struct{ dev, axis int }
	bases := map[baseKey]float64{}
	for _, d := range devs {
		if restrictDev >= 0 && d.ID != restrictDev {
			continue
		}
		if st, ok := readJoy(d.ID); ok {
			for ax := 0; ax < 6; ax++ {
				if d.ID == excludeDev && ax == excludeAxis {
					continue
				}
				bases[baseKey{d.ID, ax}] = axisNormalized(d.ID, ax, false, st)
			}
		}
	}
	bestDelta, bestSigned := 0.0, 0.0
	bestDev, bestAxis := -1, -1
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		for _, d := range devs {
			if restrictDev >= 0 && d.ID != restrictDev {
				continue
			}
			st, ok := readJoy(d.ID)
			if !ok {
				continue
			}
			for ax := 0; ax < 6; ax++ {
				if d.ID == excludeDev && ax == excludeAxis {
					continue
				}
				base, ok := bases[baseKey{d.ID, ax}]
				if !ok {
					continue
				}
				v := axisNormalized(d.ID, ax, false, st)
				signed := v - base
				delta := math.Abs(signed)
				if delta > bestDelta {
					bestDelta, bestSigned, bestDev, bestAxis = delta, signed, d.ID, ax
				}
			}
		}
		if bestDelta > 0.65 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return bestDev, bestAxis, bestSigned < 0, bestDelta, bestDelta >= minDelta
}

func (a *appState) startDetectThumbstick() {
	go func() {
		sideGUID, sideName, sideAxis, sideCenter, sideSpan, sideScore, ok := a.detectDirectInputAxis(
			"Thumbstick detect 1/2 (DirectInput): push the small thumbstick RIGHT now...", "", -1,
		)
		if !ok {
			a.queueDetect(detectResult{Side: "Thumbstick", Message: fmt.Sprintf("DirectInput did not see a strong RIGHT movement (score %.2f). Move the thumbstick while watching the DirectInput axis monitor; X Rotation (Rx) or Y Rotation (Ry) should change.", sideScore)})
			return
		}
		a.postStatus(fmt.Sprintf("DirectInput found side axis: %s on %s. Return to center, then push FORWARD...", diAxisNames[sideAxis], sideName))
		deadline := time.Now().Add(3 * time.Second)
		tolerance := int32(math.Max(math.Abs(float64(sideSpan))*0.30, 256))
		for time.Now().Before(deadline) {
			if axes, ok := a.di.State(sideGUID); ok && int64Abs(int64(axes[sideAxis])-int64(sideCenter)) <= int64(tolerance) {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		time.Sleep(250 * time.Millisecond)
		forwardGUID, forwardName, forwardAxis, forwardCenter, forwardSpan, forwardScore, ok := a.detectDirectInputAxis(
			"Thumbstick detect 2/2 (DirectInput): push the small thumbstick FORWARD now...", sideGUID, sideAxis,
		)
		if !ok {
			a.queueDetect(detectResult{Side: "Thumbstick", Message: fmt.Sprintf("DirectInput found RIGHT on %s but did not see a strong FORWARD movement (score %.2f). Watch the DirectInput monitor and note which axis changes when you push forward.", diAxisNames[sideAxis], forwardScore)})
			return
		}
		name := sideName
		if !strings.EqualFold(sideGUID, forwardGUID) {
			name = fmt.Sprintf("%s + %s", sideName, forwardName)
		}
		a.queueDetect(detectResult{
			Side: "Thumbstick", OK: true, UseDirectInput: true, DeviceName: name,
			Axis: sideAxis, Axis2: forwardAxis,
			DISideGUID: sideGUID, DIForwardGUID: forwardGUID,
			DISideCenter: sideCenter, DIForwardCenter: forwardCenter,
			DISideSpan: sideSpan, DIForwardSpan: forwardSpan,
		})
	}()
}

func int64Abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func (a *appState) detectThumbstickLegacy() {
	devs := enumerateDevices()
	if len(devs) == 0 {
		a.queueDetect(detectResult{Side: "Thumbstick", Message: "No joystick devices detected."})
		return
	}
	dev, sideAxis, sideInvert, sideDelta, ok := a.detectStrongAxis(devs, -1, -1, -1, "Legacy thumbstick detect 1/2: push the small thumbstick RIGHT now...", 0.12)
	if !ok {
		a.queueDetect(detectResult{Side: "Thumbstick", Message: fmt.Sprintf("Thumbstick side axis not detected (strongest movement %.2f).", sideDelta)})
		return
	}
	a.postStatus("Legacy side axis found. Return it to center, then push FORWARD...")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if st, ok := readJoy(dev); ok && math.Abs(axisNormalized(dev, sideAxis, sideInvert, st)) < 0.25 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	dev2, forwardAxis, forwardInvert, forwardDelta, ok := a.detectStrongAxis(devs, -1, dev, sideAxis, "Legacy thumbstick detect 2/2: push the thumbstick FORWARD now...", 0.10)
	if !ok {
		a.queueDetect(detectResult{Side: "Thumbstick", Message: fmt.Sprintf("Legacy forward axis not detected either (strongest movement %.2f). Use the Raw HID monitor to verify Rx/Ry activity.", forwardDelta)})
		return
	}
	if dev2 != dev {
		a.queueDetect(detectResult{Side: "Thumbstick", Message: fmt.Sprintf("Legacy reader split the axes across joystick %d and %d. Raw HID should handle this; use the Raw HID monitor to inspect the two rotation axes.", dev, dev2)})
		return
	}
	name := fmt.Sprintf("Joystick %d", dev)
	for _, d := range devs {
		if d.ID == dev {
			name = d.Name
			break
		}
	}
	a.queueDetect(detectResult{Side: "Thumbstick", DeviceID: dev, Axis: sideAxis, Axis2: forwardAxis, Invert: sideInvert, Invert2: forwardInvert, DeviceName: name, OK: true})
}

func (a *appState) queueDetect(r detectResult) {
	a.pendingMu.Lock()
	a.detectQueue = append(a.detectQueue, r)
	a.pendingMu.Unlock()
	procPostMessageW.Call(a.hwnd, WM_APP_DETECT, 0, 0)
}
func (a *appState) drainDetect() {
	a.pendingMu.Lock()
	q := append([]detectResult(nil), a.detectQueue...)
	a.detectQueue = nil
	a.pendingMu.Unlock()
	for _, r := range q {
		if !r.OK {
			a.postStatus(r.Message)
			continue
		}
		a.cfgMu.Lock()
		if r.Side == "Left" {
			if r.UseDirectInput {
				a.cfg.LeftDIGUID = r.DISideGUID
				if r.DeviceID >= 0 {
					a.cfg.LeftID = r.DeviceID
				}
			} else {
				a.cfg.LeftID = r.DeviceID
			}
			a.cfg.LeftAxis = r.Axis
			a.cfg.LeftInvert = r.Invert
		} else if r.Side == "Right" {
			if r.UseDirectInput {
				a.cfg.RightDIGUID = r.DISideGUID
				if r.DeviceID >= 0 {
					a.cfg.RightID = r.DeviceID
				}
			} else {
				a.cfg.RightID = r.DeviceID
			}
			a.cfg.RightAxis = r.Axis
			a.cfg.RightInvert = r.Invert
		} else if r.Side == "Thumbstick" {
			if r.UseDirectInput {
				a.cfg.OnFootMode = "Thumbstick -> WASD"
				a.cfg.ThumbWASDEnabled = true
				a.cfg.ThumbWASDUseRawHID = false
				a.cfg.ThumbDISideGUID = r.DISideGUID
				a.cfg.ThumbDIForwardGUID = r.DIForwardGUID
				a.cfg.ThumbDIDeviceName = r.DeviceName
				a.cfg.ThumbDISideAxis = r.Axis
				a.cfg.ThumbDIForwardAxis = r.Axis2
				a.cfg.ThumbDISideCenter = r.DISideCenter
				a.cfg.ThumbDIForwardCenter = r.DIForwardCenter
				a.cfg.ThumbDISideSpan = r.DISideSpan
				a.cfg.ThumbDIForwardSpan = r.DIForwardSpan
				a.cfg.ThumbWASDInvertSide = false
				a.cfg.ThumbWASDInvertForward = false
			}
		}
		a.cfgMu.Unlock()
		a.applyConfigToUI()
		saveConfig()
		if r.Side == "Thumbstick" {
			if r.UseDirectInput {
				a.postStatus(fmt.Sprintf("Thumbstick detected through DirectInput: %s, side=%s, forward=%s. Right and forward are calibrated positive.", r.DeviceName, diAxisNames[r.Axis], diAxisNames[r.Axis2]))
			}
		} else {
			inv := "normal"
			if r.Invert {
				inv = "inverted"
			}
			if r.UseDirectInput {
				axisName := fmt.Sprintf("axis %d", r.Axis)
				if r.Axis >= 0 && r.Axis < len(diAxisNames) {
					axisName = diAxisNames[r.Axis]
				}
				a.postStatus(fmt.Sprintf("%s detected through DirectInput: %s, %s (%s). Forward is now positive.", r.Side, r.DeviceName, axisName, inv))
			} else {
				axisName := fmt.Sprintf("axis %d", r.Axis)
				if r.Axis >= 0 && r.Axis < len(a.axisNames) {
					axisName = a.axisNames[r.Axis]
				}
				a.postStatus(fmt.Sprintf("%s detected through WinMM: %s [ID %d], %s (%s). Forward is now positive.", r.Side, r.DeviceName, r.DeviceID, axisName, inv))
			}
		}
	}
}

func (a *appState) startLearn(key, label string) {
	a.postStatus(fmt.Sprintf("Learning %s: press the VKB button, trigger, hat, or switch position you want...", label))
	go func() {
		devs := enumerateDevices()
		type baseline struct {
			buttons uint32
			pov     uint32
		}
		base := map[int]baseline{}
		for _, d := range devs {
			if st, ok := readJoy(d.ID); ok {
				base[d.ID] = baseline{st.Buttons, st.POV}
			}
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			for _, d := range devs {
				st, ok := readJoy(d.ID)
				if !ok {
					continue
				}
				b := base[d.ID]
				pressed := st.Buttons &^ b.buttons
				if pressed != 0 {
					for i := 0; i < 32; i++ {
						if pressed&(1<<uint(i)) != 0 {
							a.queueLearn(learnResult{Key: key, Binding: binding{Kind: "button", DeviceID: d.ID, Button: i}, OK: true})
							return
						}
					}
				}
				if st.POV != JOY_POVCENTERED && st.POV != b.pov {
					a.queueLearn(learnResult{Key: key, Binding: binding{Kind: "pov", DeviceID: d.ID, POV: int(st.POV)}, OK: true})
					return
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		a.queueLearn(learnResult{Key: key, Message: "Nothing detected. Try Learn again and press/move the control after clicking Learn."})
	}()
}
func (a *appState) queueLearn(r learnResult) {
	a.pendingMu.Lock()
	a.learnQueue = append(a.learnQueue, r)
	a.pendingMu.Unlock()
	procPostMessageW.Call(a.hwnd, WM_APP_LEARN, 0, 0)
}
func (a *appState) drainLearn() {
	a.pendingMu.Lock()
	q := append([]learnResult(nil), a.learnQueue...)
	a.learnQueue = nil
	a.pendingMu.Unlock()
	for _, r := range q {
		if !r.OK {
			a.postStatus(r.Message)
			continue
		}
		a.cfgMu.Lock()
		if a.cfg.Bindings == nil {
			a.cfg.Bindings = make(map[string]binding)
		}
		a.cfg.Bindings[r.Key] = r.Binding
		a.cfgMu.Unlock()
		saveConfig()
		a.updateBindingLabels()
		a.postStatus(fmt.Sprintf("%s assigned to %s.", outputLabel(r.Key), bindingDisplay(r.Binding)))
	}
}
func outputLabel(k string) string {
	for _, b := range buttonOutputs {
		if b.Key == k {
			return b.Label
		}
	}
	for _, b := range wasdButtonDefs {
		if b.Key == k {
			return b.Label + " (on-foot)"
		}
	}
	return k
}
func (a *appState) updateBindingLabels() {
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()
	for i, bo := range buttonOutputs {
		s := "Not assigned"
		if b, ok := c.Bindings[bo.Key]; ok {
			s = bindingDisplay(b)
		}
		setText(a.controls[idLearnBase+100+i], s)
	}
	for i, def := range wasdButtonDefs {
		s := "Not assigned"
		if b, ok := c.Bindings[def.Key]; ok {
			s = bindingDisplay(b)
		}
		setText(a.controls[idWASDLabelBase+i], s)
	}
}
func bindingDisplay(b binding) string {
	if strings.EqualFold(b.Kind, "button") {
		return fmt.Sprintf("Joystick %d / Button %d", b.DeviceID, b.Button+1)
	}
	if strings.EqualFold(b.Kind, "pov") {
		return fmt.Sprintf("Joystick %d / POV %s", b.DeviceID, povName(b.POV))
	}
	return "Not assigned"
}
func povName(v int) string {
	switch {
	case v < 2250 || v >= 33750:
		return "Up"
	case v < 6750:
		return "Up-Right"
	case v < 11250:
		return "Right"
	case v < 15750:
		return "Down-Right"
	case v < 20250:
		return "Down"
	case v < 24750:
		return "Down-Left"
	case v < 29250:
		return "Left"
	default:
		return "Up-Left"
	}
}
