//go:build windows

// Galactic Racer HOSAS / HOTAS Bridge
// Copyright (C) 2026 JaffaQuake
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
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
	appTitle = "Galactic Racer HOSAS / HOTAS Bridge v2.2.3 — by JaffaQuake and Mars"

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

	BS_PUSHBUTTON      = 0x00000000
	BS_MULTILINE       = 0x00002000
	BS_AUTORADIOBUTTON = 0x00000009
	BS_PUSHLIKE        = 0x00001000
	WS_GROUP           = 0x00020000
	BS_AUTOCHECKBOX    = 0x00000003
	CBS_DROPDOWNLIST   = 0x00000003
	LBS_NOTIFY         = 0x00000001
	ES_NUMBER          = 0x00002000
	SS_LEFT            = 0x00000000

	CW_USEDEFAULT     = ^uintptr(0x7fffffff)
	SW_HIDE           = 0
	SW_SHOWNOACTIVATE = 4
	SW_SHOW           = 5

	WM_CREATE         = 0x0001
	WM_SETTEXT        = 0x000C
	WM_DESTROY        = 0x0002
	WM_SIZE           = 0x0005
	WM_CLOSE          = 0x0010
	WM_LBUTTONDOWN    = 0x0201
	WM_MOUSEWHEEL     = 0x020A
	WM_PARENTNOTIFY   = 0x0210
	WM_NOTIFY         = 0x004E
	WM_COMMAND        = 0x0111
	WM_HSCROLL        = 0x0114
	WM_VSCROLL        = 0x0115
	WM_TIMER          = 0x0113
	WM_SETFONT        = 0x0030
	WM_CTLCOLORSTATIC = 0x0138
	WM_SETICON        = 0x0080
	WM_APP            = 0x8000

	WM_APP_STATUS  = WM_APP + 1
	WM_APP_DETECT  = WM_APP + 2
	WM_APP_LEARN   = WM_APP + 3
	WM_APP_XBOX    = WM_APP + 4
	WM_APP_PROFILE = WM_APP + 5

	BM_GETCHECK = 0x00F0
	BM_SETCHECK = 0x00F1
	BM_SETSTATE = 0x00F3
	BST_CHECKED = 1

	SWP_NOSIZE     = 0x0001
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040
	SWP_HIDEWINDOW = 0x0080

	RDW_INVALIDATE  = 0x0001
	RDW_ERASE       = 0x0004
	RDW_ALLCHILDREN = 0x0080
	RDW_UPDATENOW   = 0x0100

	CB_ADDSTRING    = 0x0143
	CB_RESETCONTENT = 0x014B
	CB_GETCURSEL    = 0x0147
	CB_SETCURSEL    = 0x014E
	CB_SHOWDROPDOWN = 0x014F
	CB_GETCOUNT     = 0x0146
	TCM_FIRST       = 0x1300
	TCM_GETCURSEL   = TCM_FIRST + 11
	TCM_ADJUSTRECT  = TCM_FIRST + 40
	TCM_SETCURSEL   = TCM_FIRST + 12
	TCM_INSERTITEMW = TCM_FIRST + 62
	TCIF_TEXT       = 0x0001
	TCN_SELCHANGE   = -551
	LB_ADDSTRING    = 0x0180
	LB_RESETCONTENT = 0x0184
	LB_GETCURSEL    = 0x0188
	LB_SETCURSEL    = 0x0186

	BN_CLICKED    = 0
	CBN_SELCHANGE = 1
	LBN_SELCHANGE = 1
	LBN_DBLCLK    = 2
	LBN_KILLFOCUS = 5
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
	idHotasBrakeBehavior
	idHotasBrakeThreshold
	idHotasBrakeZone
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
	idHotasDIAxisMonitor
	idHelpText
	idTwistLeftDevice
	idTwistLeftAxis
	idTwistLeftInvert
	idTwistLeftAxisType
	idTwistLeftDeadzone
	idTwistLeftMode
	idTwistLeftStrength
	idTwistLeftCameraSensitivity
	idTwistLeftCameraDeadzone
	idTwistLeftLeftKey
	idTwistLeftRightKey
	idTwistLeftKeyThreshold
	idTwistRightDevice
	idTwistRightAxis
	idTwistRightInvert
	idTwistRightAxisType
	idTwistRightDeadzone
	idTwistRightMode
	idTwistRightStrength
	idTwistRightCameraSensitivity
	idTwistRightCameraDeadzone
	idTwistRightLeftKey
	idTwistRightRightKey
	idTwistRightKeyThreshold
	idTwistLeftOutputNegative
	idTwistRightOutputNegative
	idAxisTuning
	idSocialLinks
	idMonitorDuringGameplay
	idMappedThumbstick
	idMappingsTab    = 1720
	idCalibrationTab = 1721
	idRightPanelTabs = 1722
	idWalkingToggle  = 1890
	idWalkingLearn   = 1891
	idWalkingClear   = 1892
	idWalkingLabel   = 1893
	idLearnBase      = 1200
)

const (
	idWheelDevice = 1800 + iota
	idWheelAxis
	idWheelInvert
	idWheelDetect
	idWheelPedal1Device
	idWheelPedal1Axis
	idWheelPedal1Invert
	idWheelPedal1Detect
	idWheelPedal2Device
	idWheelPedal2Axis
	idWheelPedal2Invert
	idWheelPedal2Detect
	idWheelPedal3Device
	idWheelPedal3Axis
	idWheelPedal3Invert
	idWheelPedal3Detect
	idWheelPedal3Mode
	idWheelHandbrakeButton
	idWheelHandbrakeLearn
	idWheelHandbrakeDetect
	idWheelCenteringStrength
	idWheelRumbleStrength
	idWheelAudioStrength
	idWheelTrueforce
)

const (
	idProfileFirst        = 1950 // Five numbered independent saved-profile buttons.
	idProfileIncludeFirst = 1960 // Global flags: include these slots in cycling.
	idCycleLearn          = 1970
	idCycleClear          = 1971
	idCycleLabel          = 1972
)

const (
	idHOTATBrakeBehavior = 1900 + iota
	idHOTATBrakeThreshold
	idHOTATBrakeZone
	idLeftThrottleBar
	idRightThrottleBar
	idWheelThrottleBar
	idWheelBrakeBar
)

const (
	PBS_SMOOTH   = 0x01
	PBS_VERTICAL = 0x04
)

var controlModeValues = []string{"HOSAS", "HOTAT", "HOTAS", "SINGLE", "WHEEL"}
var controlModeLabels = []string{"Podracer Style (HOSAS)", "Podracer Style (HOTAT)", "Traditional HOTAS", "Single Stick", "Driving Wheel"}
var brakeModeValues = []string{"Off", "Digital at Idle", "Analog Brake Zone", "Centered Stick"}

const (
	idWASDLearnBase     = 1500
	idWASDLabelBase     = 1510
	idWASDIndicatorBase = 1520
	idLearnClearBase    = 1600
	idWASDClearBase     = 1650
)

// Native Windows tab control notifications and item descriptors (Win64 layout).
type tabNotifyHeader struct {
	HwndFrom uintptr
	IDFrom   uintptr
	Code     int32
	_        uint32
}
type tabControlItem struct {
	Mask      uint32
	State     uint32
	StateMask uint32
	Text      *uint16
	TextMax   int32
	Image     int32
	Param     uintptr
}

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
	Kind       string `json:"kind"`
	Backend    string `json:"backend,omitempty"`
	DeviceID   int    `json:"device_id"`
	DIGUID     string `json:"di_guid,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
	Button     int    `json:"button,omitempty"`
	POVIndex   int    `json:"pov_index,omitempty"`
	POV        int    `json:"pov,omitempty"`
}

type keyOption struct {
	Name string
	VK   int
}

// auxiliaryAxisConfig mirrors the pedal controls without sharing settings between axes.
// A lever is 0..1 travel; a centered control is -1..+1 around its midpoint.
type auxiliaryAxisConfig struct {
	DeviceID          int    `json:"device_id"`
	DIGUID            string `json:"di_guid"`
	Axis              int    `json:"axis"`
	Invert            bool   `json:"invert"`
	OutputNegative    bool   `json:"output_negative"`
	AxisType          string `json:"axis_type"`
	Deadzone          int    `json:"deadzone"`
	Mode              string `json:"mode"`
	SteerStrength     int    `json:"steer_strength"`
	CameraSensitivity int    `json:"camera_sensitivity"`
	CameraDeadzone    int    `json:"camera_deadzone"`
	LeftKey           int    `json:"left_key"`
	RightKey          int    `json:"right_key"`
	KeyThreshold      int    `json:"key_threshold"`
}

func defaultAuxiliaryAxis() auxiliaryAxisConfig {
	return auxiliaryAxisConfig{DeviceID: -1, AxisType: "Centered", Deadzone: 6, Mode: "Off",
		SteerStrength: 30, CameraSensitivity: 100, CameraDeadzone: 6,
		LeftKey: VK_DELETE, RightKey: VK_NEXT, KeyThreshold: 20}
}

func normalizeAxisType(s string) string {
	if strings.EqualFold(strings.TrimSpace(s), "Lever") {
		return "Lever"
	}
	return "Centered"
}

func validateAuxiliaryAxis(v *auxiliaryAxisConfig, present bool, maxAxis int) {
	if !present {
		*v = defaultAuxiliaryAxis()
		return
	}
	v.AxisType = normalizeAxisType(v.AxisType)
	v.Mode = normalizePedalMode(v.Mode)
	if v.DeviceID < -1 {
		v.DeviceID = -1
	}
	if v.Axis < 0 || v.Axis > maxAxis {
		v.Axis = 0
	}
	if v.Deadzone < 0 || v.Deadzone > 40 {
		v.Deadzone = 6
	}
	if v.SteerStrength < 0 || v.SteerStrength > 100 {
		v.SteerStrength = 30
	}
	if v.CameraSensitivity < 10 || v.CameraSensitivity > 250 {
		v.CameraSensitivity = 100
	}
	if v.CameraDeadzone < 0 || v.CameraDeadzone > 40 {
		v.CameraDeadzone = 6
	}
	if !validKeyboardVK(v.LeftKey) {
		v.LeftKey = VK_DELETE
	}
	if !validKeyboardVK(v.RightKey) {
		v.RightKey = VK_NEXT
	}
	if v.KeyThreshold < 5 || v.KeyThreshold > 95 {
		v.KeyThreshold = 20
	}
}

// Existing v1.98.x shared deadzone migrates to independent axis profiles.
// Legacy fields are preserved in JSON for safe downgrade, but v1.99 applies
// the new profile values during live controller processing.
func initAxisProfiles(c *config) {
	if c.Tuning == nil {
		c.Tuning = make(map[string]axisTuneProfile)
	}
	for _, entry := range axisTuneNames {
		if p, ok := c.Tuning[entry.Key]; ok {
			c.Tuning[entry.Key] = checkedAxisTune(p)
			continue
		}
		dz := 0
		switch entry.Key {
		case profileHOSASLeft, profileHOSASRight, profileHOTASX, profileHOTASY, profilePedal, profileWheelSteering, profileWheelPedal1, profileWheelPedal2, profileWheelPedal3, profileWheelHandbrake:
			dz = c.Deadzone
		case profileTwistLeft:
			dz = c.TwistLeft.Deadzone
		case profileTwistRight:
			dz = c.TwistRight.Deadzone
		}
		c.Tuning[entry.Key] = defaultAxisTune(dz)
	}
}

func profileFromConfig(c config, key string) axisTuneProfile {
	if p, ok := c.Tuning[key]; ok {
		return checkedAxisTune(p)
	}
	return defaultAxisTune(0)
}

func normalizeAuxiliaryValue(centered float64, v auxiliaryAxisConfig, deadzone int) float64 {
	if normalizeAxisType(v.AxisType) == "Lever" {
		out := clamp((centered+1)/2, 0, 1)
		if v.Invert {
			out = 1 - out
		}
		dz := clamp(float64(deadzone)/100, 0, .4)
		if out <= dz {
			return 0
		}
		out = clamp((out-dz)/(1-dz), 0, 1)
		if v.OutputNegative {
			return -out
		}
		return out
	}
	if v.Invert {
		centered = -centered
	}
	return applyDeadzone(centered, deadzone)
}

type wheelAxisConfig struct {
	DeviceID int    `json:"device_id"`
	DIGUID   string `json:"di_guid"`
	Axis     int    `json:"axis"`
	Invert   bool   `json:"invert"`
}
type drivingWheelConfig struct {
	Wheel           wheelAxisConfig `json:"wheel"`
	Pedal1          wheelAxisConfig `json:"pedal_1"`
	Pedal2          wheelAxisConfig `json:"pedal_2"`
	Pedal3          wheelAxisConfig `json:"pedal_3"`
	Pedal3Mode      string          `json:"pedal_3_mode"`
	Handbrake       wheelAxisConfig `json:"handbrake"`
	HandbrakeMode   string          `json:"handbrake_mode"`
	HandbrakeButton bool            `json:"handbrake_button"`
}

func defaultDrivingWheel() drivingWheelConfig {
	none := wheelAxisConfig{DeviceID: -1}
	return drivingWheelConfig{Wheel: none, Pedal1: none, Pedal2: none, Pedal3: none, Handbrake: none, Pedal3Mode: "Off", HandbrakeMode: "Drift"}
}
func normalizeDrivingWheel(c *drivingWheelConfig, present bool, maxAxis int) {
	if !present {
		*c = defaultDrivingWheel()
		return
	}
	for _, v := range []*wheelAxisConfig{&c.Wheel, &c.Pedal1, &c.Pedal2, &c.Pedal3, &c.Handbrake} {
		if v.DeviceID < -1 {
			v.DeviceID = -1
		}
		if v.Axis < 0 || v.Axis > maxAxis {
			v.Axis = 0
		}
	}
	switch c.Pedal3Mode {
	case "Off", "Boost", "Drift":
	default:
		c.Pedal3Mode = "Off"
	}
	if c.HandbrakeMode != "Boost" && c.HandbrakeMode != "Drift" {
		c.HandbrakeMode = "Drift"
	}
}

func axisDigitalButtons(mode string, value float64) uint16 {
	switch digitalAxisAction(mode, value) {
	case "Boost":
		return XUSB_A
	case "Drift":
		return XUSB_X
	}
	return 0
}

func drivingPedals(accel, brake float64) (byte, byte) {
	return pedalTriggerByte(accel), pedalTriggerByte(brake)
}

type config struct {
	DrivingWheel           drivingWheelConfig         `json:"driving_wheel"`
	InputBackend           string                     `json:"input_backend"`
	ControlMode            string                     `json:"control_mode"`
	HotasThrottleID        int                        `json:"hotas_throttle_id"`
	HotasThrottleDIGUID    string                     `json:"hotas_throttle_di_guid"`
	HotasThrottleAxis      int                        `json:"hotas_throttle_axis"`
	HotasThrottleInvert    bool                       `json:"hotas_throttle_invert"`
	HotasBrakeBehavior     string                     `json:"hotas_brake_behavior"`
	HotasBrakeThreshold    int                        `json:"hotas_brake_threshold"`
	HotasBrakeZone         int                        `json:"hotas_brake_zone"`
	HotatBrakeBehavior     string                     `json:"hotat_brake_behavior"`
	HotatBrakeThreshold    int                        `json:"hotat_brake_threshold"`
	HotatBrakeZone         int                        `json:"hotat_brake_zone"`
	HotasStickID           int                        `json:"hotas_stick_id"`
	HotasStickDIGUID       string                     `json:"hotas_stick_di_guid"`
	HotasStickXAxis        int                        `json:"hotas_stick_x_axis"`
	HotasStickYAxis        int                        `json:"hotas_stick_y_axis"`
	HotasStickInvertX      bool                       `json:"hotas_stick_invert_x"`
	HotasStickInvertY      bool                       `json:"hotas_stick_invert_y"`
	LeftID                 int                        `json:"left_id"`
	LeftDIGUID             string                     `json:"left_di_guid"`
	RightID                int                        `json:"right_id"`
	RightDIGUID            string                     `json:"right_di_guid"`
	PedalID                int                        `json:"pedal_id"`
	PedalDIGUID            string                     `json:"pedal_di_guid"`
	TwistLeft              auxiliaryAxisConfig        `json:"twist_left"`
	TwistRight             auxiliaryAxisConfig        `json:"twist_right"`
	LeftAxis               int                        `json:"left_axis"`
	RightAxis              int                        `json:"right_axis"`
	PedalAxis              int                        `json:"pedal_axis"`
	LeftInvert             bool                       `json:"left_invert"`
	RightInvert            bool                       `json:"right_invert"`
	LeftThrottleMode       bool                       `json:"left_throttle_mode"`
	RightThrottleMode      bool                       `json:"right_throttle_mode"`
	PedalInvert            bool                       `json:"pedal_invert"`
	PedalMode              string                     `json:"pedal_mode"`
	Deadzone               int                        `json:"deadzone"`
	Tuning                 map[string]axisTuneProfile `json:"axis_tuning,omitempty"`
	Bindings               map[string]binding         `json:"bindings"`
	PreserveThrust         bool                       `json:"preserve_thrust"`
	PreserveThrustV2       bool                       `json:"preserve_thrust_v2_initialized,omitempty"`
	PedalSteerStrength     int                        `json:"pedal_steer_strength"`
	CameraSensitivity      int                        `json:"camera_sensitivity"`
	CameraDeadzone         int                        `json:"camera_deadzone"`
	PedalLeftKey           int                        `json:"pedal_left_key"`
	PedalRightKey          int                        `json:"pedal_right_key"`
	PedalKeyThreshold      int                        `json:"pedal_key_threshold"`
	ThumbWASDEnabled       bool                       `json:"thumb_wasd_enabled"`
	ThumbWASDDeviceID      int                        `json:"thumb_wasd_device_id"`
	ThumbWASDSideAxis      int                        `json:"thumb_wasd_side_axis"`
	ThumbWASDForwardAxis   int                        `json:"thumb_wasd_forward_axis"`
	ThumbWASDInvertSide    bool                       `json:"thumb_wasd_invert_side"`
	ThumbWASDInvertForward bool                       `json:"thumb_wasd_invert_forward"`
	ThumbWASDThreshold     int                        `json:"thumb_wasd_threshold"`
	ThumbWASDUseRawHID     bool                       `json:"thumb_wasd_use_raw_hid"`
	ThumbHIDSidePath       string                     `json:"thumb_hid_side_path"`
	ThumbHIDForwardPath    string                     `json:"thumb_hid_forward_path"`
	ThumbHIDSideUsage      int                        `json:"thumb_hid_side_usage"`
	ThumbHIDForwardUsage   int                        `json:"thumb_hid_forward_usage"`
	ThumbHIDSideCenter     int32                      `json:"thumb_hid_side_center"`
	ThumbHIDForwardCenter  int32                      `json:"thumb_hid_forward_center"`
	ThumbHIDSideSpan       int32                      `json:"thumb_hid_side_span"`
	ThumbHIDForwardSpan    int32                      `json:"thumb_hid_forward_span"`
	OnFootMode             string                     `json:"on_foot_mode"` // retained for old configuration files; inactive since v2.2
	ThumbDISideGUID        string                     `json:"thumb_di_side_guid"`
	ThumbDIForwardGUID     string                     `json:"thumb_di_forward_guid"`
	ThumbDIDeviceName      string                     `json:"thumb_di_device_name"`
	ThumbDISideAxis        int                        `json:"thumb_di_side_axis"`
	ThumbDIForwardAxis     int                        `json:"thumb_di_forward_axis"`
	ThumbDISideCenter      int32                      `json:"thumb_di_side_center"`
	ThumbDIForwardCenter   int32                      `json:"thumb_di_forward_center"`
	ThumbDISideSpan        int32                      `json:"thumb_di_side_span"`
	ThumbDIForwardSpan     int32                      `json:"thumb_di_forward_span"`
	PollingHz              int                        `json:"polling_hz"`
	WheelRumbleStrength    int                        `json:"wheel_rumble_strength"`    // 0 = default, <0 = off
	WheelCenteringStrength int                        `json:"wheel_centering_strength"` // 0 = default, <0 = off
	WheelAudioStrength     int                        `json:"wheel_audio_strength"`     // 0 = default, <0 = off
	WheelTrueforce         bool                       `json:"wheel_trueforce"`
	MonitorDuringGameplay  bool                       `json:"monitor_during_gameplay"`
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
	Left, Right, Pedal, TwistLeft, TwistRight, Drive, Steer float64
	Throttle                                                float64
	HotasBrake                                              bool
	HotasBrakeAmount                                        float64
	RawWheelThrottle, RawWheelBrake                         float64
	LeftX, LeftY, RightX, RightY                            float64
	CameraX                                                 float64
	OutputButtons                                           uint16
	OutputHighlights                                        uint32
	OnFootHighlights                                        uint8
	WalkLX, WalkLY, WalkRX, WalkRY                          float64
	LastUpdate                                              time.Time
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
	Buttons             [128]byte
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
	ffb     wheelFFB
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

// Standard DirectInput object GUIDs. Binding the data format to these exact
// object types prevents a generic "axis slot" from being mislabeled as Rx/Ry/Rz.
var (
	diGUIDXAxis  = diGUID{Data1: 0xA36D02E0, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDYAxis  = diGUID{Data1: 0xA36D02E1, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDZAxis  = diGUID{Data1: 0xA36D02E2, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDRzAxis = diGUID{Data1: 0xA36D02E3, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDSlider = diGUID{Data1: 0xA36D02E4, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDButton = diGUID{Data1: 0xA36D02F0, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDPOV    = diGUID{Data1: 0xA36D02F2, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDRxAxis = diGUID{Data1: 0xA36D02F4, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	diGUIDRyAxis = diGUID{Data1: 0xA36D02F5, Data2: 0xC9F3, Data3: 0x11CF, Data4: [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
)

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
	axisType := uint32(didftAxis | didftAnyInstance | didftOptional)
	povType := uint32(didftPOV | didftAnyInstance | didftOptional)
	buttonType := uint32(didftButton | didftAnyInstance | didftOptional)
	objs := make([]diObjectDataFormat, 0, 140)
	// These offsets match diJoyState/DIJOYSTATE2's position fields exactly.
	axisDefs := []struct {
		guid *diGUID
		ofs  uint32
	}{
		{&diGUIDXAxis, 0}, {&diGUIDYAxis, 4}, {&diGUIDZAxis, 8},
		{&diGUIDRxAxis, 12}, {&diGUIDRyAxis, 16}, {&diGUIDRzAxis, 20},
		{&diGUIDSlider, 24}, {&diGUIDSlider, 28},
	}
	for _, d := range axisDefs {
		objs = append(objs, diObjectDataFormat{Pguid: uintptr(unsafe.Pointer(d.guid)), DwOfs: d.ofs, DwType: axisType})
	}
	for _, ofs := range []uint32{32, 36, 40, 44} {
		objs = append(objs, diObjectDataFormat{Pguid: uintptr(unsafe.Pointer(&diGUIDPOV)), DwOfs: ofs, DwType: povType})
	}
	for i := 0; i < 128; i++ {
		objs = append(objs, diObjectDataFormat{DwOfs: uint32(48 + i), DwType: buttonType})
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
	m.releaseWheelFFBLocked()
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

func (m *directInputManager) FullState(guid string) (diJoyState, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.devices {
		if strings.EqualFold(d.GUIDString, guid) {
			st, ok := m.stateLocked(d)
			if ok {
				m.noteActivityLocked(d, diAxes(st))
			}
			return st, d.Name, ok
		}
	}
	return diJoyState{}, "", false
}

type diFullSnapshot struct {
	GUID  string
	Name  string
	State diJoyState
}

func (m *directInputManager) FullSnapshots() []diFullSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]diFullSnapshot, 0, len(m.devices))
	for _, d := range m.devices {
		st, ok := m.stateLocked(d)
		if !ok {
			continue
		}
		m.noteActivityLocked(d, diAxes(st))
		out = append(out, diFullSnapshot{GUID: d.GUIDString, Name: d.Name, State: st})
	}
	return out
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
	// v1.96 asks DirectInput to expose analog axes as -32768..32767.
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

func formatDIAxis5(axes [8]int32) string {
	vals := make([]string, 0, 5)
	for _, idx := range []int{3, 4, 5, 6, 7} {
		vals = append(vals, fmt.Sprintf("%s %+.2f", diAxisNames[idx], normalizeDIAbsoluteAxis(axes[idx], false)))
	}
	return strings.Join(vals, "  ")
}

func (a *appState) refreshHotasDirectInputMonitor() {
	h := a.controls[idHotasDIAxisMonitor]
	if h == 0 {
		return
	}
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()
	if !strings.EqualFold(normalizeControlMode(c.ControlMode), "HOTAS") {
		return
	}
	if a.mappingActive.Load() {
		setText(h, "DirectInput axes: live Rx/Ry/Rz/Slider values paused while Xbox output is active.")
		return
	}
	if normalizeInputBackend(c.InputBackend) == "WinMM" {
		setText(h, "DirectInput axes: select Auto or DirectInput to view Rx/Ry/Rz/Slider values here.")
		return
	}
	parts := []string{}
	if c.HotasStickDIGUID != "" {
		if ax, name, ok := a.di.StateByGUID(c.HotasStickDIGUID); ok {
			parts = append(parts, "Stick "+name+": "+formatDIAxis5(ax))
		}
	}
	if c.HotasThrottleDIGUID != "" && !strings.EqualFold(c.HotasThrottleDIGUID, c.HotasStickDIGUID) {
		if ax, name, ok := a.di.StateByGUID(c.HotasThrottleDIGUID); ok {
			parts = append(parts, "Throttle "+name+": "+formatDIAxis5(ax))
		}
	}
	if len(parts) == 0 {
		setText(h, "DirectInput axes: select a Traditional HOTAS stick/throttle device to view Rx/Ry/Rz/Slider values.")
		return
	}
	setText(h, strings.Join(parts, " | "))
}

type xboxBridge struct {
	mu                                                             sync.Mutex
	dll                                                            *syscall.DLL
	alloc, free, connect, disconnect                               *syscall.Proc
	targetAlloc, targetFree, targetAdd, targetRemove, targetUpdate *syscall.Proc
	notifyUnregister                                               *syscall.Proc
	client                                                         uintptr
	target                                                         uintptr
	ready                                                          bool
}

type appState struct {
	hwnd                uintptr
	font                uintptr
	headerFont          uintptr
	actionFont          uintptr
	testFont            uintptr
	cfgMu               sync.RWMutex
	cfg                 config
	profileSwitchMu     sync.RWMutex // serializes poll output and in-place profile swapping
	profilesMu          sync.Mutex
	profiles            profileCollection
	profileSaveDisabled bool
	applyingProfile     bool
	devicesMu           sync.RWMutex
	devices             []deviceInfo

	controls         map[int]uintptr
	deviceComboIDs   map[int][]int
	deviceComboGUIDs map[int][]string
	axisNames        []string

	uiReady                  bool
	visualsPaused            bool
	mappingActive            atomic.Bool
	walkingActive            atomic.Bool
	mappingStarting          atomic.Bool
	profileCyclePending      atomic.Bool
	xboxReady                atomic.Bool
	xboxSlot                 atomic.Int32
	testUntilMu              sync.RWMutex
	testUntil                time.Time
	cameraTestMu             sync.RWMutex
	cameraTestUntil          time.Time
	cameraTestValue          float64
	monitorMu                sync.RWMutex
	monitor                  monitorState
	keyboardMu               sync.Mutex
	keyboardDown             map[uint16]bool
	tooltipHwnd              uintptr
	tooltipTexts             []*uint16
	tooltipHelp              map[uintptr]string
	tooltipPopup             uintptr // legacy field; v1.96 renders help inside the main window
	tooltipHover             uintptr
	comboOptions             map[int][]string
	comboSelected            map[int]int
	baseRects                map[uintptr]rect
	scrollX                  int
	scrollY                  int
	contentWidth             int
	contentHeight            int
	tuning                   *axisTuningPopup
	activeRightTab           int // 0 = mappings, 1 = embedded tuning, 2 = wheel haptics; never persisted
	mappingTabGroup          []uintptr
	calibrationTabGroup      []uintptr
	wheelFFBTabGroup         []uintptr
	monitorLeftHeading       uintptr
	monitorRightHeading      uintptr
	monitorOutputHeading     uintptr
	monitorLeftForwardAxis   uintptr
	monitorLeftBackwardAxis  uintptr
	monitorRightForwardAxis  uintptr
	monitorRightBackwardAxis uintptr
	tuningDirty              bool
	tuningSaveDue            time.Time
	controlHOSASGroup        []uintptr
	hosasLeftDetailGroup     []uintptr
	hosasRightDetailGroup    []uintptr
	hotasThrottleDetailGroup []uintptr
	hotasStickDetailGroup    []uintptr
	hotasStickRow            []uintptr
	hotasThrottleRow         []uintptr
	singleStickRowShifted    bool
	hotatBrakeGroup          []uintptr
	hotasDILabel             uintptr
	hotatBrakeDigitalGroup   []uintptr
	hotatBrakeAnalogGroup    []uintptr
	monitorLeftGrid          []uintptr
	monitorRightGrid         []uintptr
	monitorLeftSides         []uintptr
	monitorRightSides        []uintptr
	pedalDetailGroup         []uintptr
	twistLeftDetailGroup     []uintptr
	twistRightDetailGroup    []uintptr
	controlHOTASGroup        []uintptr
	controlWheelGroup        []uintptr
	drivingWheelDetails      [4][]uintptr
	handbrakeTitle           uintptr
	pedalDeviceLabel         uintptr
	handbrakeShiftHandles    []uintptr
	drivingShiftHandles      []uintptr
	drivingShifted           bool
	handbrakeButtonGroup     []uintptr
	hotasBrakeDigitalGroup   []uintptr
	hotasBrakeAnalogGroup    []uintptr
	hosasDrivingGroup        []uintptr
	cameraTestGroup          []uintptr
	pedalFineGroup           []uintptr
	pedalCameraGroup         []uintptr
	pedalKeyGroup            []uintptr
	twistLeftFineGroup       []uintptr
	twistLeftCameraGroup     []uintptr
	twistLeftKeyGroup        []uintptr
	twistRightFineGroup      []uintptr
	twistRightCameraGroup    []uintptr
	twistRightKeyGroup       []uintptr
	onFootThumbGroup         []uintptr
	onFootButtonGroup        []uintptr

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
	procBeginDeferWindowPos      = user32.NewProc("BeginDeferWindowPos")
	procDeferWindowPos           = user32.NewProc("DeferWindowPos")
	procEndDeferWindowPos        = user32.NewProc("EndDeferWindowPos")
	procRedrawWindow             = user32.NewProc("RedrawWindow")
	procSetTimer                 = user32.NewProc("SetTimer")
	procKillTimer                = user32.NewProc("KillTimer")
	procKeybdEvent               = user32.NewProc("keybd_event")
	procLoadCursorW              = user32.NewProc("LoadCursorW")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procScreenToClient           = user32.NewProc("ScreenToClient")
	procGetClientRect            = user32.NewProc("GetClientRect")
	procSetScrollRange           = user32.NewProc("SetScrollRange")
	procSetScrollPos             = user32.NewProc("SetScrollPos")
	procShowScrollBar            = user32.NewProc("ShowScrollBar")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procSetFocus                 = user32.NewProc("SetFocus")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject           = gdi32.NewProc("GetStockObject")
	procSetBkColor               = gdi32.NewProc("SetBkColor")
	procGetSysColorBrush         = user32.NewProc("GetSysColorBrush")
	procGetSysColor              = user32.NewProc("GetSysColor")
	procCreateFontW              = gdi32.NewProc("CreateFontW")
	procInitCommonControlsEx     = comctl32.NewProc("InitCommonControlsEx")

	// WinMM device enumeration/capability probing is intentionally isolated in
	// GalacticRacerWinMMHelper.exe. The main process only polls known-good IDs.
	procJoyGetPosEx = winmm.NewProc("joyGetPosEx")

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

// The main chrome uses the chosen lighter neutral grey, while the native
// tab frame/content and Windows widgets keep their system theme colors.
// The window-class brush stays alive until the window closes.
var mainBackgroundBrush uintptr

const mainBackgroundColor = 0x00D4D4D4 // Win32 COLORREF: RGB(212,212,212) / #D4D4D4

func isRightTabTextControl(a *appState, ctrl uintptr) bool {
	for _, h := range a.mappingTabGroup {
		if h == ctrl {
			return true
		}
	}
	for _, h := range a.calibrationTabGroup {
		if h == ctrl {
			return true
		}
	}
	for _, h := range a.wheelFFBTabGroup {
		if h == ctrl {
			return true
		}
	}
	return false
}

var buttonOutputs = []struct {
	Key, Label string
	Mask       uint16 // Xbox button mask; zero for keyboard-only outputs.
	VK         uint16 // Windows virtual key; zero for Xbox-only outputs.
}{
	{"A", "Xbox A", XUSB_A, 0}, {"B", "Xbox B", XUSB_B, 0},
	{"X", "Xbox X", XUSB_X, 0}, {"Y", "Xbox Y", XUSB_Y, 0},
	{"LB", "Left bumper", XUSB_LEFT_SHOULDER, 0}, {"RB", "Right bumper", XUSB_RIGHT_SHOULDER, 0},
	{"LT", "Left Trigger", 0, 0}, {"RT", "Right Trigger", 0, 0},
	{"BACK", "Back / View", XUSB_BACK, 0}, {"START", "Start / Menu", XUSB_START, 0},
	{"LTHUMB", "Left-stick click", XUSB_LEFT_THUMB, 0}, {"RTHUMB", "Right-stick click", XUSB_RIGHT_THUMB, 0},
	{"DUP", "D-pad Up", XUSB_DPAD_UP, 0}, {"DDOWN", "D-pad Down", XUSB_DPAD_DOWN, 0},
	{"DLEFT", "D-pad Left", XUSB_DPAD_LEFT, 0}, {"DRIGHT", "D-pad Right", XUSB_DPAD_RIGHT, 0},
	// Galactic Racer labels describe the equivalent Xbox control; Look outputs remain keyboard keys.
	{"PITCHUP", "Pitch Up / Left Stick ↑", 0, 0},
	{"PITCHDOWN", "Pitch Down / Left Stick ↓", 0, 0},
	{"LOOKLEFT", "Look Left / Right Stick ←", 0, VK_DELETE},
	{"LOOKRIGHT", "Look Right / Right Stick →", 0, VK_NEXT},
	{"LOOKBACK", "Look Back / Right Stick ↓", 0, VK_END},
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
		{"Tab", 0x09}, {"Shift", 0x10}, {"Left Shift", 0xA0}, {"Right Shift", 0xA1},
		{"Left Control", 0xA2}, {"Right Control", 0xA3},
		{"Alt", 0x12}, {"Left Alt", 0xA4}, {"Right Alt", 0xA5},
		{"Enter", 0x0D}, {"Backspace", 0x08},
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

func checkViGEmConnectivity() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path := filepath.Join(filepath.Dir(exe), "ViGEmClient.dll")
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return fmt.Errorf("ViGEmClient.dll unavailable: %w", err)
	}
	defer dll.Release()
	alloc, err := dll.FindProc("vigem_alloc")
	if err != nil {
		return err
	}
	connect, err := dll.FindProc("vigem_connect")
	if err != nil {
		return err
	}
	disconnect, err := dll.FindProc("vigem_disconnect")
	if err != nil {
		return err
	}
	free, err := dll.FindProc("vigem_free")
	if err != nil {
		return err
	}
	client, _, _ := alloc.Call()
	if client == 0 {
		return fmt.Errorf("vigem_alloc failed")
	}
	defer free.Call(client)
	result, _, _ := connect.Call(client)
	if uint32(result) != 0x20000000 {
		return fmt.Errorf("vigem_connect returned 0x%X", uint32(result))
	}
	disconnect.Call(client)
	return nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--check-vigem" {
		// Installer's functional preflight: use the same shipped DLL and
		// driver connection path without creating an Xbox controller.
		err := checkViGEmConnectivity()
		if err != nil {
			fmt.Fprintln(os.Stderr, "ViGEmBus not ready:", err)
			os.Exit(2)
		}
		fmt.Println("ViGEmBus ready: ViGEmClient successfully connected")
		return
	}
	runtime.LockOSThread()
	icc := initCommonControlsEx{DwSize: uint32(unsafe.Sizeof(initCommonControlsEx{})), DwICC: 0x000000FF}
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	app = &appState{
		controls:         make(map[int]uintptr),
		deviceComboIDs:   make(map[int][]int),
		deviceComboGUIDs: make(map[int][]string),
		baseRects:        make(map[uintptr]rect),
		contentWidth:     1680,
		contentHeight:    1110,
		axisNames:        []string{"X", "Y", "Z", "X Rotation (R)", "Y Rotation (U)", "Z Rotation (V)"},
		tooltipHelp:      make(map[uintptr]string),
		keyboardDown:     make(map[uint16]bool),
		stopCh:           make(chan struct{}),
	}
	app.xboxSlot.Store(-1)
	app.cfg = defaultConfig()
	loadConfig(&app.cfg)
	initAxisProfiles(&app.cfg)
	if err := app.loadProfiles(); err != nil {
		app.postStatus("Profile load warning: " + err.Error())
	}
	// v1.96.1: do not probe legacy WinMM joystick slots during normal Auto/DirectInput startup.
	// WinMM device-capability queries are isolated in GalacticRacerWinMMHelper.exe and
	// are only requested when compatibility mode actually needs them.
	app.devices = nil
	if normalizeInputBackend(app.cfg.InputBackend) == "WinMM" {
		app.devices = enumerateDevices()
	}

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	// Window background color is application-level chrome, not a recolor of
	// the native controls, so Windows' standard widgets retain their theme.
	mainBackgroundBrush, _, _ = procCreateSolidBrush.Call(mainBackgroundColor)
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
		HbrBackground: mainBackgroundBrush,
		LpszClassName: className,
		HIconSm:       smallIcon,
	}
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		panic(fmt.Sprintf("RegisterClassExW failed: %v", e))
	}
	style := uintptr(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_THICKFRAME | WS_VSCROLL | WS_HSCROLL)
	hwnd, _, e := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16ptr(appTitle))), style,
		CW_USEDEFAULT, CW_USEDEFAULT, 1860, 1120, 0, 0, hInstance, 0,
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

	app.pollWG.Add(1)
	go app.pollLoop()
	go runAudioRumble(app.stopCh, func(err error) { app.postStatus("Audio rumble unavailable: " + err.Error()) })
	tfStream.reportErr = func(err error) { app.postStatus(err.Error()) }

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
	tfStream.Set(false) // end the Trueforce session cleanly before exit
	app.releaseAllKeyboardOutputs()
	app.xbox.Stop()
	app.di.Stop()
	if mainBackgroundBrush != 0 {
		procDeleteObject.Call(mainBackgroundBrush)
		mainBackgroundBrush = 0
	}
}

func defaultConfig() config {
	return config{
		InputBackend:    "Auto",
		ControlMode:     "HOSAS",
		DrivingWheel:    defaultDrivingWheel(),
		HotasThrottleID: -1, HotasThrottleAxis: 1, HotasBrakeBehavior: "Digital at Idle", HotasBrakeThreshold: 5, HotasBrakeZone: 20,
		HotatBrakeBehavior: "Off", HotatBrakeThreshold: 5, HotatBrakeZone: 20,
		HotasStickID: -1, HotasStickXAxis: 0, HotasStickYAxis: 1,
		LeftID: 0, RightID: 1, PedalID: -1,
		TwistLeft: defaultAuxiliaryAxis(), TwistRight: defaultAuxiliaryAxis(),
		LeftAxis: 1, RightAxis: 1, PedalAxis: 0,
		PedalMode: "Off", Deadzone: 6,
		PreserveThrust:         true,
		PreserveThrustV2:       true,
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
		MonitorDuringGameplay:  false,
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
		// v2.0 beta default: enable the hybrid thrust model once for existing
		// configurations (v1.99 defaulted to Off). After migration, saved user
		// choices—INCLUDING an explicit Off—remain authoritative.
		if _, initialized := rawKeys["preserve_thrust_v2_initialized"]; !initialized {
			tmp.PreserveThrust = true
			tmp.PreserveThrustV2 = true
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
		if _, ok := rawKeys["hotas_brake_behavior"]; !ok {
			tmp.HotasBrakeBehavior = "Digital at Idle"
		}
		tmp.HotasBrakeBehavior = normalizeHotasBrakeBehavior(tmp.HotasBrakeBehavior)
		if _, ok := rawKeys["hotat_brake_behavior"]; !ok {
			tmp.HotatBrakeBehavior = "Off"
		}
		tmp.HotatBrakeBehavior = normalizeLeverBrakeBehavior(tmp.HotatBrakeBehavior)
		if _, ok := rawKeys["hotat_brake_threshold"]; !ok {
			tmp.HotatBrakeThreshold = 5
		}
		if tmp.HotatBrakeThreshold < 0 || tmp.HotatBrakeThreshold > 50 {
			tmp.HotatBrakeThreshold = 5
		}
		if _, ok := rawKeys["hotat_brake_zone"]; !ok {
			tmp.HotatBrakeZone = 20
		}
		if tmp.HotatBrakeZone < 1 || tmp.HotatBrakeZone > 50 {
			tmp.HotatBrakeZone = 20
		}
		if _, ok := rawKeys["hotas_brake_threshold"]; !ok {
			tmp.HotasBrakeThreshold = 5
		}
		if tmp.HotasBrakeThreshold < 0 || tmp.HotasBrakeThreshold > 50 {
			tmp.HotasBrakeThreshold = 5
		}
		if _, ok := rawKeys["hotas_brake_zone"]; !ok {
			tmp.HotasBrakeZone = 20
		}
		if tmp.HotasBrakeZone < 1 || tmp.HotasBrakeZone > 50 {
			tmp.HotasBrakeZone = 20
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
		normalizeDrivingWheel(&tmp.DrivingWheel, rawKeys["driving_wheel"] != nil, maxAxis)
		validateAuxiliaryAxis(&tmp.TwistLeft, rawKeys["twist_left"] != nil, maxAxis)
		validateAuxiliaryAxis(&tmp.TwistRight, rawKeys["twist_right"] != nil, maxAxis)
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
	if _, initialized := raw["preserve_thrust_v2_initialized"]; !initialized {
		d.PreserveThrust = true
		d.PreserveThrustV2 = true
	} else {
		readBool(raw, "preserve_thrust_v2_initialized", &d.PreserveThrustV2)
	}
	readInt(raw, "pedal_steer_strength", &d.PedalSteerStrength)
	readInt(raw, "camera_sensitivity", &d.CameraSensitivity)
	readInt(raw, "camera_deadzone", &d.CameraDeadzone)
	readInt(raw, "pedal_left_key", &d.PedalLeftKey)
	readInt(raw, "pedal_right_key", &d.PedalRightKey)
	readInt(raw, "pedal_key_threshold", &d.PedalKeyThreshold)
	if b, ok := raw["hotas_brake_behavior"]; ok {
		_ = json.Unmarshal(b, &d.HotasBrakeBehavior)
	}
	readInt(raw, "hotas_brake_threshold", &d.HotasBrakeThreshold)
	readInt(raw, "hotas_brake_zone", &d.HotasBrakeZone)
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
	d.HotasBrakeBehavior = normalizeHotasBrakeBehavior(d.HotasBrakeBehavior)
	d.HotatBrakeBehavior = normalizeLeverBrakeBehavior(d.HotatBrakeBehavior)
	if d.HotatBrakeThreshold < 0 || d.HotatBrakeThreshold > 50 {
		d.HotatBrakeThreshold = 5
	}
	if d.HotatBrakeZone < 1 || d.HotatBrakeZone > 50 {
		d.HotatBrakeZone = 20
	}
	if d.HotasBrakeThreshold < 0 || d.HotasBrakeThreshold > 50 {
		d.HotasBrakeThreshold = 5
	}
	if d.HotasBrakeZone < 1 || d.HotasBrakeZone > 50 {
		d.HotasBrakeZone = 20
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
	if b, ok := raw["twist_left"]; ok {
		_ = json.Unmarshal(b, &d.TwistLeft)
	}
	if b, ok := raw["twist_right"]; ok {
		_ = json.Unmarshal(b, &d.TwistRight)
	}
	maxAuxAxis := 7
	if normalizeInputBackend(d.InputBackend) == "WinMM" {
		maxAuxAxis = 5
	}
	validateAuxiliaryAxis(&d.TwistLeft, raw["twist_left"] != nil, maxAuxAxis)
	validateAuxiliaryAxis(&d.TwistRight, raw["twist_right"] != nil, maxAuxAxis)
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
	switch {
	case strings.Contains(u, "WHEEL"):
		return "WHEEL"
	case strings.Contains(u, "SINGLE"):
		return "SINGLE"
	case strings.Contains(u, "HOTAT"):
		return "HOTAT"
	case strings.Contains(u, "HOTAS"):
		return "HOTAS"
	default:
		return "HOSAS" // Backward-compatible existing Podracer profiles
	}
}
func normalizeHotasBrakeBehavior(s string) string {
	if strings.EqualFold(strings.TrimSpace(s), "Centered Stick") {
		return "Centered Stick"
	}
	return normalizeLeverBrakeBehavior(s)
}
func normalizeLeverBrakeBehavior(s string) string {
	ls := strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.Contains(ls, "analog") || strings.Contains(ls, "zone"):
		return "Analog Brake Zone"
	case strings.Contains(ls, "off") || strings.Contains(ls, "none"):
		return "Off"
	default:
		return "Digital at Idle"
	}
}

func normalizePedalMode(s string) string {
	ls := strings.ToLower(strings.TrimSpace(s))
	switch {
	case ls == "boost/drift" || ls == "boost drift":
		return "Boost/Drift"
	case ls == "boost":
		return "Boost"
	case ls == "drift":
		return "Drift"
	case strings.Contains(ls, "keyboard") || strings.Contains(ls, "key"):
		return "Keyboard keys"
	case strings.Contains(ls, "pitch") || strings.Contains(ls, "left stick y") || strings.Contains(ls, "ly"):
		return "Pitch"
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
		} else if n == 4 {
			*dst = "Pitch"
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
		if strings.TrimSpace(v.Backend) == "" {
			v.Backend = "WinMM" // Existing configs predate backend-aware learning.
		}
		v.Backend = normalizeBindingBackend(v.Backend)
		out[normalizeBindingKey(k)] = v
	}
	// v1.94 and earlier called the manual LT mapping "BRAKE". Preserve it.
	if _, exists := out["LT"]; !exists {
		if old, ok := out["BRAKE"]; ok {
			out["LT"] = old
		}
	}
	delete(out, "BRAKE")
	return out
}

func normalizeBindingBackend(s string) string {
	if strings.EqualFold(strings.TrimSpace(s), "DirectInput") || strings.EqualFold(strings.TrimSpace(s), "DI") {
		return "DirectInput"
	}
	return "WinMM"
}
func normalizeBindingKey(k string) string {
	s := strings.ToUpper(strings.TrimSpace(k))
	r := strings.NewReplacer(" ", "", "-", "", "_", "", "/", "")
	s = r.Replace(s)
	aliases := map[string]string{
		"XBOXA": "A", "XBOXB": "B", "XBOXX": "X", "XBOXY": "Y",
		"LEFTBUMPER": "LB", "RIGHTBUMPER": "RB", "L1": "LB", "R1": "RB",
		"LEFTTRIGGER": "LT", "RIGHTTRIGGER": "RT", "L2": "LT", "R2": "RT", "BRAKE": "LT",
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
	if err := app.persistProfiles(c); err != nil {
		app.postStatus("Profile save warning: " + err.Error())
	}
}

func winMMHelperPath() string {
	candidates := make([]string, 0, 3)
	if exe, err := os.Executable(); err == nil && exe != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "GalacticRacerWinMMHelper.exe"))
	}
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		candidates = append(candidates, filepath.Join(base, "GalacticRacerHOSAS", "GalacticRacerWinMMHelper.exe"))
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		candidates = append(candidates, filepath.Join(wd, "GalacticRacerWinMMHelper.exe"))
	}
	seen := map[string]bool{}
	for _, p := range candidates {
		p = filepath.Clean(p)
		k := strings.ToLower(p)
		if seen[k] {
			continue
		}
		seen[k] = true
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

type winMMHelperCountReply struct {
	Count int `json:"count"`
}

type winMMHelperCapsReply struct {
	OK         bool   `json:"ok"`
	Result     uint32 `json:"result"`
	Name       string `json:"name"`
	Xmin       uint32 `json:"xmin"`
	Xmax       uint32 `json:"xmax"`
	Ymin       uint32 `json:"ymin"`
	Ymax       uint32 `json:"ymax"`
	Zmin       uint32 `json:"zmin"`
	Zmax       uint32 `json:"zmax"`
	Rmin       uint32 `json:"rmin"`
	Rmax       uint32 `json:"rmax"`
	Umin       uint32 `json:"umin"`
	Umax       uint32 `json:"umax"`
	Vmin       uint32 `json:"vmin"`
	Vmax       uint32 `json:"vmax"`
	NumButtons uint32 `json:"num_buttons"`
	Caps       uint32 `json:"caps"`
	MaxAxes    uint32 `json:"max_axes"`
	NumAxes    uint32 `json:"num_axes"`
	MaxButtons uint32 `json:"max_buttons"`
}

func runWinMMHelper(args ...string) ([]byte, error) {
	path := winMMHelperPath()
	if path == "" {
		return nil, fmt.Errorf("GalacticRacerWinMMHelper.exe was not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	// Keep the compatibility helper invisible. If a legacy WinMM driver corrupts
	// the helper process, Windows terminates only this child, not the main app.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("WinMM helper timed out")
	}
	return out, err
}

func enumerateDevices() []deviceInfo {
	// All joyGetDevCapsW calls live in the helper process. We intentionally query
	// one slot per helper invocation so a single broken/stale joystick driver can
	// crash that child without losing the main Galactic Racer application.
	b, err := runWinMMHelper("count")
	if err != nil {
		return nil
	}
	var count winMMHelperCountReply
	if json.Unmarshal(b, &count) != nil || count.Count <= 0 {
		return nil
	}
	if count.Count > 64 {
		count.Count = 64
	}
	out := make([]deviceInfo, 0)
	for i := 0; i < count.Count; i++ {
		b, err := runWinMMHelper("caps", strconv.Itoa(i))
		if err != nil {
			// A helper crash here is expected to be survivable. Skip only this slot.
			continue
		}
		var r winMMHelperCapsReply
		if json.Unmarshal(b, &r) != nil || !r.OK || r.Result != JOYERR_NOERROR {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = fmt.Sprintf("Joystick %d", i)
		}
		caps := joyCaps{
			Xmin: r.Xmin, Xmax: r.Xmax,
			Ymin: r.Ymin, Ymax: r.Ymax,
			Zmin: r.Zmin, Zmax: r.Zmax,
			Rmin: r.Rmin, Rmax: r.Rmax,
			Umin: r.Umin, Umax: r.Umax,
			Vmin: r.Vmin, Vmax: r.Vmax,
			NumButtons: r.NumButtons, Caps: r.Caps,
			MaxAxes: r.MaxAxes, NumAxes: r.NumAxes, MaxButtons: r.MaxButtons,
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
	// Physical input diagnostics are independent from gameplay processing.
	// Sample no faster than the 10Hz GUI: avoid doubling device reads at
	// 500/1000Hz polling, including while the game monitor is disabled.
	var lastRawRead time.Time
	var walkingButtonWasPressed bool // poll-goroutine-only edge detector
	var cycleButtonWasPressed bool   // don't repeat while a button is held
	var lastRawMode string
	var rawCache struct{ throttle, stickX, stickY, leftY, rightY, leftX, rightX, wheelX, wheelAccel, wheelBrake float64 }

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.profileSwitchMu.RLock()
			a.cfgMu.RLock()
			c := a.cfg
			a.cfgMu.RUnlock()
			desiredInterval := pollingInterval(c.PollingHz)
			if desiredInterval != currentInterval {
				ticker.Stop()
				ticker = time.NewTicker(desiredInterval)
				currentInterval = desiredInterval
			}

			wheelFFBOn := normalizeControlMode(c.ControlMode) == "WHEEL" && a.mappingActive.Load()
			rumblePct := ffbPercent(c.WheelRumbleStrength, defaultWheelRumbleStrength)
			audioPct := ffbPercent(c.WheelAudioStrength, defaultWheelAudioStrength)
			centerPct := ffbPercent(c.WheelCenteringStrength, defaultWheelCenteringStrength)
			tfCenterPct.Store(int32(centerPct))
			tfRumblePct.Store(int32(rumblePct))
			tfAudioPct.Store(int32(audioPct))
			// Before UpdateWheelFFB: ending a Trueforce session clears the wheel's
			// effects, so it must finish before DirectInput effects are recreated.
			tfStream.Set(wheelFFBOn && c.WheelTrueforce)
			if err := a.di.UpdateWheelFFB(c.DrivingWheel.Wheel.DIGUID, c.DrivingWheel.Wheel.Axis, wheelFFBOn, c.WheelTrueforce,
				rumblePct, audioPct, centerPct); err != nil {
				a.postStatus("Wheel force feedback unavailable: " + err.Error())
			}

			cycleBind, cycleBound, nextProfile := a.cycleSettings()
			ids := []int{}
			switch normalizeControlMode(c.ControlMode) {
			case "HOTAS":
				ids = append(ids, c.HotasThrottleID, c.HotasStickID)
			case "SINGLE":
				ids = append(ids, c.HotasStickID)
			case "WHEEL":
				for _, v := range []wheelAxisConfig{c.DrivingWheel.Wheel, c.DrivingWheel.Pedal1, c.DrivingWheel.Pedal2, c.DrivingWheel.Pedal3, c.DrivingWheel.Handbrake} {
					if v.DeviceID >= 0 {
						ids = append(ids, v.DeviceID)
					}
				}
			default:
				ids = append(ids, c.LeftID, c.RightID)
			}
			if normalizeControlMode(c.ControlMode) != "WHEEL" && c.PedalID >= 0 {
				ids = append(ids, c.PedalID)
			}
			for _, tw := range []auxiliaryAxisConfig{c.TwistLeft, c.TwistRight} {
				if tw.DeviceID >= 0 {
					ids = append(ids, tw.DeviceID)
				}
			}
			for _, b := range c.Bindings {
				if normalizeBindingBackend(b.Backend) != "DirectInput" && b.DeviceID >= 0 {
					ids = append(ids, b.DeviceID)
				}
			}
			if cycleBound && normalizeBindingBackend(cycleBind.Backend) != "DirectInput" && cycleBind.DeviceID >= 0 {
				ids = append(ids, cycleBind.DeviceID)
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

			diStates := make(map[string]diJoyState)
			diOK := make(map[string]bool)
			bindingsToSample := make([]binding, 0, len(c.Bindings)+1)
			for _, b := range c.Bindings {
				bindingsToSample = append(bindingsToSample, b)
			}
			if cycleBound {
				bindingsToSample = append(bindingsToSample, cycleBind)
			}
			for _, b := range bindingsToSample {
				if normalizeBindingBackend(b.Backend) != "DirectInput" || b.DIGUID == "" {
					continue
				}
				if _, exists := diOK[b.DIGUID]; exists {
					continue
				}
				st, _, ok := a.di.FullState(b.DIGUID)
				diStates[b.DIGUID] = st
				diOK[b.DIGUID] = ok
			}

			cyclePressed := cycleBound && bindingPressed(cycleBind, states, okmap, diStates, diOK)
			if cyclePressed && !cycleButtonWasPressed && nextProfile != 0 &&
				!a.mappingStarting.Load() && a.profileCyclePending.CompareAndSwap(false, true) {
				if posted, _, _ := procPostMessageW.Call(a.hwnd, WM_APP_PROFILE, uintptr(nextProfile), 0); posted == 0 {
					a.profileCyclePending.Store(false)
				}
			}
			// A backend restart during a profile change can momentarily lose the
			// button device; do not mistake missing hardware for button release.
			cycleDeviceAvailable := cycleBound && ((normalizeBindingBackend(cycleBind.Backend) == "DirectInput" || cycleBind.DIGUID != "") && diOK[cycleBind.DIGUID] ||
				(normalizeBindingBackend(cycleBind.Backend) != "DirectInput" && cycleBind.DIGUID == "" && okmap[cycleBind.DeviceID]))
			if !cycleBound || cycleDeviceAvailable {
				cycleButtonWasPressed = cyclePressed
			}
			walkingBind, bound := c.Bindings["WALKMODE"]
			buttonPressed := bound && (!cycleBound || !samePhysicalBinding(walkingBind, cycleBind)) && bindingPressed(walkingBind, states, okmap, diStates, diOK)
			if !a.mappingActive.Load() || !a.xboxReady.Load() {
				a.walkingActive.Store(false)
			} else if buttonPressed && !walkingButtonWasPressed {
				a.walkingActive.Store(!a.walkingActive.Load())
			}
			walkingButtonWasPressed = buttonPressed
			walking := a.walkingActive.Load() && a.mappingActive.Load() && a.xboxReady.Load()
			l, r := 0.0, 0.0
			hotasThrottle, hotasX, hotasY := 0.0, 0.0, 0.0
			hotasThrottleValid := false

			pedalRaw, p, cameraX := 0.0, 0.0, 0.0
			pedalValid := false
			var twistRaw [2]float64
			var twistValues [2]float64
			var twistValid [2]bool
			twistCfg := [2]auxiliaryAxisConfig{c.TwistLeft, c.TwistRight}
			controlMode := normalizeControlMode(c.ControlMode)
			isHOTAS := controlMode == "HOTAS"
			isSingle := controlMode == "SINGLE"
			isHOTAT := controlMode == "HOTAT"
			isWheel := controlMode == "WHEEL"
			backend := normalizeInputBackend(c.InputBackend)
			if isHOTAS || isSingle {
				if isHOTAS {
					if raw, ok := a.readConfiguredAxis(backend, c.HotasThrottleDIGUID, c.HotasThrottleID, c.HotasThrottleAxis, c.HotasThrottleInvert, states, okmap); ok {
						hotasThrottle = clamp((raw+1.0)/2.0, 0, 1)
						hotasThrottleValid = true
					}
				}
				if raw, ok := a.readConfiguredAxis(backend, c.HotasStickDIGUID, c.HotasStickID, c.HotasStickXAxis, c.HotasStickInvertX, states, okmap); ok {
					hotasX = tunedCentered(raw, profileFromConfig(c, profileHOTASX))
				}
				if raw, ok := a.readConfiguredAxis(backend, c.HotasStickDIGUID, c.HotasStickID, c.HotasStickYAxis, c.HotasStickInvertY, states, okmap); ok {
					hotasY = tunedCentered(raw, profileFromConfig(c, profileHOTASY))
				}
			} else if !isWheel {
				if isHOTAT {
					if raw, ok := a.readConfiguredAxis(backend, c.LeftDIGUID, c.LeftID, c.LeftAxis, c.LeftInvert, states, okmap); ok {
						l = tunedCentered(leverDrive((raw+1)/2, normalizeLeverBrakeBehavior(c.HotatBrakeBehavior), c.HotatBrakeThreshold, c.HotatBrakeZone), profileFromConfig(c, profileHOSASLeft))
					}
					if raw, ok := a.readConfiguredAxis(backend, c.RightDIGUID, c.RightID, c.RightAxis, c.RightInvert, states, okmap); ok {
						r = tunedCentered(leverDrive((raw+1)/2, normalizeLeverBrakeBehavior(c.HotatBrakeBehavior), c.HotatBrakeThreshold, c.HotatBrakeZone), profileFromConfig(c, profileHOSASRight))
					}
				} else {
					if c.LeftThrottleMode {
						if raw, ok := a.readConfiguredThrottleCentered(backend, c.LeftDIGUID, c.LeftID, c.LeftAxis, c.LeftInvert, states, okmap); ok {
							l = tunedCentered(raw, profileFromConfig(c, profileHOSASLeft))
						}
					} else if raw, ok := a.readConfiguredAxis(backend, c.LeftDIGUID, c.LeftID, c.LeftAxis, c.LeftInvert, states, okmap); ok {
						l = tunedCentered(raw, profileFromConfig(c, profileHOSASLeft))
					}
					if c.RightThrottleMode {
						if raw, ok := a.readConfiguredThrottleCentered(backend, c.RightDIGUID, c.RightID, c.RightAxis, c.RightInvert, states, okmap); ok {
							r = tunedCentered(raw, profileFromConfig(c, profileHOSASRight))
						}
					} else if raw, ok := a.readConfiguredAxis(backend, c.RightDIGUID, c.RightID, c.RightAxis, c.RightInvert, states, okmap); ok {
						r = tunedCentered(raw, profileFromConfig(c, profileHOSASRight))
					}

				}
			}
			if !isWheel {
				if raw, ok := a.readConfiguredAxis(backend, c.PedalDIGUID, c.PedalID, c.PedalAxis, c.PedalInvert, states, okmap); ok {
					pedalRaw = raw
					pedalValid = true
					p = tunedCentered(pedalRaw, profileFromConfig(c, profilePedal))
					cameraX = clamp(
						tunedValueAfterDeadzone(applyDeadzone(pedalRaw, c.CameraDeadzone), profileFromConfig(c, profilePedal))*float64(c.CameraSensitivity)/100.0,
						-1, 1,
					)
				}
			}

			for i, tw := range twistCfg {
				if tw.DeviceID < 0 && tw.DIGUID == "" {
					continue
				}
				if raw, ok := a.readConfiguredAxis(backend, tw.DIGUID, tw.DeviceID, tw.Axis, false, states, okmap); ok {
					twistRaw[i] = raw
					key := profileTwistRight
					if i == 0 {
						key = profileTwistLeft
					}
					twistValues[i] = tunedCentered(normalizeAuxiliaryValue(raw, tw, 0), profileFromConfig(c, key))
					twistValid[i] = true
				}
			}

			wheelSteer, accelPedal, brakePedal, clutchPedal, handbrakePedal := 0.0, 0.0, 0.0, 0.0, 0.0
			if isWheel {
				readWheel := func(v wheelAxisConfig, profile string, centered bool) float64 {
					raw, ok := a.readConfiguredAxis(backend, v.DIGUID, v.DeviceID, v.Axis, v.Invert, states, okmap)
					if !ok {
						return 0
					}
					if centered {
						return tunedCentered(raw, profileFromConfig(c, profile))
					}
					return tunedOneWay(clamp((raw+1)/2, 0, 1), profileFromConfig(c, profile))
				}
				wheelSteer = readWheel(c.DrivingWheel.Wheel, profileWheelSteering, true)
				accelPedal = readWheel(c.DrivingWheel.Pedal1, profileWheelPedal1, false)
				brakePedal = readWheel(c.DrivingWheel.Pedal2, profileWheelPedal2, false)
				clutchPedal = readWheel(c.DrivingWheel.Pedal3, profileWheelPedal3, false)
				if !c.DrivingWheel.HandbrakeButton {
					handbrakePedal = readWheel(c.DrivingWheel.Handbrake, profileWheelHandbrake, false)
				} else if b, ok := c.Bindings["WHEELHANDBRAKE"]; ok && (!cycleBound || !samePhysicalBinding(b, cycleBind)) && bindingPressed(b, states, okmap, diStates, diOK) {
					handbrakePedal = 1
				}
			}
			drive := tunedCentered(calculateDrive(l, r, c.PreserveThrust), profileFromConfig(c, profileHOSASThrust))
			steer := tunedCentered(clamp(l-r, -1, 1), profileFromConfig(c, profileHOSASSteer))
			hotasBrake := false
			hotasBrakeAmount := 0.0
			if isHOTAS {
				if hotasThrottleValid {
					drive = leverDrive(hotasThrottle, normalizeHotasBrakeBehavior(c.HotasBrakeBehavior), c.HotasBrakeThreshold, c.HotasBrakeZone)
				} else {
					drive = 0
				}
				if drive > 0 {
					drive = tunedOneWay(drive, profileFromConfig(c, profileHOTASThrottle))
				}
				hotasBrakeAmount = math.Max(0, -drive)
				hotasBrake = hotasBrakeAmount > 0.001
				steer = hotasX
			}
			if isSingle {
				drive = hotasY // -1..+1, centered by existing joystick calibration
				steer = hotasX
			}
			if isWheel {
				drive = accelPedal
				steer = wheelSteer
			}
			if !isWheel && strings.EqualFold(c.PedalMode, "Fine steering") {
				steer = clamp(steer+p*float64(c.PedalSteerStrength)/100.0, -1, 1)
			}
			pitchValue, pitchAssigned := 0.0, false
			cameraValue, cameraAssigned := 0.0, false
			if pedalValid && !isWheel {
				if strings.EqualFold(c.PedalMode, "Pitch") {
					pitchValue += p
					pitchAssigned = true
				}
				if strings.EqualFold(c.PedalMode, "Move Camera") {
					cameraValue += cameraX
					cameraAssigned = true
				}
			}
			for i, tw := range twistCfg {
				if !twistValid[i] {
					continue
				}
				v := twistValues[i]
				switch normalizePedalMode(tw.Mode) {
				case "Fine steering":
					steer = clamp(steer+v*float64(tw.SteerStrength)/100, -1, 1)
				case "Pitch":
					pitchValue += v
					pitchAssigned = true
				case "Move Camera":
					key := profileTwistRight
					if i == 0 {
						key = profileTwistLeft
					}
					cameraValue += tunedValueAfterDeadzone(normalizeAuxiliaryValue(twistRaw[i], tw, tw.CameraDeadzone), profileFromConfig(c, key)) * float64(tw.CameraSensitivity) / 100
					cameraAssigned = true
				}
			}
			pitchValue = clamp(pitchValue, -1, 1)
			cameraValue = clamp(cameraValue, -1, 1)

			var buttonMask uint16
			if isWheel {
				buttonMask |= axisDigitalButtons(c.DrivingWheel.Pedal3Mode, clutchPedal)
				buttonMask |= axisDigitalButtons(c.DrivingWheel.HandbrakeMode, handbrakePedal)
			} else {
				buttonMask |= axisDigitalButtons(c.PedalMode, p)
			}
			for i, tw := range twistCfg {
				if !twistValid[i] {
					continue
				}
				buttonMask |= axisDigitalButtons(tw.Mode, twistValues[i])
			}
			var outputHighlights uint32
			manualLT, manualRT := false, false
			manualPitchUp, manualPitchDown := false, false
			keyboardWanted := make(map[uint16]bool)
			for i, bo := range buttonOutputs {
				pressed := false
				if b, ok := c.Bindings[bo.Key]; ok && (!bound || !samePhysicalBinding(b, walkingBind)) && (!cycleBound || !samePhysicalBinding(b, cycleBind)) {
					pressed = bindingPressed(b, states, okmap, diStates, diOK)
				}
				if pressed {
					outputHighlights |= uint32(1) << uint(i)
					if bo.Mask != 0 {
						buttonMask |= bo.Mask
					}
					if bo.Key == "LT" {
						manualLT = true
					}
					if bo.Key == "RT" {
						manualRT = true
					}
					if bo.Key == "PITCHUP" {
						manualPitchUp = true
					}
					if bo.Key == "PITCHDOWN" {
						manualPitchDown = true
					}
				}
				if bo.VK != 0 {
					keyboardWanted[bo.VK] = keyboardWanted[bo.VK] || (pressed && a.mappingActive.Load())
				}
			}
			if !isWheel && strings.EqualFold(c.PedalMode, "Keyboard keys") && a.mappingActive.Load() && (c.PedalID >= 0 || c.PedalDIGUID != "") {
				threshold := clamp(float64(c.PedalKeyThreshold)/100.0, 0.05, 0.95)
				if pedalRaw <= -threshold {
					keyboardWanted[uint16(c.PedalLeftKey)] = true
				} else if pedalRaw >= threshold {
					keyboardWanted[uint16(c.PedalRightKey)] = true
				}
			}

			if a.mappingActive.Load() {
				for i, tw := range twistCfg {
					if !twistValid[i] || !strings.EqualFold(tw.Mode, "Keyboard keys") {
						continue
					}
					v := normalizeAuxiliaryValue(twistRaw[i], tw, 0)
					threshold := clamp(float64(tw.KeyThreshold)/100, .05, .95)
					if v <= -threshold {
						keyboardWanted[uint16(tw.LeftKey)] = true
					}
					if v >= threshold {
						keyboardWanted[uint16(tw.RightKey)] = true
					}
				}
			}

			// v2.2: Walking Mode uses analog Xbox sticks, not synthesized WASD.
			// Do not fire race-specific pedal/twist keyboard macros while walking.
			if a.walkingActive.Load() && a.mappingActive.Load() {
				keyboardWanted = make(map[uint16]bool)
			}
			var onFootHighlights uint8 // retained as a zero-valued legacy monitor field

			a.applyKeyboardOutputs(keyboardWanted)

			var walkOutput walkingOutput
			if walking {
				readWalk := func(guid string, id, axis int, invert bool) float64 {
					v, _ := a.readConfiguredAxis(backend, guid, id, axis, invert, states, okmap)
					return v
				}
				var in walkingAxes
				switch controlMode {
				case "HOSAS", "HOTAT":
					in.LeftY = readWalk(c.LeftDIGUID, c.LeftID, c.LeftAxis, c.LeftInvert)
					in.RightY = readWalk(c.RightDIGUID, c.RightID, c.RightAxis, c.RightInvert)
					if controlMode == "HOSAS" {
						in.LeftX = readWalk(c.LeftDIGUID, c.LeftID, monitorHorizontalAxis(c.LeftAxis), false)
						in.RightX = readWalk(c.RightDIGUID, c.RightID, monitorHorizontalAxis(c.RightAxis), false)
					}
				case "HOTAS":
					in.Throttle = readWalk(c.HotasThrottleDIGUID, c.HotasThrottleID, c.HotasThrottleAxis, c.HotasThrottleInvert)
					fallthrough
				case "SINGLE":
					in.StickX = readWalk(c.HotasStickDIGUID, c.HotasStickID, c.HotasStickXAxis, c.HotasStickInvertX)
					in.StickY = readWalk(c.HotasStickDIGUID, c.HotasStickID, c.HotasStickYAxis, c.HotasStickInvertY)
				case "WHEEL":
					w := c.DrivingWheel
					in.WheelX = readWalk(w.Wheel.DIGUID, w.Wheel.DeviceID, w.Wheel.Axis, w.Wheel.Invert)
					if raw, ok := a.readConfiguredAxis(backend, w.Pedal1.DIGUID, w.Pedal1.DeviceID, w.Pedal1.Axis, w.Pedal1.Invert, states, okmap); ok {
						in.Accelerator = clamp((raw+1)/2, 0, 1)
					}
					if raw, ok := a.readConfiguredAxis(backend, w.Pedal2.DIGUID, w.Pedal2.DeviceID, w.Pedal2.Axis, w.Pedal2.Invert, states, okmap); ok {
						in.Brake = clamp((raw+1)/2, 0, 1)
					}
				}
				// Physical throttle axes get a neutral 45-55% band while walking.
				// Spring-centered HOSAS sticks and independent wheel pedals are unchanged.
				switch controlMode {
				case "HOTAT":
					in.LeftY = walkingThrottleNeutral(in.LeftY)
					in.RightY = walkingThrottleNeutral(in.RightY)
				case "HOTAS":
					in.Throttle = walkingThrottleNeutral(in.Throttle)
				case "HOSAS":
					if c.LeftThrottleMode {
						in.LeftY = walkingThrottleNeutral(in.LeftY)
					}
					if c.RightThrottleMode {
						in.RightY = walkingThrottleNeutral(in.RightY)
					}
				}
				walkOutput = calculateWalkingOutput(controlMode, in)
			}
			// The first refresh after a mode switch takes a new sample immediately;
			// otherwise reuse the 10Hz cached raw positions.
			active := a.mappingActive.Load()
			if (!active || c.MonitorDuringGameplay) && (time.Since(lastRawRead) >= 100*time.Millisecond || lastRawMode != controlMode) {
				rawCache = struct{ throttle, stickX, stickY, leftY, rightY, leftX, rightX, wheelX, wheelAccel, wheelBrake float64 }{}
				if isWheel {
					rawCache.wheelX, _ = a.readConfiguredAxis(backend, c.DrivingWheel.Wheel.DIGUID, c.DrivingWheel.Wheel.DeviceID, c.DrivingWheel.Wheel.Axis, false, states, okmap)
					rawCache.wheelAccel = -1 // unassigned pedals draw empty bars
					rawCache.wheelBrake = -1
					if v, ok := a.readConfiguredAxis(backend, c.DrivingWheel.Pedal1.DIGUID, c.DrivingWheel.Pedal1.DeviceID, c.DrivingWheel.Pedal1.Axis, false, states, okmap); ok {
						rawCache.wheelAccel = v
					}
					if v, ok := a.readConfiguredAxis(backend, c.DrivingWheel.Pedal2.DIGUID, c.DrivingWheel.Pedal2.DeviceID, c.DrivingWheel.Pedal2.Axis, false, states, okmap); ok {
						rawCache.wheelBrake = v
					}
				} else if isHOTAS || isSingle {
					if isHOTAS {
						rawCache.throttle = 0 // Centered Stick shows neutral when not assigned
						if normalizeHotasBrakeBehavior(c.HotasBrakeBehavior) != "Centered Stick" {
							rawCache.throttle = -1
						}
						if v, ok := a.readConfiguredAxis(backend, c.HotasThrottleDIGUID, c.HotasThrottleID, c.HotasThrottleAxis, false, states, okmap); ok {
							rawCache.throttle = v
						}
					}
					rawCache.stickX, _ = a.readConfiguredAxis(backend, c.HotasStickDIGUID, c.HotasStickID, c.HotasStickXAxis, false, states, okmap)
					rawCache.stickY, _ = a.readConfiguredAxis(backend, c.HotasStickDIGUID, c.HotasStickID, c.HotasStickYAxis, false, states, okmap)
				} else {
					rawCache.leftY, _ = a.readConfiguredAxis(backend, c.LeftDIGUID, c.LeftID, c.LeftAxis, false, states, okmap)
					rawCache.rightY, _ = a.readConfiguredAxis(backend, c.RightDIGUID, c.RightID, c.RightAxis, false, states, okmap)
					if isHOTAT || c.LeftThrottleMode {
						if _, ok := a.readConfiguredAxis(backend, c.LeftDIGUID, c.LeftID, c.LeftAxis, false, states, okmap); !ok {
							rawCache.leftY = -1
						}
					}
					if isHOTAT || c.RightThrottleMode {
						if _, ok := a.readConfiguredAxis(backend, c.RightDIGUID, c.RightID, c.RightAxis, false, states, okmap); !ok {
							rawCache.rightY = -1
						}
					}
					rawCache.leftX, _ = a.readConfiguredAxis(backend, c.LeftDIGUID, c.LeftID, monitorHorizontalAxis(c.LeftAxis), false, states, okmap)
					rawCache.rightX, _ = a.readConfiguredAxis(backend, c.RightDIGUID, c.RightID, monitorHorizontalAxis(c.RightAxis), false, states, okmap)
				}
				lastRawRead = time.Now()
				lastRawMode = controlMode
			}
			leftX, leftY, rightX, rightY := rawCache.leftX, rawCache.leftY, rawCache.rightX, rawCache.rightY
			if isWheel {
				leftX, leftY = 0, rawCache.wheelAccel
				rightX, rightY = rawCache.wheelX, 0
			}
			if isHOTAS {
				leftX, leftY = 0, rawCache.throttle
				rightX, rightY = rawCache.stickX, rawCache.stickY
			}
			if isSingle {
				leftX, leftY = rawCache.stickX, rawCache.stickY
				rightX, rightY = 0, 0
			}

			a.monitorMu.Lock()
			a.monitor = monitorState{
				Left:             l,
				Right:            r,
				Pedal:            p,
				TwistLeft:        twistValues[0],
				TwistRight:       twistValues[1],
				Drive:            drive,
				Steer:            steer,
				Throttle:         hotasThrottle,
				HotasBrake:       hotasBrake,
				HotasBrakeAmount: hotasBrakeAmount,
				RawWheelThrottle: clamp((rawCache.wheelAccel+1)/2, 0, 1),
				RawWheelBrake:    clamp((rawCache.wheelBrake+1)/2, 0, 1),
				LeftX:            leftX,
				LeftY:            leftY,
				RightX:           rightX,
				RightY:           rightY,
				CameraX:          cameraX,
				OutputButtons:    buttonMask,
				OutputHighlights: outputHighlights,
				OnFootHighlights: onFootHighlights,
				WalkLX:           walkOutput.LX, WalkLY: walkOutput.LY,
				WalkRX: walkOutput.RX, WalkRY: walkOutput.RY,
				LastUpdate: time.Now(),
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
					if walking {
						rep.ThumbLX = floatToStick(walkOutput.LX)
						rep.ThumbLY = floatToStick(walkOutput.LY)
						rep.ThumbRX = floatToStick(walkOutput.RX)
						rep.ThumbRY = floatToStick(walkOutput.RY)
						// Only independently learned face/shoulder button assignments
						// remain active; no racing-trigger or pitch overrides.
						for _, bo := range buttonOutputs {
							if bo.Mask == 0 || bo.Key == "LT" || bo.Key == "RT" {
								continue
							}
							if b, ok := c.Bindings[bo.Key]; ok && (!bound || !samePhysicalBinding(b, walkingBind)) && (!cycleBound || !samePhysicalBinding(b, cycleBind)) && bindingPressed(b, states, okmap, diStates, diOK) {
								rep.Buttons |= bo.Mask
							}
						}
					} else {
						if isWheel {
							rep.RightTrigger, rep.LeftTrigger = drivingPedals(accelPedal, brakePedal)
							rep.ThumbLX = floatToStick(wheelSteer)
						} else if isHOTAS || isSingle {
							if drive >= 0 {
								rep.RightTrigger = byte(math.Round(clamp(drive, 0, 1) * 255))
							} else {
								rep.LeftTrigger = byte(math.Round(clamp(-drive, 0, 1) * 255))
							}
							rep.ThumbLX = floatToStick(steer)
							if isHOTAS {
								rep.ThumbLY = floatToStick(hotasY)
							}
						} else {
							if drive >= 0 {
								rep.RightTrigger = byte(math.Round(clamp(drive, 0, 1) * 255))
							} else {
								rep.LeftTrigger = byte(math.Round(clamp(-drive, 0, 1) * 255))
							}
							rep.ThumbLX = floatToStick(steer)
						}
						if cameraAssigned {
							rep.ThumbRX = floatToStick(cameraValue)
						}
						// Each configured Pitch source contributes to the analog Xbox Left Stick Y.
						// Centered Invert reverses direction; Lever Output Negative controls polarity.
						if pitchAssigned {
							rep.ThumbLY = floatToStick(pitchValue)
						}
						// Learned trigger mappings are independent of the analog drive calculation.
						if manualLT {
							rep.LeftTrigger = 255
						}
						if manualRT {
							rep.RightTrigger = 255
						}
						// Digital Pitch Up/Down mappings are full-deflection Left Stick Y outputs.
						// If both are held, they cancel to neutral.
						if manualPitchUp && !manualPitchDown {
							rep.ThumbLY = floatToStick(1.0)
						} else if manualPitchDown && !manualPitchUp {
							rep.ThumbLY = floatToStick(-1.0)
						} else if manualPitchUp && manualPitchDown {
							rep.ThumbLY = 0
						}
						rep.Buttons = buttonMask
					}
				}

				if cameraTest {
					rep.ThumbRX = floatToStick(cameraTestValue)
				}
				if testA {
					rep.Buttons |= XUSB_A
				}
				_ = a.xbox.Update(rep)
			}
			a.profileSwitchMu.RUnlock()
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

// samePhysicalBinding identifies a reserved Walking Mode button/hat across
// learned Xbox mappings, preventing duplicate in-game button presses.
func samePhysicalBinding(a, b binding) bool {
	if a.Kind == "" || b.Kind == "" || a.Kind != b.Kind {
		return false
	}
	if a.DIGUID != "" || b.DIGUID != "" {
		if a.DIGUID == "" || b.DIGUID == "" || a.DIGUID != b.DIGUID {
			return false
		}
	} else if a.DeviceID < 0 || a.DeviceID != b.DeviceID {
		return false
	}
	if a.Kind == "button" {
		return a.Button == b.Button
	}
	if a.Kind == "pov" {
		return a.POVIndex == b.POVIndex && a.POV == b.POV
	}
	return false
}

func bindingPressed(b binding, states map[int]joyInfoEx, oks map[int]bool, diStates map[string]diJoyState, diOK map[string]bool) bool {
	if normalizeBindingBackend(b.Backend) == "DirectInput" || b.DIGUID != "" {
		if b.DIGUID == "" || !diOK[b.DIGUID] {
			return false
		}
		st := diStates[b.DIGUID]
		if strings.EqualFold(b.Kind, "button") {
			if b.Button < 0 || b.Button >= len(st.Buttons) {
				return false
			}
			return st.Buttons[b.Button]&0x80 != 0
		}
		if strings.EqualFold(b.Kind, "pov") {
			idx := b.POVIndex
			if idx < 0 || idx >= len(st.POV) {
				idx = 0
			}
			v := st.POV[idx]
			if v == 0xFFFFFFFF {
				return false
			}
			return angleDistance(int(v), b.POV) <= 2300
		}
		return false
	}

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
	a.walkingActive.Store(false) // always begin a new virtual controller in Racing Mode
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
		a.postStatus(a.mappingStatus())
	} else {
		a.postStatus(a.mappingStatus())
	}
	procPostMessageW.Call(a.hwnd, WM_APP_XBOX, 0, 0)
}

func (a *appState) stopMapping() {
	// Stop output first so the polling loop no longer submits reports while the target is removed.
	a.mappingActive.Store(false)
	a.walkingActive.Store(false)
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
	// Game rumble for the wheel; optional so older ViGEmClient builds still work.
	if reg, err := dll.FindProc("vigem_target_x360_register_notification"); err == nil {
		if r, _, _ = reg.Call(client, target, x360NotificationCallback, 0); uint32(r) == 0x20000000 {
			x.notifyUnregister, _ = dll.FindProc("vigem_target_x360_unregister_notification")
		}
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
	if x.target != 0 && x.notifyUnregister != nil {
		x.notifyUnregister.Call(x.target)
	}
	x.notifyUnregister = nil
	rumbleLarge.Store(0)
	rumbleSmall.Store(0)
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
	case WM_CTLCOLORSTATIC:
		// Win32 STATIC text may otherwise draw opaque *white* rectangles
		// behind labels, especially after a themed tab switch. Always set
		// the DC background mode explicitly and return the matching brush
		// for the relevant pane. This does not recolor real input widgets.
		procSetBkMode.Call(wParam, TRANSPARENT)
		if isRightTabTextControl(app, lParam) {
			face, _, _ := procGetSysColorBrush.Call(COLOR_BTNFACE)
			color, _, _ := procGetSysColor.Call(COLOR_BTNFACE)
			procSetBkColor.Call(wParam, color)
			return face
		}
		if mainBackgroundBrush != 0 {
			procSetBkColor.Call(wParam, mainBackgroundColor)
			return mainBackgroundBrush
		}
	case WM_NOTIFY:
		// SysTabControl32 is a real themed tab strip, not two push-buttons.
		// Programmatic selection in selectRightTab does not emit this notification.
		if lParam != 0 {
			hdr := (*tabNotifyHeader)(unsafe.Pointer(lParam))
			if hdr.HwndFrom == app.controls[idRightPanelTabs] && hdr.Code == TCN_SELCHANGE {
				i, _, _ := procSendMessageW.Call(hdr.HwndFrom, TCM_GETCURSEL, 0, 0)
				app.selectRightTab(int(i))
				return 0
			}
		}
	case WM_MOUSEWHEEL:
		app.closeCombo(false)
		app.handleMouseWheel(wParam)
		return 0
	case WM_SIZE:
		app.closeCombo(false)
		app.updateScrollbars()
		return 0
	case WM_VSCROLL:
		app.closeCombo(false)
		app.handleScroll(false, wParam)
		return 0
	case WM_HSCROLL:
		app.closeCombo(false)
		app.handleScroll(true, wParam)
		return 0
	case WM_TIMER:
		// Curve values affect the poll loop immediately, while file writes are
		// coalesced so dragging a slider does not hammer the config file.
		app.flushTuningWhenDue(false)
		app.refreshWalkingUI()
		app.refreshMonitor()
		if app.tuning != nil {
			app.tuning.refreshLiveInput()
		}
		app.refreshDirectInputMonitor()
		app.refreshHotasDirectInputMonitor()
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
	case WM_APP_PROFILE:
		app.switchProfile(int(wParam))
		app.profileCyclePending.Store(false)
		return 0
	case WM_APP_XBOX:
		app.refreshXboxUI()
		return 0
	case WM_CLOSE:
		procDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		app.flushTuningWhenDue(true)
		procKillTimer.Call(hwnd, 1)
		for _, font := range []uintptr{app.headerFont, app.actionFont, app.testFont} {
			if font != 0 {
				procDeleteObject.Call(font)
			}
		}
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

// Opt-in emphasis for the header and selected action buttons. All normal labels,
// mapping rows and hover-help text retain the exact v2.0.4 default font.
func createEmphasisFont(pixelHeight, weight int) uintptr {
	height := uint32(int32(-pixelHeight))
	font, _, _ := procCreateFontW.Call(uintptr(height), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1, 0, 0, 4, 0, uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
	return font
}
func applyEmphasisFont(h, f uintptr) {
	if h != 0 && f != 0 {
		procSendMessageW.Call(h, WM_SETFONT, f, 1)
	}
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

// All main-window selectors use native CBS_DROPDOWNLIST controls. The old
// capture-friendly child LISTBOX could sit behind embedded tuning panes or
// obscure controls; the OS owns the popup Z-order, focus, and keyboard behavior.
// The application's option/index/GUID maps retain exactly their old identities.
func (a *appState) createCombo(x, y, w, h, id int) uintptr {
	if a.comboOptions == nil {
		a.comboOptions = make(map[int][]string)
	}
	if a.comboSelected == nil {
		a.comboSelected = make(map[int]int)
	}
	a.comboOptions[id] = nil
	a.comboSelected[id] = 0
	popupHeight := h
	if popupHeight < 180 {
		popupHeight = 180
	}
	ctrl := a.createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST, x, y, w, popupHeight, id)
	// COMBOBOX CreateWindow height includes the open dropdown; virtual scroll
	// dimensions should use only the collapsed selector (not its floating list).
	if ctrl != 0 {
		a.baseRects[ctrl] = rect{Left: int32(x), Top: int32(y), Right: int32(x + w), Bottom: int32(y + 26)}
	}
	return ctrl
}
func (a *appState) isComboID(id int) bool {
	_, ok := a.comboOptions[id]
	return ok
}
func (a *appState) comboReset(id int) {
	if h := a.controls[id]; h != 0 {
		procSendMessageW.Call(h, CB_RESETCONTENT, 0, 0)
	}
	a.comboOptions[id] = nil
	a.comboSelected[id] = 0
}
func (a *appState) comboAdd(id int, text string) {
	a.comboOptions[id] = append(a.comboOptions[id], text)
	if h := a.controls[id]; h != 0 {
		procSendMessageW.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(text))))
		if len(a.comboOptions[id]) == 1 {
			procSendMessageW.Call(h, CB_SETCURSEL, 0, 0)
		}
	}
	if len(a.comboOptions[id]) == 1 {
		a.comboSelected[id] = 0
	}
}

// Called when scrolling/changing tabs. Close native popup lists without
// changing any selection or dispatching synthetic selection notifications.
func (a *appState) closeCombo(_ bool) {
	for id := range a.comboOptions {
		if h := a.controls[id]; h != 0 {
			procSendMessageW.Call(h, CB_SHOWDROPDOWN, 0, 0)
		}
	}
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
	// v1.96 deliberately keeps help inside the main application window. Native Win32
	// tooltip/dropdown popup windows can float over unrelated apps and can disappear
	// from window-only screen capture.
	a.tooltipHelp[control] = text
}

func (a *appState) tooltipForID(id int, text string) { a.addTooltip(a.controls[id], text) }

// updateHoverTooltip renders contextual help into the main window instead of a
// separate top-most tooltip window. This keeps help capture-friendly and prevents
// it from floating above unrelated applications.
func (a *appState) updateHoverTooltip() {
	if len(a.tooltipHelp) == 0 || a.hwnd == 0 || a.controls[idHelpText] == 0 {
		return
	}
	var pt point
	if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		return
	}
	var hovered uintptr
	help := ""
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
		if a.tooltipHover != 0 {
			setText(a.controls[idHelpText], "Help: hover over a control to see what it does.")
			a.tooltipHover = 0
		}
		return
	}
	if hovered != a.tooltipHover {
		setText(a.controls[idHelpText], "Help: "+help)
		a.tooltipHover = hovered
	}
}

// Keep the original v2.0.4 control positions, including the help strip and
// footer. Calculate scrollbar bounds from actual UI child positions instead
// of the stale 1680x1110 virtual canvas that showed phantom scrollbars.
func (a *appState) fitContentBounds() {
	maxRight, maxBottom := 0, 0
	for _, r := range a.baseRects {
		if int(r.Right) > maxRight {
			maxRight = int(r.Right)
		}
		if int(r.Bottom) > maxBottom {
			maxBottom = int(r.Bottom)
		}
	}
	a.contentWidth = maxRight + 20
	a.contentHeight = maxBottom + 20
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

func (a *appState) redrawAfterScroll() {
	procRedrawWindow.Call(a.hwnd, 0, 0, RDW_INVALIDATE|RDW_ERASE|RDW_ALLCHILDREN|RDW_UPDATENOW)
}

func (a *appState) repositionScrolledChildren() {
	// Batch every child move into one layout transaction. Moving each child independently
	// caused visible tearing/stale separator pixels while dragging the scrollbars.
	hdwp, _, _ := procBeginDeferWindowPos.Call(uintptr(len(a.baseRects)))
	for h, r := range a.baseRects {
		if h == 0 {
			continue
		}
		x := int(r.Left) - a.scrollX
		y := int(r.Top) - a.scrollY
		if hdwp != 0 {
			next, _, _ := procDeferWindowPos.Call(hdwp, h, 0, uintptr(x), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
			if next != 0 {
				hdwp = next
			}
		} else {
			procSetWindowPos.Call(h, 0, uintptr(x), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
		}
	}
	if hdwp != 0 {
		procEndDeferWindowPos.Call(hdwp)
	}
	a.redrawAfterScroll()
}

func (a *appState) handleMouseWheel(wParam uintptr) {
	delta := int(int16(hiword(wParam)))
	if delta == 0 {
		return
	}
	steps := delta / 120
	if steps == 0 {
		if delta > 0 {
			steps = 1
		} else {
			steps = -1
		}
	}
	var rc rect
	procGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&rc)))
	limit := a.contentHeight - int(rc.Bottom-rc.Top)
	if limit < 0 {
		limit = 0
	}
	// Wheel-up scrolls toward the top; three 20px logical lines per wheel notch.
	a.scrollY -= steps * 60
	if a.scrollY < 0 {
		a.scrollY = 0
	}
	if a.scrollY > limit {
		a.scrollY = limit
	}
	procSetScrollPos.Call(a.hwnd, SB_VERT, uintptr(a.scrollY), 1)
	a.repositionScrolledChildren()
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

func (a *appState) openHidHideSetup() {
	cli, client, ok := hidhidePaths()
	if !ok {
		a.postStatus("HidHide tools were not found in the default Program Files location. If you installed to a custom drive, open HidHide there manually. Opening the official releases page.")
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

// configuredInputDevice checks the selected device identity rather than just the
// combo index: a DirectInput GUID or a WinMM ID is required ("None" is invalid).
func (a *appState) configuredInputDevice(id int) bool {
	a.cfgMu.RLock()
	backend := normalizeInputBackend(a.cfg.InputBackend)
	a.cfgMu.RUnlock()
	return inputDeviceSelected(backend, comboDevice(id), comboGUID(id))
}

func (a *appState) updateMappedThumbstickLabel() {
	if a.controls[idMappedThumbstick] == 0 {
		return
	}
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()
	if c.ThumbDISideGUID == "" || c.ThumbDIForwardGUID == "" {
		setText(a.controls[idMappedThumbstick], "Mapped Thumbstick: Not assigned — use Detect Thumbstick")
		return
	}
	deviceName := strings.TrimSpace(c.ThumbDIDeviceName)
	if deviceName == "" {
		deviceName = "Saved DirectInput device"
	}
	side, forward := fmt.Sprintf("Axis %d", c.ThumbDISideAxis), fmt.Sprintf("Axis %d", c.ThumbDIForwardAxis)
	if c.ThumbDISideAxis >= 0 && c.ThumbDISideAxis < len(diAxisNames) {
		side = diAxisNames[c.ThumbDISideAxis]
	}
	if c.ThumbDIForwardAxis >= 0 && c.ThumbDIForwardAxis < len(diAxisNames) {
		forward = diAxisNames[c.ThumbDIForwardAxis]
	}
	setText(a.controls[idMappedThumbstick], fmt.Sprintf("Mapped Thumbstick: %s  |  Horizontal: %s  |  Forward: %s", deviceName, side, forward))
}
func (a *appState) configureInputMonitorBars(left, right, wheel bool) {
	for _, h := range a.monitorLeftGrid {
		if left {
			procShowWindow.Call(h, SW_HIDE)
		} else {
			procShowWindow.Call(h, SW_SHOW)
		}
	}
	for _, h := range a.monitorRightGrid {
		if right {
			procShowWindow.Call(h, SW_HIDE)
		} else {
			procShowWindow.Call(h, SW_SHOW)
		}
	}
	for _, h := range a.monitorLeftSides {
		if left {
			procShowWindow.Call(h, SW_HIDE)
		} else {
			procShowWindow.Call(h, SW_SHOW)
		}
	}
	for _, h := range a.monitorRightSides {
		if right {
			procShowWindow.Call(h, SW_HIDE)
		} else {
			procShowWindow.Call(h, SW_SHOW)
		}
	}
	for _, row := range []struct {
		id   int
		show bool
	}{
		{idLeftDot, !left}, {idRightDot, !right},
		{idLeftThrottleBar, left && !wheel}, {idRightThrottleBar, right},
		{idWheelThrottleBar, wheel}, {idWheelBrakeBar, wheel},
	} {
		if h := a.controls[row.id]; h != 0 {
			if row.show {
				procShowWindow.Call(h, SW_SHOW)
			} else {
				procShowWindow.Call(h, SW_HIDE)
			}
		}
	}
	setText(a.monitorLeftForwardAxis, "F")
	setText(a.monitorLeftBackwardAxis, "B")
	setText(a.monitorRightForwardAxis, "F")
	setText(a.monitorRightBackwardAxis, "B")
	if left {
		setText(a.monitorLeftForwardAxis, "100%")
		setText(a.monitorLeftBackwardAxis, "0%")
	}
	if right {
		setText(a.monitorRightForwardAxis, "100%")
		setText(a.monitorRightBackwardAxis, "0%")
	}
	if wheel {
		setText(a.monitorLeftForwardAxis, "100%")
		setText(a.monitorLeftBackwardAxis, "0%")
	}
	// Wheel steering uses horizontal axis only; its forward/back labels add no meaning.
	for _, h := range []uintptr{a.monitorRightForwardAxis, a.monitorRightBackwardAxis} {
		if wheel {
			procShowWindow.Call(h, SW_HIDE)
		} else {
			procShowWindow.Call(h, SW_SHOW)
		}
	}
}

func (a *appState) updateContextVisibility() {
	if !a.uiReady {
		return
	}
	controlModes := controlModeValues
	ci := comboIndex(idControlMode)
	if ci < 0 || ci >= len(controlModes) {
		ci = 0
	}
	cm := controlModes[ci]
	isHOSAS := cm == "HOSAS"
	isHOTAT := cm == "HOTAT"
	isHOTAS := cm == "HOTAS"
	isSingle := cm == "SINGLE"
	isWheel := cm == "WHEEL"
	a.setGroupVisible(a.controlHOSASGroup, isHOSAS || isHOTAT)
	a.setGroupVisible(a.controlHOTASGroup, isHOTAS || isSingle)
	a.setGroupVisible(a.controlWheelGroup, isWheel)
	// Reuse the Traditional HOTAS stick axis selectors without exposing a
	// redundant throttle row for Single Stick mode.
	if a.singleStickRowShifted != isSingle {
		shift := int32(-34)
		if !isSingle {
			shift = 34
		}
		for _, h := range a.hotasStickRow {
			if r, ok := a.baseRects[h]; ok {
				r.Top += shift
				r.Bottom += shift
				a.baseRects[h] = r
			}
		}
		a.singleStickRowShifted = isSingle
		a.repositionScrolledChildren()
	}

	for i, row := range []int{idWheelDevice, idWheelPedal1Device, idWheelPedal2Device, idWheelPedal3Device} {
		a.setGroupVisible(a.drivingWheelDetails[i], isWheel && a.configuredInputDevice(row))
	}
	if a.drivingShifted != isWheel {
		shift := int32(45)
		if !isWheel {
			shift = -45
		}
		for _, h := range a.drivingShiftHandles {
			r, ok := a.baseRects[h]
			if ok {
				r.Top += shift
				r.Bottom += shift
				a.baseRects[h] = r
			}
		}
		a.drivingShifted = isWheel
		a.repositionScrolledChildren()
	}
	if isWheel {
		setText(a.handbrakeTitle, "Hand Brake")
	} else {
		setText(a.handbrakeTitle, "Rudder Pedals")
	}
	// This shared row is called Hand Brake in Wheel mode. The old
	// "Pedal device" label is exclusive to the Rudder Pedals modes; keep
	// the device selector, analog axis and saved assignments intact.
	a.setGroupVisible([]uintptr{a.pedalDeviceLabel}, !isWheel)
	a.setGroupVisible(a.handbrakeButtonGroup, isWheel)
	a.setGroupVisible(a.hosasDrivingGroup, (isHOSAS || isHOTAT) && a.activeRightTab == 1)
	// Only the device selectors (and Detect buttons) remain visible when
	// a hardware input is not selected. Hidden controls keep their values.
	leftConfigured := a.configuredInputDevice(idLeftDevice)
	rightConfigured := a.configuredInputDevice(idRightDevice)
	throttleConfigured := a.configuredInputDevice(idHotasThrottleDevice)
	stickConfigured := a.configuredInputDevice(idHotasStickDevice)
	pedalConfigured := a.configuredInputDevice(idPedalDevice)
	twistLeftConfigured := a.configuredInputDevice(idTwistLeftDevice)
	twistRightConfigured := a.configuredInputDevice(idTwistRightDevice)
	a.setGroupVisible(a.hosasLeftDetailGroup, (isHOSAS || isHOTAT) && leftConfigured)
	a.setGroupVisible(a.hosasRightDetailGroup, (isHOSAS || isHOTAT) && rightConfigured)
	a.setGroupVisible(a.hotasThrottleDetailGroup, isHOTAS && throttleConfigured)
	a.setGroupVisible(a.hotasStickDetailGroup, (isHOTAS || isSingle) && stickConfigured)
	// Digital handbrakes do not need an analog device; Mode must still be selectable.
	a.setGroupVisible(a.pedalDetailGroup, pedalConfigured || (isWheel && isChecked(a.controls[idWheelHandbrakeButton])))
	a.setGroupVisible(a.twistLeftDetailGroup, twistLeftConfigured)
	a.setGroupVisible(a.twistRightDetailGroup, twistRightConfigured)
	brakeModes := brakeModeValues
	bi := comboIndex(idHotasBrakeBehavior)
	if bi < 0 || bi >= len(brakeModes) {
		bi = 1
	}
	bm := brakeModes[bi]
	a.setGroupVisible([]uintptr{a.controls[idHotasThrottleDevice]}, isHOTAS)
	// Non-throttle Single Stick row is a direct reuse of the HOTAS stick mapping.
	a.setGroupVisible(a.hotasStickRow[:2], (isHOTAS || isSingle))
	// Guard each independently; HOTAT always shows bars, never throttle checkboxes.
	a.setGroupVisible([]uintptr{a.controls[idLeftThrottleMode]}, isHOSAS && leftConfigured)
	a.setGroupVisible([]uintptr{a.controls[idRightThrottleMode]}, isHOSAS && rightConfigured)
	a.setGroupVisible(a.hotasThrottleRow, isHOTAS)
	a.setGroupVisible([]uintptr{a.hotasDILabel}, isHOTAS)
	a.setGroupVisible(a.hotatBrakeGroup, isHOTAT)
	hotatMode := a.selectedComboText(idHOTATBrakeBehavior)
	a.setGroupVisible(a.hotatBrakeDigitalGroup, isHOTAT && hotatMode == "Digital at Idle")
	a.setGroupVisible(a.hotatBrakeAnalogGroup, isHOTAT && hotatMode == "Analog Brake Zone")
	a.setGroupVisible(a.hotasBrakeDigitalGroup, isHOTAS && throttleConfigured && bm == "Digital at Idle")
	a.setGroupVisible(a.hotasBrakeAnalogGroup, isHOTAS && throttleConfigured && bm == "Analog Brake Zone")
	pedalModes := a.comboOptions[idPedalMode]
	pi := comboIndex(idPedalMode)
	if pi < 0 || pi >= len(pedalModes) {
		pi = 0
	}
	pm := pedalModes[pi]
	a.setGroupVisible(a.pedalFineGroup, !isWheel && pedalConfigured && strings.EqualFold(pm, "Fine steering"))
	a.setGroupVisible(a.pedalCameraGroup, !isWheel && pedalConfigured && strings.EqualFold(pm, "Move Camera"))
	a.setGroupVisible(a.pedalKeyGroup, !isWheel && pedalConfigured && strings.EqualFold(pm, "Keyboard keys"))
	cameraModes := []string{}
	if pedalConfigured {
		cameraModes = append(cameraModes, pm)
	}
	for _, row := range []struct {
		modeID             int
		configured         bool
		fine, camera, keys []uintptr
	}{
		{idTwistLeftMode, twistLeftConfigured, a.twistLeftFineGroup, a.twistLeftCameraGroup, a.twistLeftKeyGroup},
		{idTwistRightMode, twistRightConfigured, a.twistRightFineGroup, a.twistRightCameraGroup, a.twistRightKeyGroup},
	} {
		mode := a.selectedComboText(row.modeID)
		a.setGroupVisible(row.fine, row.configured && mode == "Fine steering")
		a.setGroupVisible(row.camera, row.configured && mode == "Move Camera")
		if row.configured {
			cameraModes = append(cameraModes, mode)
		}
		a.setGroupVisible(row.keys, row.configured && mode == "Keyboard keys")
	}
	// Diagnostic buttons require an active virtual controller. Selecting
	// Move Camera alone must not expose buttons that cannot send output.
	controllerActive := a.mappingActive.Load() && a.xboxReady.Load()
	a.setGroupVisible(a.cameraTestGroup, cameraTestControlsVisible(controllerActive, cameraModes...))
	a.setGroupVisible([]uintptr{a.controls[idTestA]}, controllerActive)
	// Negative polarity is a lever-only option. Invert continues to reverse travel.
	for _, twist := range []struct {
		axisTypeID, negativeID int
		configured             bool
	}{
		{idTwistLeftAxisType, idTwistLeftOutputNegative, twistLeftConfigured},
		{idTwistRightAxisType, idTwistRightOutputNegative, twistRightConfigured},
	} {
		a.setGroupVisible([]uintptr{a.controls[twist.negativeID]}, twist.configured && comboIndex(twist.axisTypeID) == 1)
	}

	if a.tuning != nil {
		a.tuning.refreshEntriesFromConfig()
	}
	switch {
	case isWheel:
		setText(a.monitorLeftHeading, "Throttle / Brake")
		setText(a.monitorRightHeading, "Wheel")
	case isHOTAS:
		setText(a.monitorLeftHeading, "Throttle Input")
		setText(a.monitorRightHeading, "Steering Stick")
	case isSingle:
		setText(a.monitorLeftHeading, "Single Stick")
		setText(a.monitorRightHeading, "Not Used")
	default:
		setText(a.monitorLeftHeading, "Left Input")
		setText(a.monitorRightHeading, "Right Input")
	}
	// Bar overlays retain separate output plots. Raw input visualizations
	// never inherit an invert checkbox, response curve, or brake mapping.
	leftBar := isHOTAT || (isHOSAS && isChecked(a.controls[idLeftThrottleMode])) || (isHOTAS && bm != "Centered Stick") || isWheel
	rightBar := isHOTAT || (isHOSAS && isChecked(a.controls[idRightThrottleMode]))
	a.configureInputMonitorBars(leftBar, rightBar, isWheel)

}

// selectRightTab toggles panel controls without resetting the active tab on mode changes.
func (a *appState) selectRightTab(tab int) {
	if tab < 0 || tab > 2 {
		tab = 0
	}
	a.closeCombo(false)
	if a.activeRightTab == 1 && tab != 1 {
		a.flushTuningWhenDue(true)
	}
	a.activeRightTab = tab
	if nativeTabs := a.controls[idRightPanelTabs]; nativeTabs != 0 {
		procSendMessageW.Call(nativeTabs, TCM_SETCURSEL, uintptr(tab), 0)
	}
	a.setGroupVisible(a.mappingTabGroup, tab == 0)
	a.setGroupVisible(a.wheelFFBTabGroup, tab == 2)
	for _, h := range a.calibrationTabGroup {
		if h != a.controls[idPreserve] {
			procShowWindow.Call(h, map[bool]uintptr{true: SW_SHOW, false: SW_HIDE}[tab == 1])
		}
	}
	if a.tuning != nil && a.tuning.hwnd != 0 {
		if tab == 1 {
			procShowWindow.Call(a.tuning.hwnd, SW_SHOW)
		} else {
			procShowWindow.Call(a.tuning.hwnd, SW_HIDE)
		}
	}
	if a.uiReady {
		a.updateContextVisibility()
	}
}

func (a *appState) flushTuningWhenDue(force bool) {
	if !a.tuningDirty || (!force && time.Now().Before(a.tuningSaveDue)) {
		return
	}
	a.tuningDirty = false
	saveConfig()
}

func (a *appState) createDrivingWheelUI() {
	// The third optional pedal stays above the Additional Analogue Inputs area.
	// The existing HOSAS/HOTAS layout and footer remain unchanged in those modes.
	rows := []struct {
		name                         string
		y                            int
		device, axis, invert, detect int
	}{
		{"Wheel", 72, idWheelDevice, idWheelAxis, idWheelInvert, idWheelDetect},
		{"Pedal 1 (Throttle)", 105, idWheelPedal1Device, idWheelPedal1Axis, idWheelPedal1Invert, idWheelPedal1Detect},
		{"Pedal 2 (Brake)", 138, idWheelPedal2Device, idWheelPedal2Axis, idWheelPedal2Invert, idWheelPedal2Detect},
		{"Pedal 3 (Clutch)", 171, idWheelPedal3Device, idWheelPedal3Axis, idWheelPedal3Invert, idWheelPedal3Detect},
	}
	a.controlWheelGroup = append(a.controlWheelGroup,
		a.createLabel("Input Devices", 20, 48, 160, 20),
		a.createLabel("Device", 155, 48, 240, 20),
		a.createLabel("Axis", 405, 48, 85, 20))
	for i, row := range rows {
		a.controlWheelGroup = append(a.controlWheelGroup,
			a.createLabel(row.name, 20, row.y+4, 129, 23),
			a.createCombo(150, row.y, 240, 260, row.device))
		detail := []uintptr{
			a.createCombo(405, row.y, 85, 220, row.axis),
			a.createCheck("Invert", 500, row.y+1, 75, 24, row.invert),
		}
		detect := a.createButton("Detect", 582, row.y-1, 96, 28, row.detect)
		a.drivingWheelDetails[i] = detail
		a.controlWheelGroup = append(a.controlWheelGroup, detail...)
		a.controlWheelGroup = append(a.controlWheelGroup, detect)
	}
	// Pedal 1/2 are fixed to acceleration/brake; only the optional third pedal is remappable.
	a.controlWheelGroup = append(a.controlWheelGroup,
		a.createLabel("Mode", 698, 175, 50, 22), a.createCombo(748, 171, 160, 170, idWheelPedal3Mode))
	// The Rudder Pedals row is reused as Hand Brake in this control mode.
	// Shifted row occupies y=433 after updateContextVisibility applies its 45px offset.
	a.handbrakeButtonGroup = append(a.handbrakeButtonGroup,
		a.createButton("Detect", 839, 433, 70, 27, idWheelHandbrakeDetect),
		a.createCheck("Button", 911, 433, 79, 25, idWheelHandbrakeButton),
		a.createButton("Learn Btn", 994, 433, 111, 27, idWheelHandbrakeLearn))
}

func (a *appState) createTwistUI(left bool, y, detailY int) {
	ids := auxiliaryIDs(left)
	name := "Additional Input 2"
	if left {
		name = "Additional Input 1"
	}
	a.createLabel(name, 20, y+4, 130, 23)
	a.createCombo(150, y, 232, 250, ids[0])
	detail := []uintptr{
		a.createCombo(392, y, 83, 210, ids[1]),
		a.createCheck("Invert", 483, y+1, 75, 24, ids[2]),
		a.createLabel("Type", 555, y+4, 38, 22),
		a.createCombo(594, y, 103, 120, ids[3]),
		a.createLabel("Mode", 709, y+4, 43, 22),
		a.createCombo(755, y, 352, 160, ids[5]),
	}
	// Separate from Invert; placed on the detail row without overlapping mode options.
	// Only visible for Lever axis type (see updateContextVisibility).
	a.createCheck("Output Negative", 915, detailY+1, 176, 24, ids[12])
	var fine, camera, keys []uintptr
	fine = append(fine,
		a.createLabel("Fine-steer authority", 150, detailY+4, 135, 24),
		a.createEdit("30", 290, detailY, 52, 25, ids[6]),
		a.createLabel("%", 347, detailY+4, 20, 22))
	camera = append(camera,
		a.createLabel("Camera sensitivity", 150, detailY+4, 132, 24),
		a.createEdit("100", 285, detailY, 54, 25, ids[7]),
		a.createLabel("%", 343, detailY+4, 20, 22),
		a.createLabel("Camera deadzone", 387, detailY+4, 120, 24),
		a.createEdit("6", 511, detailY, 52, 25, ids[8]),
		a.createLabel("%", 569, detailY+4, 20, 22))
	keys = append(keys,
		a.createLabel("Negative key", 150, detailY+4, 89, 24),
		a.createCombo(244, detailY, 160, 230, ids[9]),
		a.createLabel("Positive key", 418, detailY+4, 91, 24),
		a.createCombo(513, detailY, 160, 230, ids[10]),
		a.createLabel("Threshold %", 692, detailY+4, 98, 24),
		a.createEdit("20", 792, detailY, 54, 25, ids[11]))
	if left {
		a.twistLeftDetailGroup = detail
		a.twistLeftFineGroup, a.twistLeftCameraGroup, a.twistLeftKeyGroup = fine, camera, keys
	} else {
		a.twistRightDetailGroup = detail
		a.twistRightFineGroup, a.twistRightCameraGroup, a.twistRightKeyGroup = fine, camera, keys
	}
}

func (a *appState) createUI() {
	f, _, _ := procGetStockObject.Call(DEFAULT_GUI_FONT)
	a.font = f
	a.headerFont = createEmphasisFont(16, 700)
	a.actionFont = createEmphasisFont(16, 700)
	a.testFont = createEmphasisFont(16, 400)

	// One single-line heading keeps attribution near the title. We do not
	// change default fonts for any section headings or the help/tooltip bar.
	applyEmphasisFont(a.createLabel("Flight controls -> Xbox Controls Mapping Bridge for Galactic Racer     Designed by JaffaQuake (Andy N.)", 20, 12, 1090, 30), a.headerFont)
	// Five complete settings profiles live above Control Mode. Native
	// push-like check buttons show which slot is active without adding rows
	// to the existing Podracer/HOTAS input area.
	a.createLabel("Profiles", 830, 52, 90, 22)
	for i := 0; i < profileCount; i++ {
		a.createControl("BUTTON", fmt.Sprintf("%d", i+1), WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX|BS_PUSHLIKE,
			925+i*38, 45, 33, 27, idProfileFirst+i)
	}
	for i := 0; i < profileCount; i++ {
		a.createCheck("", 928+i*38, 77, 26, 24, idProfileIncludeFirst+i)
	}
	a.createLabel("Control mode", 830, 110, 95, 22)
	a.createCombo(925, 105, 185, 160, idControlMode)
	a.createLabel("Input API", 830, 143, 90, 22)
	a.createCombo(925, 137, 185, 140, idInputBackend)
	a.createButton("Refresh devices", 830, 168, 120, 27, idRefresh)
	a.createButton("HidHide Setup", 960, 168, 150, 27, idHidHideSetup)

	// Podracer Style controls. Hidden automatically when Traditional HOTAS is selected.
	a.controlHOSASGroup = append(a.controlHOSASGroup,
		a.createLabel("Input Devices", 20, 48, 160, 20),
		a.createLabel("Device", 155, 48, 240, 20),
		a.createLabel("Forward axis", 405, 48, 110, 20),
		a.createLabel("Left Input", 20, 76, 120, 24),
		a.createCombo(150, 72, 240, 260, idLeftDevice),
		a.createCombo(405, 72, 85, 220, idLeftAxis),
		a.createCheck("Invert", 500, 73, 75, 24, idLeftInvert),
		a.createCheck("Throttle axis", 580, 73, 115, 24, idLeftThrottleMode),
		a.createButton("Detect Left", 697, 70, 106, 28, idDetectLeft),
		a.createLabel("Right Input", 20, 110, 120, 24),
		a.createCombo(150, 106, 240, 260, idRightDevice),
		a.createCombo(405, 106, 85, 220, idRightAxis),
		a.createCheck("Invert", 500, 107, 75, 24, idRightInvert),
		a.createCheck("Throttle axis", 580, 107, 115, 24, idRightThrottleMode),
		a.createButton("Detect Right", 697, 104, 106, 28, idDetectRight))

	// HOTAT shares this single brake mode between both levers. Place it on its
	// own row BELOW Left/Right Input (y=72/106), and ABOVE Additional Analogue
	// Inputs (heading y=178). Never overlap Control Mode / Input API (x>=830).
	a.hotatBrakeGroup = append(a.hotatBrakeGroup,
		a.createLabel("Brake behavior", 20, 145, 115, 20),
		a.createCombo(150, 140, 240, 160, idHOTATBrakeBehavior))
	a.hotatBrakeDigitalGroup = append(a.hotatBrakeDigitalGroup,
		a.createLabel("Idle %", 405, 145, 62, 20), a.createEdit("5", 470, 140, 52, 25, idHOTATBrakeThreshold))
	a.hotatBrakeAnalogGroup = append(a.hotatBrakeAnalogGroup,
		a.createLabel("Zone %", 405, 145, 62, 20), a.createEdit("20", 470, 140, 52, 25, idHOTATBrakeZone))
	a.comboReset(idHOTATBrakeBehavior)
	for _, m := range []string{"Off", "Digital at Idle", "Analog Brake Zone"} {
		a.comboAdd(idHOTATBrakeBehavior, m)
	}
	a.hosasLeftDetailGroup = []uintptr{a.controls[idLeftAxis], a.controls[idLeftInvert], a.controls[idLeftThrottleMode]}
	a.hosasRightDetailGroup = []uintptr{a.controls[idRightAxis], a.controls[idRightInvert], a.controls[idRightThrottleMode]}

	brakeLabel := a.createLabel("Brake behavior", 580, 48, 105, 20)
	throttleRowLabel := a.createLabel("Throttle", 20, 76, 120, 24)
	stickRowLabel := a.createLabel("Stick", 20, 110, 120, 24)
	// Traditional HOTAS controls occupy the same space as the Podracer rows. Throttle behavior is configurable;
	// the selected flight stick maps directly to the virtual Xbox left thumbstick.
	a.controlHOTASGroup = append(a.controlHOTASGroup,
		a.createLabel("Input Devices", 20, 48, 160, 20),
		a.createLabel("Device", 155, 48, 220, 20),
		a.createLabel("X / Throttle axis", 405, 48, 120, 20),
		a.createLabel("Y axis", 500, 48, 70, 20),
		throttleRowLabel,
		a.createCombo(150, 72, 240, 260, idHotasThrottleDevice),
		a.createCombo(405, 72, 85, 220, idHotasThrottleAxis),
		a.createCheck("Invert", 500, 73, 75, 24, idHotasThrottleInvert),
		brakeLabel,
		a.createCombo(580, 72, 150, 160, idHotasBrakeBehavior),
		stickRowLabel,
		a.createCombo(150, 106, 240, 260, idHotasStickDevice),
		a.createCombo(405, 106, 85, 220, idHotasStickXAxis),
		a.createCombo(500, 106, 85, 220, idHotasStickYAxis),
		a.createCheck("Invert X", 590, 107, 78, 24, idHotasStickInvertX),
		a.createCheck("Invert Y", 670, 107, 78, 24, idHotasStickInvertY))

	a.hotasThrottleRow = []uintptr{throttleRowLabel, a.controls[idHotasThrottleDevice]}
	a.hotasStickRow = []uintptr{stickRowLabel, a.controls[idHotasStickDevice], a.controls[idHotasStickXAxis], a.controls[idHotasStickYAxis], a.controls[idHotasStickInvertX], a.controls[idHotasStickInvertY]}
	a.hotasThrottleDetailGroup = []uintptr{a.controls[idHotasThrottleAxis], a.controls[idHotasThrottleInvert], a.controls[idHotasBrakeBehavior], brakeLabel}
	a.hotasStickDetailGroup = []uintptr{a.controls[idHotasStickXAxis], a.controls[idHotasStickYAxis], a.controls[idHotasStickInvertX], a.controls[idHotasStickInvertY]}
	// Brake behavior header is contextual to the throttle; the other axis
	// headers remain static to describe the columns when any input is chosen.

	a.hotasBrakeDigitalGroup = append(a.hotasBrakeDigitalGroup,
		a.createLabel("Idle %", 735, 76, 48, 22),
		a.createEdit("5", 783, 72, 40, 25, idHotasBrakeThreshold))
	a.hotasBrakeAnalogGroup = append(a.hotasBrakeAnalogGroup,
		a.createLabel("Zone %", 735, 76, 48, 22),
		a.createEdit("20", 783, 72, 40, 25, idHotasBrakeZone))
	for _, h := range a.hotasBrakeDigitalGroup {
		a.controlHOTASGroup = append(a.controlHOTASGroup, h)
	}
	for _, h := range a.hotasBrakeAnalogGroup {
		a.controlHOTASGroup = append(a.controlHOTASGroup, h)
	}
	a.handbrakeTitle = a.createLabel("Rudder Pedals", 20, 154, 130, 20)
	a.pedalDeviceLabel = a.createLabel("Pedal device", 20, 182, 115, 24)
	a.createCombo(150, 178, 240, 260, idPedalDevice)
	a.createCombo(405, 178, 85, 220, idPedalAxis)
	a.createCheck("Invert", 500, 179, 75, 24, idPedalInvert)
	// Keep 25 logical pixels clear of the Refresh devices button (x=830).
	pedalUseLabel := a.createLabel("Mode", 585, 182, 45, 20)
	a.createCombo(635, 178, 170, 180, idPedalMode)

	a.pedalDetailGroup = []uintptr{a.controls[idPedalAxis], a.controls[idPedalInvert], a.controls[idPedalMode], pedalUseLabel}
	a.handbrakeShiftHandles = append(a.handbrakeShiftHandles, a.handbrakeTitle, a.pedalDeviceLabel, a.controls[idPedalDevice])
	a.handbrakeShiftHandles = append(a.handbrakeShiftHandles, a.pedalDetailGroup...)
	// Label "Pedal use" is also conditional on hardware selection; it must
	// hide together with the mode selector, not remain floating in the row.
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

	// Camera test buttons are directly beneath Rudder Pedals and appear only if
	// either twist axis or Rudder Pedals uses Move Camera.
	a.cameraTestGroup = append(a.cameraTestGroup,
		a.createButton("Camera Left", 845, 212, 122, 29, idTestCamLeft),
		a.createButton("Camera Right", 975, 212, 128, 29, idTestCamRight))

	// v2.2: two-field Walking Mode panel. The previous On-Foot/WASD setup
	// is retired; walking reuses the chosen control mode's existing hardware.
	a.createCheck("Walking Mode: OFF", 20, 258, 220, 28, idWalkingToggle)
	a.createLabel("Walking Toggle Button", 20, 302, 195, 22)
	a.createControl("STATIC", "Not assigned", WS_CHILD|WS_VISIBLE|SS_LEFT, 220, 304, 520, 24, idWalkingLabel)
	a.createButton("Learn", 765, 297, 95, 30, idWalkingLearn)
	a.createButton("Clear", 870, 297, 80, 30, idWalkingClear)
	// Global physical profile-cycle button (not copied into individual slots).
	a.createLabel("Cycle Profile Button", 20, 355, 195, 22)
	a.createControl("STATIC", "Not assigned", WS_CHILD|WS_VISIBLE|SS_LEFT, 220, 357, 520, 24, idCycleLabel)
	a.createButton("Learn", 765, 349, 95, 30, idCycleLearn)
	a.createButton("Clear", 870, 349, 80, 30, idCycleClear)

	a.createLabel("Live Input Monitor", 20, 508, 185, 20)
	a.createCheck("Monitor during gameplay", 325, 505, 240, 27, idMonitorDuringGameplay)
	a.createControl("STATIC", "Waiting for devices...", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 533, 545, 56, idMonitor)
	a.createControl("STATIC", "Virtual Xbox disconnected. Click Initiate Xbox Controller to create it.", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 593, 555, 45, idStatus)
	a.createControl("STATIC", "XInput readback: disconnected", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 643, 545, 28, idXInputReadback)
	hotasDI := a.createControl("STATIC", "HOTAS DirectInput live: waiting for selected devices...", WS_CHILD|WS_VISIBLE|SS_LEFT, 20, 745, 1085, 55, idHotasDIAxisMonitor)
	a.controlHOTASGroup = append(a.controlHOTASGroup, hotasDI)
	a.hotasDILabel = hotasDI

	const boxSize = 126
	leftBoxX, boxY := 600, 542
	rightBoxX := 770
	outputBoxX := 940
	a.monitorLeftHeading = a.createLabel("Left Input", leftBoxX+2, 506, 144, 20)
	a.monitorRightHeading = a.createLabel("Right Input", rightBoxX+2, 506, 144, 20)
	a.monitorOutputHeading = a.createLabel("Drive / Steer", outputBoxX+9, 506, 140, 20)
	for _, x := range []int{leftBoxX, rightBoxX, outputBoxX} {
		frame := a.createControl("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, x, boxY, boxSize, boxSize, 0)
		vLine := a.createControl("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, x+boxSize/2, boxY, 1, boxSize, 0)
		hLine := a.createControl("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, x, boxY+boxSize/2, boxSize, 1, 0)
		if x == leftBoxX {
			a.monitorLeftGrid = []uintptr{frame, vLine, hLine}
		}
		if x == rightBoxX {
			a.monitorRightGrid = []uintptr{frame, vLine, hLine}
		}
		forward := a.createLabel("F", x+boxSize/2-23, boxY-18, 48, 18)
		backward := a.createLabel("B", x+boxSize/2-23, boxY+boxSize+2, 48, 18)
		if x == leftBoxX {
			a.monitorLeftForwardAxis, a.monitorLeftBackwardAxis = forward, backward
		} else if x == rightBoxX {
			a.monitorRightForwardAxis, a.monitorRightBackwardAxis = forward, backward
		}
		sideL := a.createLabel("L", x-14, boxY+boxSize/2-8, 14, 18)
		sideR := a.createLabel("R", x+boxSize+3, boxY+boxSize/2-8, 14, 18)
		if x == leftBoxX {
			a.monitorLeftSides = []uintptr{sideL, sideR}
		}
		if x == rightBoxX {
			a.monitorRightSides = []uintptr{sideL, sideR}
		}
	}
	// Colored, truly circular plot markers: red input 1, blue input 2,
	// green combined output. The 12px bounds and motion remain unchanged.
	a.createLiveMonitorDot(leftBoxX+boxSize/2-6, boxY+boxSize/2-6, idLeftDot)
	a.createLiveMonitorDot(rightBoxX+boxSize/2-6, boxY+boxSize/2-6, idRightDot)
	a.createLiveMonitorDot(outputBoxX+boxSize/2-6, boxY+boxSize/2-6, idDriveSteerDot)
	// Native Windows vertical progress bars fill upward, mapping physical 0% to bottom.
	// Overlay a complete chart; the old crosshair grid and indicator are hidden.
	// Driving Wheel has two independent pedal bars in its first panel.
	a.createControl("msctls_progress32", "", WS_CHILD|PBS_VERTICAL|PBS_SMOOTH, leftBoxX+23, boxY, 75, boxSize, idLeftThrottleBar)
	a.createControl("msctls_progress32", "", WS_CHILD|PBS_VERTICAL|PBS_SMOOTH, rightBoxX+23, boxY, 75, boxSize, idRightThrottleBar)
	a.createControl("msctls_progress32", "", WS_CHILD|PBS_VERTICAL|PBS_SMOOTH, leftBoxX+15, boxY, 36, boxSize, idWheelThrottleBar)
	a.createControl("msctls_progress32", "", WS_CHILD|PBS_VERTICAL|PBS_SMOOTH, leftBoxX+72, boxY, 36, boxSize, idWheelBrakeBar)
	for _, id := range []int{idLeftThrottleBar, idRightThrottleBar, idWheelThrottleBar, idWheelBrakeBar} {
		procSendMessageW.Call(a.controls[id], 0x0400+6, 0, 100) // PBM_SETRANGE32
	}
	a.createControl("STATIC", "Side +0.00  F +0.00", WS_CHILD|WS_VISIBLE|SS_LEFT, leftBoxX, boxY+boxSize+22, 155, 42, idLeftVectorText)
	a.createControl("STATIC", "Side +0.00  F +0.00", WS_CHILD|WS_VISIBLE|SS_LEFT, rightBoxX, boxY+boxSize+22, 155, 42, idRightVectorText)
	a.createControl("STATIC", "Drive +0.00 Neutral\r\nSteer +0.00 Straight", WS_CHILD|WS_VISIBLE|SS_LEFT, outputBoxX-5, boxY+boxSize+20, 185, 38, idDriveSteerVectorText)

	// Keep the existing downstream 210px layout shift and use the newly
	// available space to place Twist Axis between the HOSAS/HOTAS rows and
	// Rudder Pedals. v1.98.2 lowers only the twist rows by 22 logical px,
	// balancing the space above and below without moving existing controls.
	// The layout engine uses baseRects to handle scrolling and window resizing.
	for h, r := range a.baseRects {
		originalTop := r.Top
		if originalTop < 154 {
			continue
		}
		// Use the vertical space reclaimed from left-panel Calibration.
		// Reserve y=855..910 for verbose HOTAS diagnostics BELOW all three plots.
		// Keep on-foot children/learn buttons together without reordering HWNDs.
		offset := 210
		if originalTop >= 250 && originalTop < 380 { // On-foot group and saved thumbstick label
			offset = 205
		} else if originalTop >= 500 && originalTop < 725 { // Live graphs and monitor diagnostics
			offset = 100
		} else if originalTop >= 725 && originalTop < 800 { // HOTAS axis diagnostics go BELOW the plots
			offset = 110
		}
		r.Top += int32(offset)
		r.Bottom += int32(offset)
		a.baseRects[h] = r
		procSetWindowPos.Call(h, 0, uintptr(r.Left), uintptr(r.Top), 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
	}
	// Help is kept at the bottom of the virtual layout, BELOW the HOTAS
	// diagnostics at y=855..910 and above the footer at y=1027. Create it
	// after the vertical-reflow loop so it cannot inherit an offset.
	// Custom-painted help card: white face, 1px border, inset text and a
	// restrained two-pixel grey shadow. Its content still updates with SetText.
	// It remains below the HOTAS diagnostics and above the footer controls.
	a.createHelpPanel(20, 940, 1090, 64)

	// Centered/Lever ranges are already explained by the Axis Type tooltip.
	beforeExtra := make(map[uintptr]bool, len(a.baseRects))
	for h := range a.baseRects {
		beforeExtra[h] = true
	}
	a.createLabel("Additional Analogue Inputs", 20, 178, 220, 22)
	a.createTwistUI(true, 207, 244)
	a.createTwistUI(false, 285, 322)
	for h := range a.baseRects {
		if !beforeExtra[h] {
			a.drivingShiftHandles = append(a.drivingShiftHandles, h)
		}
	}
	a.drivingShiftHandles = append(a.drivingShiftHandles, a.handbrakeShiftHandles...)
	a.createDrivingWheelUI()

	// Right-side mappings panel: adjacent to input configuration and live monitor,
	// not under them. 43px rows retain a two-line physical-button assignment.
	// The new sidebar bounds keep total content height near the main controls.
	const bindingX = 1145
	const bindingRowHeight = 41
	const bindingTop = 84
	// A real, OS-themed tab control supplies raised selected and recessed
	// inactive tabs with a continuous shared page border. It belongs to the
	// common window, not either page's hide/show group. Keep it underneath
	// the page content (created next) in sibling Z-order.
	panelTabs := a.createControl("SysTabControl32", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP,
		bindingX-13, 12, 640, 976, idRightPanelTabs)
	for i, name := range []string{"Mappings", "Input Curves/Axis Tuning", "Wheel Haptics"} {
		label := utf16ptr(name)
		item := tabControlItem{Mask: TCIF_TEXT, Text: label}
		procSendMessageW.Call(panelTabs, TCM_INSERTITEMW, uintptr(i), uintptr(unsafe.Pointer(&item)))
	}
	procSendMessageW.Call(panelTabs, TCM_SETCURSEL, 0, 0)

	// Native tab-frame border replaces the old free-standing vertical divider.
	// Respect TCM_ADJUSTRECT so the embedded calibration child cannot paint
	// over the right edge of the tab (as it did at 655px in v2.0.9).
	tabArea := rect{Left: 0, Top: 0, Right: 640, Bottom: 976}
	procSendMessageW.Call(panelTabs, TCM_ADJUSTRECT, 0, uintptr(unsafe.Pointer(&tabArea)))
	panelRight := bindingX - 13 + int(tabArea.Right)
	usableRight := panelRight - 12 // conservative themed-client inset
	axisWidth := usableRight - bindingX
	if axisWidth > 610 {
		axisWidth = 610
	}
	// Even on a theme with unusual padding, never give the embedded child
	// more width than the tab frame can contain.
	// The existing selector is 594px. A normal 640px tab supplies over
	// 610px inside its display rectangle; do not expand beyond measured
	// bounds if a different native theme reports less space.
	if axisWidth < 0 {
		axisWidth = 0
	}
	standardCount := len(buttonOutputs) - 5
	specificHeadingY := bindingTop + standardCount*bindingRowHeight + 12

	mappingBefore := make(map[uintptr]bool, len(a.baseRects))
	for h := range a.baseRects {
		mappingBefore[h] = true
	}
	a.createLabel("Standard Xbox Mappings", bindingX, 54, 455, 24)
	drawBinding := func(i, y int) {
		bo := buttonOutputs[i]
		a.createControl("BUTTON", bo.Label, WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON|BS_MULTILINE, bindingX, y, 145, 37, idLearnBase+200+i)
		a.createControl("STATIC", "Not assigned", WS_CHILD|WS_VISIBLE|SS_LEFT, bindingX+153, y+2, 305, 37, idLearnBase+100+i)
		a.createButton("Learn", bindingX+465, y+3, 62, 30, idLearnBase+i)
		a.createButton("Clear", bindingX+535, y+3, 62, 30, idLearnClearBase+i)
	}
	for i := 0; i < standardCount; i++ {
		drawBinding(i, bindingTop+i*bindingRowHeight)
	}
	a.createLabel("Galactic Racer Specific Mappings", bindingX, specificHeadingY, 455, 24)
	for i := standardCount; i < len(buttonOutputs); i++ {
		drawBinding(i, specificHeadingY+28+(i-standardCount)*bindingRowHeight)
	}

	for h := range a.baseRects {
		if !mappingBefore[h] {
			a.mappingTabGroup = append(a.mappingTabGroup, h)
		}
	}
	// Calibration tab: selector and Podracer-only Hybrid are no longer in the left panel.
	a.calibrationTabGroup = append(a.calibrationTabGroup,
		a.createLabel("Calibration", bindingX, 64, 240, 24),
		a.createLabel("Polling rate", bindingX, 105, 92, 22),
		a.createCombo(bindingX+100, 99, 136, 155, idPollingRate),
		a.createLabel("Hz", bindingX+244, 105, 28, 22))
	a.hosasDrivingGroup = append(a.hosasDrivingGroup,
		a.createCheck("Hybrid preserve thrust", bindingX+292, 99, 240, 27, idPreserve))
	a.calibrationTabGroup = append(a.calibrationTabGroup, a.controls[idPreserve])
	a.createEmbeddedAxisTuning(bindingX, 152, axisWidth, 807)
	// Wheel Haptics tab: force feedback strengths (wheel_ffb_windows.go).
	a.wheelFFBTabGroup = append(a.wheelFFBTabGroup,
		a.createLabel("Wheel Haptics", bindingX, 64, 240, 24),
		a.createLabel("Applies in Driving Wheel control mode while the virtual Xbox controller is active. 0% turns an effect off. Close SimHub/Trueforce plugins first: the wheel needs exclusive force feedback access.", bindingX, 92, axisWidth, 44))
	for i, f := range []struct {
		name string
		id   int
	}{{"Centering spring", idWheelCenteringStrength}, {"Game rumble", idWheelRumbleStrength}, {"Audio rumble", idWheelAudioStrength}} {
		y := 150 + i*40
		a.wheelFFBTabGroup = append(a.wheelFFBTabGroup,
			a.createLabel(f.name, bindingX, y+4, 150, 22),
			a.createEdit("0", bindingX+160, y, 50, 25, f.id),
			a.createLabel("%", bindingX+216, y+4, 20, 22))
	}
	a.wheelFFBTabGroup = append(a.wheelFFBTabGroup,
		a.createCheck("Send audio rumble to Trueforce (experimental; close G HUB)", bindingX, 270, axisWidth, 25, idWheelTrueforce))
	a.wheelFFBTabGroup = append(a.wheelFFBTabGroup,
		a.createLabel("Audio rumble listens only to Galactic Racer's own sound (voice chat and other apps are ignored) and starts automatically when the game is running. Lower in-game music volume if it drowns out the engine.", bindingX, 310, axisWidth, 44))
	// Start on Mappings regardless of the saved control mode.
	a.selectRightTab(0)

	// Right-align the three action buttons against the usable tab content edge.
	// Preserve each width, inter-button spacing, and the help/footer clearance.
	footerShift := axisWidth - (408 + 153)
	if footerShift < 0 {
		footerShift = 0
	}
	applyEmphasisFont(a.createButton("Links to new updates/socials", 20, 1027, 260, 34, idSocialLinks), a.actionFont)
	applyEmphasisFont(a.createButton("Test Xbox A", bindingX+footerShift, 1027, 110, 34, idTestA), a.testFont)
	applyEmphasisFont(a.createButton("Initiate Xbox Controller", bindingX+footerShift+118, 1027, 280, 34, idStart), a.actionFont)
	applyEmphasisFont(a.createButton("Save settings", bindingX+footerShift+408, 1027, 153, 34, idSave), a.actionFont)

	// Hover help.
	a.tooltipForID(idHOTATBrakeBehavior, "RACING: HOTAT shared left/right braking. Off = forward only; Digital at Idle = full brake under Idle %; Analog Brake Zone = proportional braking below Zone %. Walking instead uses a fixed 45-55% travel neutral zone (stand still), regardless of these racing settings.")
	for i := 0; i < profileCount; i++ {
		a.tooltipForID(idProfileFirst+i, fmt.Sprintf("Profile %d: select this independent configuration. First use copies the current slot. Switching while Xbox is active sends neutral output, clears held keyboard keys and applies the next profile without removing the virtual controller. Walking Mode resets OFF.", i+1))
		a.tooltipForID(idProfileIncludeFirst+i, fmt.Sprintf("Include Profile %d in physical button cycling. Uncheck to skip it while cycling; the profile remains saved and selectable using the numbered button. Inclusion settings persist across restarts.", i+1))
	}
	a.tooltipForID(idControlMode, "Podracer Style (HOSAS): two spring-centered sticks with differential steering. Podracer Style (HOTAT): two throttle levers with shared braking and differential steering. Traditional HOTAS: a throttle or centered throttle stick plus a separate steering stick. Single Stick: one joystick, X=steering, Y=acceleration/braking. Driving Wheel: steering wheel and independent accelerator/brake pedals.")
	a.tooltipForID(idInputBackend, "Axis input backend. Auto uses DirectInput first and falls back to WinMM when possible. DirectInput exposes X/Y/Z, Rx/Ry/Rz and Slider 1/2. WinMM keeps the legacy six-axis path and is isolated in a helper process so problematic legacy joystick drivers cannot take down the main app. Learned buttons/hats use the selected Input API. Existing WinMM bindings remain WinMM; DirectInput bindings are stored separately by device GUID.")
	a.tooltipForID(idHidHideSetup, "Optional double-input protection. Locates HidHide tools under your default Program Files directory, adds this mapper to its allowlist, and opens its configuration client. Custom installations on other drives might need to be opened and configured manually.")
	a.tooltipForID(idHotasThrottleDevice, "Select the Traditional HOTAS throttle lever or the joystick you use as a centered throttle. Brake Mode controls whether negative travel generates Xbox LT.")
	a.tooltipForID(idHotasThrottleAxis, "Physical throttle axis. Full aft maps to RT 0%; full forward maps to RT 100%. Use Invert if the direction is backwards.")
	a.tooltipForID(idHotasBrakeBehavior, "RACING only: Off, Digital at Idle, Analog Brake Zone, or Centered Stick (50% neutral). Digital applies full LT at idle; Analog reserves the bottom zone for proportional braking. Walking Mode always uses 45-55% throttle travel as stand-still, regardless of this brake choice or custom curve midpoint.")
	a.tooltipForID(idHotasBrakeThreshold, "Idle Brake Threshold %. Used only by Digital at Idle. At or below this throttle percentage, LT is 100%; above it, brake releases and RT follows the throttle normally.")
	a.tooltipForID(idHotasBrakeZone, "Brake Zone %. Used only by Analog Brake Zone. The bottom part of throttle travel becomes proportional LT braking: minimum = 100% brake, fading to 0% across the zone. Remaining travel is rescaled to full 0-100% RT.")
	a.tooltipForID(idHotasStickDevice, "Select the steering stick for Traditional HOTAS, or the sole joystick for Single Stick. Single Stick uses X to steer and centered Y to accelerate/brake.")
	a.tooltipForID(idHotasStickXAxis, "Physical flight-stick horizontal axis mapped to Xbox Left Stick X.")
	a.tooltipForID(idHotasStickYAxis, "In Traditional HOTAS this is Xbox Left Stick Y. In Single Stick it is centered acceleration (RT forward) and brake (LT backward).")
	a.tooltipForID(idDetectLeft, "Move the LEFT flight stick forward after clicking. The app identifies the device, forward/back axis, and inversion automatically.")
	a.tooltipForID(idDetectRight, "Move the RIGHT flight stick forward after clicking. The app identifies the device, forward/back axis, and inversion automatically.")
	a.tooltipForID(idLeftThrottleMode, "Treat the selected LEFT HOSAS axis as a throttle lever: 0% = -1, 50% = neutral (0), 100% = +1. Useful for non-spring-centered throttle hardware.")
	a.tooltipForID(idRightThrottleMode, "Treat the selected RIGHT HOSAS axis as a throttle lever: 0% = -1, 50% = neutral (0), 100% = +1. Useful for non-spring-centered throttle hardware.")
	a.tooltipForID(idRefresh, "Rescan Windows for connected controller devices and restart the DirectInput device list.")
	a.tooltipForID(idWalkingToggle, "Walking Mode: physical throttle travel 45-55% means STAND STILL; below 45% walks backward and above 55% walks forward, reaching full speed at either end. Applies to HOTAT, Traditional HOTAS and HOSAS axes marked Throttle. Racing brake settings and custom curve midpoints do NOT change this neutral zone. Spring-centered sticks and wheel pedals keep their own behavior. Walking resets OFF when the controller stops or profiles switch.")
	a.tooltipForID(idWalkingLearn, "Assign a physical button/hat from either connected controller to toggle Walking Mode ON/OFF, one change per button press.")
	a.tooltipForID(idWalkingClear, "Remove the Walking Mode toggle binding. The software toggle will still work.")
	a.tooltipForID(idWalkingLabel, "Assigned physical button or POV hat toggles walking once per press. Reserved from Xbox mappings. Do not assign the same control as Cycle Profile: profile cycling takes priority.")
	a.tooltipForID(idCycleLearn, "Learn a physical controller button or POV hat to cycle through checked profiles. Each fresh press advances one included slot (wrapping 5 to 1). This binding is GLOBAL across all five profiles and reserved from ordinary Xbox mappings.")
	a.tooltipForID(idCycleClear, "Remove the global Cycle Profile Button assignment. Profiles and inclusion checkboxes remain saved.")
	a.tooltipForID(idCycleLabel, "Cycle Profile Button: global hardware binding, independent of all five profiles. Checked profiles are visited in order; unchecked profiles are only skipped, never deleted. If none are checked, the button does nothing.")
	a.tooltipForID(idHOTATBrakeZone, "Analog Brake Zone controls RACING braking only, not Walking Mode. Walking always stands still at 45-55% throttle travel, walks backward below and forward above that range.")
	a.tooltipForID(idStart, "Creates the virtual Xbox controller and begins sending configured controller/keyboard outputs to the game. Deactivate Xbox Controller removes the virtual controller.")
	a.tooltipForID(idTestA, "Send a short Xbox A-button pulse through the virtual controller. Initiate Xbox Controller must be active.")
	a.tooltipForID(idTestCamLeft, "Force virtual Xbox Right Stick X fully left briefly. Useful for camera-output diagnostics.")
	a.tooltipForID(idTestCamRight, "Force virtual Xbox Right Stick X fully right briefly. Useful for camera-output diagnostics.")
	a.tooltipForID(idSave, "Save this profile's device, axis, polling, twist, pedal, Walking Mode and learned Xbox-button settings. Global Profile Cycle binding and inclusion checkboxes are saved in the profile collection.")
	a.tooltipForID(idRightPanelTabs, "Mappings contains Xbox button bindings and the GLOBAL Cycle Profile button. Input Curves/Axis Tuning adjusts racing response shapes and polling. Wheel Haptics sets centering and rumble strength for force feedback wheels. Two Stage Custom Midpoint determines where curve segments join; it does NOT move Walking Mode neutral, which stays fixed at 45-55% on physical throttle axes.")
	a.tooltipForID(idPreserve, "ON (default): when both inputs are forward, thrust follows the stronger input, preserving acceleration while steering. Pulling one input behind center progressively reduces thrust. Both inputs backward retain reverse/braking. OFF: thrust is the average of Left Input and Right Input, so turning by moving one stick less forward also reduces acceleration (e.g. Left 100%, Right 0% produces only 50% thrust).")
	a.tooltipForID(idPollingRate, "Native Windows dropdown: all five 100/200/250/500/1000 Hz choices are available. Controller loop polling frequency. Changes save to polling_hz in your existing settings file and are restored after installing an update. Higher rates may use more CPU; hardware/Windows update rates can still be lower.")
	a.tooltipForID(idSocialLinks, "Open https://linktr.ee/jaffaquake in your default web browser to find new updates and socials.")
	a.tooltipForID(idMonitorDuringGameplay, "OFF (default): suspend live input graphs and mapped-button highlights while the virtual Xbox controller is active. ON: continue showing the mode-specific input graphs, Drive/Steer and learned button presses during gameplay. Visuals update at 10 Hz regardless of controller polling rate. May cause a slight increase in CPU usage.")
	for _, left := range []bool{true, false} {
		ids := auxiliaryIDs(left)
		name := "Additional Input 2"
		if left {
			name = "Additional Input 1"
		}
		a.tooltipForID(ids[0], name+": choose a physical joystick/twist/lever device, or None to disable it.")
		a.tooltipForID(ids[1], "Select the physical analog axis (X/Y/Z/Rx/Ry/Rz/Slider) for this twist input.")
		a.tooltipForID(ids[2], "Invert axis direction. Centered reverses sign; Lever reverses 0-100% travel.")
		a.tooltipForID(ids[3], "Centered: -100..+100 around midpoint. Lever: 0..100 travel; enable Output Negative for 0..-100. Invert reverses travel independently.")
		a.tooltipForID(ids[4], "Independent twist deadzone percentage. For Lever, deadzone applies near 0% travel.")
		a.tooltipForID(ids[5], "Same modes as rudder pedals: Off, Fine steering, Move Camera, Pitch, Keyboard keys. Camera and Pitch are analog Xbox outputs; Keyboard keys emits native key presses.")
		a.tooltipForID(ids[9], "Keyboard key held when this twist input moves negative past threshold. Lever axes can use this with Output Negative enabled.")
		a.tooltipForID(ids[10], "Keyboard key held when this twist input moves positive past threshold.")
		a.tooltipForID(ids[12], "Lever mode only: output 0 to -100% instead of 0 to +100%. Invert still independently reverses lever travel. Does not change Centered behavior.")
	}
	a.tooltipForID(idPedalMode, "In Driving Wheel mode, Hand Brake Mode selects Boost (Xbox A) or Drift (Xbox X). In the other modes, choose how the rudder axis is used. Pitch maps the pedal axis continuously to Xbox Left Stick Y; positive pedal input is Pitch Up and Pedal Invert reverses the direction. Only settings relevant to the selected mode are shown.")
	a.tooltipForID(idPedalLeftKey, "Keyboard key held when the pedal axis moves left past the configured threshold. Includes Shift, left/right Control, Alt, Tab and other common keys.")
	a.tooltipForID(idPedalRightKey, "Keyboard key held when the pedal axis moves right past the configured threshold. Includes Shift, left/right Control, Alt, Tab and other common keys.")
	for i, bo := range buttonOutputs {
		a.tooltipForID(idLearnBase+i, "Click Learn, then press the controller button/hat/switch you want assigned to "+bo.Label+".")
		a.tooltipForID(idLearnBase+200+i, "Live indicator for "+bo.Label+". It highlights when the learned physical input is pressed while the visual monitor is active.")
		a.tooltipForID(idLearnClearBase+i, "Clear the saved "+bo.Label+" assignment and return it to Not assigned.")
		if bo.Key == "LT" {
			msg := "Optional manual Left Trigger mapping. Your analog controls already generate LT automatically when the selected control mode/brake behavior calls for braking or reverse. You do not need to assign this for normal braking; use it only if you also want a button/hat to send full LT on demand."
			a.tooltipForID(idLearnBase+i, msg)
			a.tooltipForID(idLearnBase+200+i, msg)
		}
		if bo.Key == "RT" {
			msg := "Optional manual Right Trigger mapping. Acceleration is already generated automatically by the analog Podracer Style or Traditional HOTAS controls. You do not need to assign this for normal thrust; use it only if you also want a button/hat to send full RT on demand."
			a.tooltipForID(idLearnBase+i, msg)
			a.tooltipForID(idLearnBase+200+i, msg)
		}
		if bo.Key == "PITCHUP" || bo.Key == "PITCHDOWN" {
			direction := "UP"
			if bo.Key == "PITCHDOWN" {
				direction = "DOWN"
			}
			msg := "Digital pitch mapping. While this learned button/hat is held, Xbox Left Stick Y is sent fully " + direction + ". Use Pedal mode Pitch instead when you want proportional analog pitch."
			a.tooltipForID(idLearnBase+i, msg)
			a.tooltipForID(idLearnBase+200+i, msg)
		}
	}

	for _, row := range []struct {
		device, axis, invert, detect int
		label                        string
	}{
		{idWheelDevice, idWheelAxis, idWheelInvert, idWheelDetect, "Wheel"},
		{idWheelPedal1Device, idWheelPedal1Axis, idWheelPedal1Invert, idWheelPedal1Detect, "Pedal 1 throttle"},
		{idWheelPedal2Device, idWheelPedal2Axis, idWheelPedal2Invert, idWheelPedal2Detect, "Pedal 2 brake"},
		{idWheelPedal3Device, idWheelPedal3Axis, idWheelPedal3Invert, idWheelPedal3Detect, "Optional Pedal 3"},
	} {
		a.tooltipForID(row.device, row.label+": select a physical USB wheel/pedal device (or None).")
		a.tooltipForID(row.axis, row.label+": choose X/Y/Z/Rx/Ry/Rz/Slider axis.")
		a.tooltipForID(row.invert, row.label+": reverse physical travel/direction.")
		a.tooltipForID(row.detect, row.label+": move this wheel or pedal strongly to automatically detect device, axis and direction.")
	}
	a.tooltipForID(idWheelCenteringStrength, "Centering spring strength while the virtual Xbox controller is active (0-100%). 0 turns it off, e.g. to use G HUB's centering instead.")
	a.tooltipForID(idWheelRumbleStrength, "Game rumble (boosts, crashes) played on the wheel (0-100%). 0 turns it off.")
	a.tooltipForID(idWheelTrueforce, "Logitech RS50 / G PRO only. Plays game audio on the wheel's Trueforce haptics instead of as force feedback, so it buzzes without pulling the rim. Centering and game rumble move into the same stream. G HUB and SimHub must be closed.")
	a.tooltipForID(idWheelAudioStrength, "Continuous rumble from the bass of Galactic Racer's own audio (0-100%). Needs Windows 10 build 20348 or newer. 0 turns it off.")
	a.tooltipForID(idWheelPedal3Mode, "Optional Pedal 3: Off, Boost (Xbox A), or Drift (Xbox X). Signals past the calibration deadzone send a full button press.")
	a.tooltipForID(idWheelHandbrakeButton, "Use a learned digital handbrake button instead of the analog Hand Brake axis. Pressed equals 100%.")
	a.tooltipForID(idWheelHandbrakeDetect, "Detect Hand Brake analog axis: pull the handbrake once. Use Button Input instead for a digital switch.")
	a.tooltipForID(idWheelHandbrakeLearn, "Learn a digital button/switch from a wheel or separate USB handbrake. Use Button Input enables it.")
	a.populateCombos()
	a.applyConfigToUI()
	a.refreshProfileButtons()
	a.updateBindingLabels()
	a.fitContentBounds()
	a.uiReady = true
	a.updateContextVisibility()
	procSetTimer.Call(a.hwnd, 1, 100, 0)
}

func (a *appState) populateCombos() {
	a.cfgMu.RLock()
	backend := normalizeInputBackend(a.cfg.InputBackend)
	a.cfgMu.RUnlock()

	deviceIDs := []int{idLeftDevice, idRightDevice, idPedalDevice, idHotasThrottleDevice, idHotasStickDevice, idTwistLeftDevice, idTwistRightDevice, idWheelDevice, idWheelPedal1Device, idWheelPedal2Device, idWheelPedal3Device}
	if backend == "WinMM" {
		a.devicesMu.RLock()
		devs := append([]deviceInfo(nil), a.devices...)
		a.devicesMu.RUnlock()
		for _, id := range deviceIDs {
			a.comboReset(id)
			ids := []int{}
			guids := []string{}
			a.comboAdd(id, "None")
			ids = append(ids, -1)
			guids = append(guids, "")
			for _, d := range devs {
				a.comboAdd(id, fmt.Sprintf("%s [WinMM ID %d]", d.Name, d.ID))
				ids = append(ids, d.ID)
				guids = append(guids, "")
			}
			a.deviceComboIDs[id] = ids
			a.deviceComboGUIDs[id] = guids
		}
	} else {
		devs := a.di.DeviceList()
		for _, id := range deviceIDs {
			a.comboReset(id)
			ids := []int{-1}
			guids := []string{""}
			a.comboAdd(id, "None")
			for _, d := range devs {
				a.comboAdd(id, fmt.Sprintf("%s [DirectInput]", d.Name))
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
	for _, id := range []int{idLeftAxis, idRightAxis, idPedalAxis, idHotasThrottleAxis, idHotasStickXAxis, idHotasStickYAxis, idTwistLeftAxis, idTwistRightAxis, idWheelAxis, idWheelPedal1Axis, idWheelPedal2Axis, idWheelPedal3Axis} {
		a.comboReset(id)
		for _, name := range axisList {
			a.comboAdd(id, name)
		}
	}

	a.comboReset(idInputBackend)
	for _, mode := range []string{"Auto (DirectInput first)", "DirectInput", "WinMM (compatibility)"} {
		a.comboAdd(idInputBackend, mode)
	}

	a.comboReset(idControlMode)
	for _, mode := range controlModeLabels {
		a.comboAdd(idControlMode, mode)
	}

	a.comboReset(idHotasBrakeBehavior)
	for _, mode := range brakeModeValues {
		a.comboAdd(idHotasBrakeBehavior, mode)
	}

	a.comboReset(idPedalMode)
	for _, mode := range []string{"Off", "Fine steering", "Move Camera", "Pitch", "Keyboard keys", "Boost/Drift"} {
		a.comboAdd(idPedalMode, mode)
	}

	for _, id := range []int{idTwistLeftAxisType, idTwistRightAxisType} {
		a.comboReset(id)
		for _, name := range []string{"Centered", "Lever"} {
			a.comboAdd(id, name)
		}
	}
	for _, id := range []int{idTwistLeftMode, idTwistRightMode} {
		a.comboReset(id)
		for _, mode := range []string{"Off", "Fine steering", "Move Camera", "Pitch", "Keyboard keys", "Boost/Drift"} {
			a.comboAdd(id, mode)
		}
	}
	a.comboReset(idWheelPedal3Mode)
	for _, mode := range []string{"Off", "Boost", "Drift"} {
		a.comboAdd(idWheelPedal3Mode, mode)
	}

	a.comboReset(idPollingRate)
	for _, hz := range pollingRateOptions {
		a.comboAdd(idPollingRate, fmt.Sprintf("%d", hz))
	}
	for _, id := range []int{idPedalLeftKey, idPedalRightKey, idTwistLeftLeftKey, idTwistLeftRightKey, idTwistRightLeftKey, idTwistRightRightKey} {
		a.comboReset(id)
		for _, k := range keyboardKeyOptions {
			a.comboAdd(id, k.Name)
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
	for i, mode := range controlModeValues {
		if normalizeControlMode(c.ControlMode) == mode {
			controlIdx = i
			break
		}
	}
	comboSetIndex(idControlMode, controlIdx)
	if normalizeInputBackend(c.InputBackend) == "WinMM" {
		comboSetDevice(idHotasThrottleDevice, c.HotasThrottleID)
		comboSetDevice(idHotasStickDevice, c.HotasStickID)
		comboSetDevice(idLeftDevice, c.LeftID)
		comboSetDevice(idRightDevice, c.RightID)
		if normalizeControlMode(c.ControlMode) == "WHEEL" {
			comboSetDevice(idPedalDevice, c.DrivingWheel.Handbrake.DeviceID)
		} else {
			comboSetDevice(idPedalDevice, c.PedalID)
		}
		comboSetDevice(idTwistLeftDevice, c.TwistLeft.DeviceID)
		comboSetDevice(idTwistRightDevice, c.TwistRight.DeviceID)
	} else {
		comboSetGUID(idHotasThrottleDevice, c.HotasThrottleDIGUID)
		comboSetGUID(idHotasStickDevice, c.HotasStickDIGUID)
		comboSetGUID(idLeftDevice, c.LeftDIGUID)
		comboSetGUID(idRightDevice, c.RightDIGUID)
		if normalizeControlMode(c.ControlMode) == "WHEEL" {
			comboSetGUID(idPedalDevice, c.DrivingWheel.Handbrake.DIGUID)
		} else {
			comboSetGUID(idPedalDevice, c.PedalDIGUID)
		}
		comboSetGUID(idTwistLeftDevice, c.TwistLeft.DIGUID)
		comboSetGUID(idTwistRightDevice, c.TwistRight.DIGUID)
	}
	comboSetIndex(idHotasThrottleAxis, c.HotasThrottleAxis)
	setChecked(a.controls[idHotasThrottleInvert], c.HotasThrottleInvert)
	brakeModeIdx := 1
	switch normalizeHotasBrakeBehavior(c.HotasBrakeBehavior) {
	case "Off":
		brakeModeIdx = 0
	case "Analog Brake Zone":
		brakeModeIdx = 2
	case "Centered Stick":
		brakeModeIdx = 3
	}
	comboSetIndex(idHotasBrakeBehavior, brakeModeIdx)
	hotatIdx := 0
	for i, mode := range []string{"Off", "Digital at Idle", "Analog Brake Zone"} {
		if mode == normalizeLeverBrakeBehavior(c.HotatBrakeBehavior) {
			hotatIdx = i
			break
		}
	}
	comboSetIndex(idHOTATBrakeBehavior, hotatIdx)
	setText(a.controls[idHOTATBrakeThreshold], strconv.Itoa(c.HotatBrakeThreshold))
	setText(a.controls[idHOTATBrakeZone], strconv.Itoa(c.HotatBrakeZone))
	setText(a.controls[idHotasBrakeThreshold], strconv.Itoa(c.HotasBrakeThreshold))
	setText(a.controls[idHotasBrakeZone], strconv.Itoa(c.HotasBrakeZone))
	comboSetIndex(idHotasStickXAxis, c.HotasStickXAxis)
	comboSetIndex(idHotasStickYAxis, c.HotasStickYAxis)
	setChecked(a.controls[idHotasStickInvertX], c.HotasStickInvertX)
	setChecked(a.controls[idHotasStickInvertY], c.HotasStickInvertY)
	comboSetIndex(idLeftAxis, c.LeftAxis)
	comboSetIndex(idRightAxis, c.RightAxis)
	if normalizeControlMode(c.ControlMode) == "WHEEL" {
		comboSetIndex(idPedalAxis, c.DrivingWheel.Handbrake.Axis)
	} else {
		comboSetIndex(idPedalAxis, c.PedalAxis)
	}
	a.applyAuxiliaryAxisToUI(c.TwistLeft, true)
	a.applyAuxiliaryAxisToUI(c.TwistRight, false)
	for _, row := range []struct {
		v                    wheelAxisConfig
		device, axis, invert int
	}{
		{c.DrivingWheel.Wheel, idWheelDevice, idWheelAxis, idWheelInvert},
		{c.DrivingWheel.Pedal1, idWheelPedal1Device, idWheelPedal1Axis, idWheelPedal1Invert},
		{c.DrivingWheel.Pedal2, idWheelPedal2Device, idWheelPedal2Axis, idWheelPedal2Invert},
		{c.DrivingWheel.Pedal3, idWheelPedal3Device, idWheelPedal3Axis, idWheelPedal3Invert},
	} {
		a.applyWheelAxis(row.v, row.device, row.axis, row.invert, normalizeInputBackend(c.InputBackend))
	}
	comboSetIndex(idWheelPedal3Mode, indexInOptions(a.comboOptions[idWheelPedal3Mode], c.DrivingWheel.Pedal3Mode))
	setChecked(a.controls[idWheelHandbrakeButton], c.DrivingWheel.HandbrakeButton)
	setChecked(a.controls[idLeftInvert], c.LeftInvert)
	setChecked(a.controls[idRightInvert], c.RightInvert)
	setChecked(a.controls[idLeftThrottleMode], c.LeftThrottleMode)
	setChecked(a.controls[idRightThrottleMode], c.RightThrottleMode)
	if normalizeControlMode(c.ControlMode) == "WHEEL" {
		setChecked(a.controls[idPedalInvert], c.DrivingWheel.Handbrake.Invert)
	} else {
		setChecked(a.controls[idPedalInvert], c.PedalInvert)
	}
	modeIdx := 0
	if strings.EqualFold(c.PedalMode, "Fine steering") {
		modeIdx = 1
	} else if strings.EqualFold(c.PedalMode, "Move Camera") {
		modeIdx = 2
	} else if strings.EqualFold(c.PedalMode, "Pitch") {
		modeIdx = 3
	} else if strings.EqualFold(c.PedalMode, "Keyboard keys") {
		modeIdx = 4
	} else if c.PedalMode == "Boost/Drift" {
		modeIdx = 5
	}
	a.populatePedalModes(normalizeControlMode(c.ControlMode))
	if normalizeControlMode(c.ControlMode) != "WHEEL" {
		comboSetIndex(idPedalMode, modeIdx)
	}
	comboSetIndex(idPedalLeftKey, keyOptionIndex(c.PedalLeftKey))
	comboSetIndex(idPedalRightKey, keyOptionIndex(c.PedalRightKey))
	setText(a.controls[idPedalStrength], strconv.Itoa(c.PedalSteerStrength))
	setText(a.controls[idWheelCenteringStrength], strconv.Itoa(ffbPercent(c.WheelCenteringStrength, defaultWheelCenteringStrength)))
	setText(a.controls[idWheelRumbleStrength], strconv.Itoa(ffbPercent(c.WheelRumbleStrength, defaultWheelRumbleStrength)))
	setText(a.controls[idWheelAudioStrength], strconv.Itoa(ffbPercent(c.WheelAudioStrength, defaultWheelAudioStrength)))
	setChecked(a.controls[idWheelTrueforce], c.WheelTrueforce)
	setText(a.controls[idCameraSensitivity], strconv.Itoa(c.CameraSensitivity))
	setText(a.controls[idCameraDeadzone], strconv.Itoa(c.CameraDeadzone))
	setText(a.controls[idPedalKeyThreshold], strconv.Itoa(c.PedalKeyThreshold))
	pidx := 0
	for i, hz := range pollingRateOptions {
		if hz == c.PollingHz {
			pidx = i
			break
		}
	}
	comboSetIndex(idPollingRate, pidx)
	setChecked(a.controls[idPreserve], c.PreserveThrust)
	setChecked(a.controls[idMonitorDuringGameplay], c.MonitorDuringGameplay)
	if a.uiReady {
		a.updateContextVisibility()
	}
}

func indexInOptions(list []string, value string) int {
	for i, v := range list {
		if v == value {
			return i
		}
	}
	return 0
}
func auxiliaryModeOptions(axisType string) []string {
	modes := []string{"Off", "Fine steering", "Move Camera", "Pitch", "Keyboard keys"}
	if normalizeAxisType(axisType) == "Lever" {
		return append(modes, "Boost", "Drift")
	}
	return append(modes, "Boost/Drift")
}
func (a *appState) setComboOptions(id int, opts []string) {
	a.comboReset(id)
	for _, o := range opts {
		a.comboAdd(id, o)
	}
}
func (a *appState) selectedComboText(id int) string {
	opts := a.comboOptions[id]
	if i := comboIndex(id); i >= 0 && i < len(opts) {
		return opts[i]
	}
	return "Off"
}
func (a *appState) populatePedalModes(mode string) {
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()
	opts := []string{"Off", "Fine steering", "Move Camera", "Pitch", "Keyboard keys", "Boost/Drift"}
	selected := c.PedalMode
	if mode == "WHEEL" {
		opts = []string{"Boost", "Drift"}
		selected = c.DrivingWheel.HandbrakeMode
	}
	a.setComboOptions(idPedalMode, opts)
	for i, m := range opts {
		if m == selected {
			comboSetIndex(idPedalMode, i)
			break
		}
	}
}
func auxiliaryIDs(left bool) []int {
	if left {
		return []int{idTwistLeftDevice, idTwistLeftAxis, idTwistLeftInvert, idTwistLeftAxisType, idTwistLeftDeadzone, idTwistLeftMode, idTwistLeftStrength, idTwistLeftCameraSensitivity, idTwistLeftCameraDeadzone, idTwistLeftLeftKey, idTwistLeftRightKey, idTwistLeftKeyThreshold, idTwistLeftOutputNegative}
	}
	return []int{idTwistRightDevice, idTwistRightAxis, idTwistRightInvert, idTwistRightAxisType, idTwistRightDeadzone, idTwistRightMode, idTwistRightStrength, idTwistRightCameraSensitivity, idTwistRightCameraDeadzone, idTwistRightLeftKey, idTwistRightRightKey, idTwistRightKeyThreshold, idTwistRightOutputNegative}
}

func (a *appState) applyAuxiliaryAxisToUI(v auxiliaryAxisConfig, left bool) {
	ids := auxiliaryIDs(left)
	comboSetIndex(ids[1], v.Axis)
	setChecked(a.controls[ids[2]], v.Invert)
	setChecked(a.controls[ids[12]], v.OutputNegative)
	if normalizeAxisType(v.AxisType) == "Lever" {
		comboSetIndex(ids[3], 1)
	} else {
		comboSetIndex(ids[3], 0)
	}
	modes := auxiliaryModeOptions(v.AxisType)
	a.setComboOptions(ids[5], modes)
	for i, mode := range modes {
		if strings.EqualFold(mode, v.Mode) {
			comboSetIndex(ids[5], i)
			break
		}
	}
	setText(a.controls[ids[6]], strconv.Itoa(v.SteerStrength))
	setText(a.controls[ids[7]], strconv.Itoa(v.CameraSensitivity))
	setText(a.controls[ids[8]], strconv.Itoa(v.CameraDeadzone))
	comboSetIndex(ids[9], keyOptionIndex(v.LeftKey))
	comboSetIndex(ids[10], keyOptionIndex(v.RightKey))
	setText(a.controls[ids[11]], strconv.Itoa(v.KeyThreshold))
}

func readLimitedUIInt(h uintptr, fallback, minValue, maxValue int) int {
	v, err := strconv.Atoi(strings.TrimSpace(getText(h)))
	if err != nil {
		return fallback
	}
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func (a *appState) readAuxiliaryAxisFromUI(v auxiliaryAxisConfig, left bool) auxiliaryAxisConfig {
	ids := auxiliaryIDs(left)
	v.DeviceID = comboDevice(ids[0])
	v.DIGUID = comboGUID(ids[0])
	v.Axis = comboIndex(ids[1])
	v.Invert = isChecked(a.controls[ids[2]])
	v.OutputNegative = isChecked(a.controls[ids[12]])
	v.AxisType = []string{"Centered", "Lever"}[min(comboIndex(ids[3]), 1)]
	// Axis deadzone is managed exclusively in Axis Tuning.
	modes := auxiliaryModeOptions(v.AxisType)
	v.Mode = modes[min(comboIndex(ids[5]), len(modes)-1)]
	v.SteerStrength = readLimitedUIInt(a.controls[ids[6]], v.SteerStrength, 0, 100)
	v.CameraSensitivity = readLimitedUIInt(a.controls[ids[7]], v.CameraSensitivity, 10, 250)
	v.CameraDeadzone = readLimitedUIInt(a.controls[ids[8]], v.CameraDeadzone, 0, 40)
	v.LeftKey = selectedKeyVK(ids[9])
	v.RightKey = selectedKeyVK(ids[10])
	v.KeyThreshold = readLimitedUIInt(a.controls[ids[11]], v.KeyThreshold, 5, 95)
	return v
}

func comboSetIndex(id, idx int) {
	opts := app.comboOptions[id]
	if len(opts) == 0 {
		app.comboSelected[id] = 0
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(opts) {
		idx = len(opts) - 1
	}
	app.comboSelected[id] = idx
	if h := app.controls[id]; h != 0 {
		procSendMessageW.Call(h, CB_SETCURSEL, uintptr(idx), 0)
	}
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
	comboSetIndex(id, idx)
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
	comboSetIndex(id, idx)
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
	idx, ok := app.comboSelected[id]
	if !ok || idx < 0 {
		return 0
	}
	return idx
}
func comboDevice(id int) int {
	idx := comboIndex(id)
	ids := app.deviceComboIDs[id]
	if idx >= 0 && idx < len(ids) {
		return ids[idx]
	}
	return -1
}

func (a *appState) readWheelAxis(v wheelAxisConfig, deviceID, axisID, invertID int) wheelAxisConfig {
	v.DeviceID = comboDevice(deviceID)
	v.DIGUID = comboGUID(deviceID)
	v.Axis = comboIndex(axisID)
	v.Invert = isChecked(a.controls[invertID])
	return v
}
func (a *appState) applyWheelAxis(v wheelAxisConfig, deviceID, axisID, invertID int, backend string) {
	if backend == "WinMM" {
		comboSetDevice(deviceID, v.DeviceID)
	} else {
		comboSetGUID(deviceID, v.DIGUID)
	}
	comboSetIndex(axisID, v.Axis)
	setChecked(a.controls[invertID], v.Invert)
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
	controlModes := controlModeValues
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
		if c.ControlMode != "WHEEL" {
			c.PedalID = comboDevice(idPedalDevice)
		}
	} else {
		c.HotasThrottleDIGUID = comboGUID(idHotasThrottleDevice)
		c.HotasStickDIGUID = comboGUID(idHotasStickDevice)
		c.LeftDIGUID = comboGUID(idLeftDevice)
		c.RightDIGUID = comboGUID(idRightDevice)
		if c.ControlMode != "WHEEL" {
			c.PedalDIGUID = comboGUID(idPedalDevice)
		}
		// Keep a matching WinMM ID when names line up so Auto has a safe legacy fallback.
		c.HotasThrottleID = comboDevice(idHotasThrottleDevice)
		c.HotasStickID = comboDevice(idHotasStickDevice)
		c.LeftID = comboDevice(idLeftDevice)
		c.RightID = comboDevice(idRightDevice)
		if c.ControlMode != "WHEEL" {
			c.PedalID = comboDevice(idPedalDevice)
		}
	}
	c.HotasThrottleAxis = comboIndex(idHotasThrottleAxis)
	c.HotasThrottleInvert = isChecked(a.controls[idHotasThrottleInvert])
	brakeBehaviors := brakeModeValues
	bbi := comboIndex(idHotasBrakeBehavior)
	if bbi < 0 || bbi >= len(brakeBehaviors) {
		bbi = 1
	}
	c.HotasBrakeBehavior = brakeBehaviors[bbi]
	hi := comboIndex(idHOTATBrakeBehavior)
	hotatModes := []string{"Off", "Digital at Idle", "Analog Brake Zone"}
	if hi < 0 || hi >= len(hotatModes) {
		hi = 0
	}
	c.HotatBrakeBehavior = hotatModes[hi]
	c.HotatBrakeThreshold = readLimitedUIInt(a.controls[idHOTATBrakeThreshold], c.HotatBrakeThreshold, 0, 50)
	c.HotatBrakeZone = readLimitedUIInt(a.controls[idHOTATBrakeZone], c.HotatBrakeZone, 1, 50)
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idHotasBrakeThreshold]))); e == nil {
		if n < 0 {
			n = 0
		}
		if n > 50 {
			n = 50
		}
		c.HotasBrakeThreshold = n
	}
	if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[idHotasBrakeZone]))); e == nil {
		if n < 1 {
			n = 1
		}
		if n > 50 {
			n = 50
		}
		c.HotasBrakeZone = n
	}
	c.HotasStickXAxis = comboIndex(idHotasStickXAxis)
	c.HotasStickYAxis = comboIndex(idHotasStickYAxis)
	c.HotasStickInvertX = isChecked(a.controls[idHotasStickInvertX])
	c.HotasStickInvertY = isChecked(a.controls[idHotasStickInvertY])
	c.LeftAxis = comboIndex(idLeftAxis)
	c.RightAxis = comboIndex(idRightAxis)
	if c.ControlMode != "WHEEL" {
		c.PedalAxis = comboIndex(idPedalAxis)
	}
	c.TwistLeft = a.readAuxiliaryAxisFromUI(c.TwistLeft, true)
	c.TwistRight = a.readAuxiliaryAxisFromUI(c.TwistRight, false)
	pi := comboIndex(idPollingRate)
	if pi >= 0 && pi < len(pollingRateOptions) {
		c.PollingHz = pollingRateOptions[pi]
	}
	c.LeftInvert = isChecked(a.controls[idLeftInvert])
	c.RightInvert = isChecked(a.controls[idRightInvert])
	c.LeftThrottleMode = isChecked(a.controls[idLeftThrottleMode])
	c.RightThrottleMode = isChecked(a.controls[idRightThrottleMode])
	if c.ControlMode != "WHEEL" {
		c.PedalInvert = isChecked(a.controls[idPedalInvert])
	}
	// Preserve previously saved legacy WASD calibration for downgrade safety.
	if c.ControlMode == "WHEEL" {
		if a.selectedComboText(idPedalMode) == "Boost" {
			c.DrivingWheel.HandbrakeMode = "Boost"
		} else {
			c.DrivingWheel.HandbrakeMode = "Drift"
		}
	} else {
		c.PedalMode = normalizePedalMode(a.selectedComboText(idPedalMode))
	}
	c.DrivingWheel.Wheel = a.readWheelAxis(c.DrivingWheel.Wheel, idWheelDevice, idWheelAxis, idWheelInvert)
	c.DrivingWheel.Pedal1 = a.readWheelAxis(c.DrivingWheel.Pedal1, idWheelPedal1Device, idWheelPedal1Axis, idWheelPedal1Invert)
	c.DrivingWheel.Pedal2 = a.readWheelAxis(c.DrivingWheel.Pedal2, idWheelPedal2Device, idWheelPedal2Axis, idWheelPedal2Invert)
	c.DrivingWheel.Pedal3 = a.readWheelAxis(c.DrivingWheel.Pedal3, idWheelPedal3Device, idWheelPedal3Axis, idWheelPedal3Invert)
	c.DrivingWheel.Pedal3Mode = a.selectedComboText(idWheelPedal3Mode)
	if c.ControlMode == "WHEEL" {
		c.DrivingWheel.Handbrake = a.readWheelAxis(c.DrivingWheel.Handbrake, idPedalDevice, idPedalAxis, idPedalInvert)
	}
	c.DrivingWheel.HandbrakeButton = isChecked(a.controls[idWheelHandbrakeButton])
	// The legacy WASD generator is retired. Preserve its other historical config
	// fields for backward compatibility but never activate keyboard walking.
	c.OnFootMode = "Off"
	c.ThumbWASDEnabled = false
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
	c.WheelTrueforce = isChecked(a.controls[idWheelTrueforce])
	// The UI shows the effective 0-100%; config keeps 0 for "default", so an explicit 0 is stored as off (-1).
	for _, f := range []struct {
		id  int
		dst *int
	}{{idWheelCenteringStrength, &c.WheelCenteringStrength}, {idWheelRumbleStrength, &c.WheelRumbleStrength}, {idWheelAudioStrength, &c.WheelAudioStrength}} {
		if n, e := strconv.Atoi(strings.TrimSpace(getText(a.controls[f.id]))); e == nil {
			if n <= 0 {
				n = -1
			}
			*f.dst = min(n, 100)
		}
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
	c.MonitorDuringGameplay = isChecked(a.controls[idMonitorDuringGameplay])
	a.cfg = c
	a.cfgMu.Unlock()
}

func (a *appState) onCommand(id, notify int) {
	// Native combo selection notifications are authoritative: synchronize
	// option index before any mode/device/axis handler reads comboIndex().
	if notify == CBN_SELCHANGE && a.isComboID(id) {
		if h := a.controls[id]; h != 0 {
			v, _, _ := procSendMessageW.Call(h, CB_GETCURSEL, 0, 0)
			idx := int(int32(v))
			if idx >= 0 && idx < len(a.comboOptions[id]) {
				a.comboSelected[id] = idx
			}
		}
	}
	if a.applyingProfile {
		return
	}
	switch {
	case id >= idProfileFirst && id < idProfileFirst+profileCount && notify == BN_CLICKED:
		a.switchProfile(id - idProfileFirst + 1)
		return
	case id >= idProfileIncludeFirst && id < idProfileIncludeFirst+profileCount && notify == BN_CLICKED:
		a.updateProfileIncluded(id-idProfileIncludeFirst, isChecked(a.controls[id]))
		return
	case id == idCycleLearn && notify == BN_CLICKED:
		a.startLearn("PROFILECYCLE", "global Profile Cycle button")
		return
	case id == idCycleClear && notify == BN_CLICKED:
		a.updateCycleBinding(nil)
		a.postStatus("Global Profile Cycle button cleared.")
		return
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
		setText(a.controls[idStart], "Initiating...")
		a.postStatus("Initiating virtual Xbox controller...")
		go a.startMappingAsync()
		return
	case id == idWalkingToggle && notify == BN_CLICKED:
		if !a.mappingActive.Load() || !a.xboxReady.Load() {
			a.walkingActive.Store(false)
			a.refreshWalkingUI()
			a.postStatus("Initiate Xbox Controller before enabling Walking Mode.")
			return
		}
		a.walkingActive.Store(!a.walkingActive.Load())
		a.refreshWalkingUI()
		return
	case id == idWalkingLearn && notify == BN_CLICKED:
		a.startLearn("WALKMODE", "Walking Mode toggle")
		return
	case id == idWalkingClear && notify == BN_CLICKED:
		a.clearBinding("WALKMODE", "Walking Mode toggle")
		return
	case id == idTestA && notify == BN_CLICKED:
		if !a.xboxReady.Load() {
			a.postStatus("Initiate Xbox Controller first; the virtual Xbox controller is intentionally disconnected while output is inactive.")
			return
		}
		a.testUntilMu.Lock()
		a.testUntil = time.Now().Add(350 * time.Millisecond)
		a.testUntilMu.Unlock()
		a.postStatus("Sent Xbox A test pulse.")
		return
	case id == idTestCamLeft && notify == BN_CLICKED:
		if !a.xboxReady.Load() {
			a.postStatus("Initiate Xbox Controller first; the virtual Xbox controller is intentionally disconnected while output is inactive.")
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
			a.postStatus("Initiate Xbox Controller first; the virtual Xbox controller is intentionally disconnected while output is inactive.")
			return
		}
		a.cameraTestMu.Lock()
		a.cameraTestValue = 1.0
		a.cameraTestUntil = time.Now().Add(800 * time.Millisecond)
		a.cameraTestMu.Unlock()
		a.postStatus("Camera test: virtual Xbox Right Stick X = full RIGHT for 0.8 seconds.")
		return
	case id == idSocialLinks && notify == BN_CLICKED:
		procShellExecuteW.Call(0, uintptr(unsafe.Pointer(utf16ptr("open"))), uintptr(unsafe.Pointer(utf16ptr("https://linktr.ee/jaffaquake"))), 0, 0, SW_SHOW)
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
		a.cfgMu.RLock()
		backend := normalizeInputBackend(a.cfg.InputBackend)
		a.cfgMu.RUnlock()
		if backend == "WinMM" {
			a.devicesMu.Lock()
			a.devices = enumerateDevices()
			a.devicesMu.Unlock()
		} else {
			_ = a.di.Restart(a.hwnd)
		}
		a.populateCombos()
		a.applyConfigToUI()
		a.updateContextVisibility()
		if backend == "WinMM" {
			a.postStatus("WinMM compatibility device list refreshed through the isolated helper.")
		} else {
			a.postStatus("DirectInput device list refreshed.")
		}
		return
	case id == idDetectLeft && notify == BN_CLICKED:
		a.startDetect("Left")
		return
	case id == idDetectRight && notify == BN_CLICKED:
		a.startDetect("Right")
		return
	case id == idWheelDetect && notify == BN_CLICKED:
		a.startDetect("Wheel Steering")
		return
	case id == idWheelPedal1Detect && notify == BN_CLICKED:
		a.startDetect("Wheel Pedal 1")
		return
	case id == idWheelPedal2Detect && notify == BN_CLICKED:
		a.startDetect("Wheel Pedal 2")
		return
	case id == idWheelPedal3Detect && notify == BN_CLICKED:
		a.startDetect("Wheel Pedal 3")
		return
	case id == idWheelHandbrakeDetect && notify == BN_CLICKED:
		a.startDetect("Wheel Hand Brake")
		return
	case id == idWheelHandbrakeLearn && notify == BN_CLICKED:
		a.startLearn("WHEELHANDBRAKE", "Hand Brake button")
		return
	case id >= idLearnClearBase && id < idLearnClearBase+len(buttonOutputs) && notify == BN_CLICKED:
		def := buttonOutputs[id-idLearnClearBase]
		a.clearBinding(def.Key, def.Label)
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
		if backends[idx] == "WinMM" {
			a.devicesMu.Lock()
			a.devices = enumerateDevices()
			a.devicesMu.Unlock()
		}
		a.populateCombos()
		a.applyConfigToUI()
		a.updateContextVisibility()
		saveConfig()
		a.postStatus("Input API changed to " + backends[idx] + ". DirectInput exposes additional rotation/slider axes; WinMM remains available as the compatibility fallback.")
		return
	}
	if notify == CBN_SELCHANGE || notify == EN_CHANGE || notify == BN_CLICKED {
		if (id == idTwistLeftAxisType || id == idTwistRightAxisType) && notify == CBN_SELCHANGE {
			modeID := idTwistLeftMode
			if id == idTwistRightAxisType {
				modeID = idTwistRightMode
			}
			previousMode := a.selectedComboText(modeID)
			axisType := "Centered"
			if comboIndex(id) == 1 {
				axisType = "Lever"
			}
			opts := auxiliaryModeOptions(axisType)
			a.setComboOptions(modeID, opts)
			comboSetIndex(modeID, indexInOptions(opts, previousMode)) // invalid old type-specific modes become Off
		}
		if id == idControlMode && notify == CBN_SELCHANGE {
			modes := controlModeValues
			idx := comboIndex(idControlMode)
			if idx >= len(modes) {
				idx = 0
			}
			a.cfgMu.Lock()
			a.cfg.ControlMode = modes[idx]
			a.cfgMu.Unlock()
			a.populatePedalModes(modes[idx])
			a.cfgMu.RLock()
			c := a.cfg
			a.cfgMu.RUnlock()
			if modes[idx] == "WHEEL" {
				if normalizeInputBackend(c.InputBackend) == "WinMM" {
					comboSetDevice(idPedalDevice, c.DrivingWheel.Handbrake.DeviceID)
				} else {
					comboSetGUID(idPedalDevice, c.DrivingWheel.Handbrake.DIGUID)
				}
				comboSetIndex(idPedalAxis, c.DrivingWheel.Handbrake.Axis)
				setChecked(a.controls[idPedalInvert], c.DrivingWheel.Handbrake.Invert)
			} else {
				if normalizeInputBackend(c.InputBackend) == "WinMM" {
					comboSetDevice(idPedalDevice, c.PedalID)
				} else {
					comboSetGUID(idPedalDevice, c.PedalDIGUID)
				}
				comboSetIndex(idPedalAxis, c.PedalAxis)
				setChecked(a.controls[idPedalInvert], c.PedalInvert)
			}
		}
		a.syncConfigFromUI()
		saveConfig() // Includes polling_hz; a changed rate survives software updates.
		if notify == CBN_SELCHANGE || (notify == BN_CLICKED && (id == idLeftThrottleMode || id == idRightThrottleMode)) {
			if id == idTwistLeftAxisType || id == idTwistRightAxisType {
				a.cfgMu.RLock()
				c := a.cfg
				a.cfgMu.RUnlock()
				if id == idTwistLeftAxisType {
					a.applyAuxiliaryAxisToUI(c.TwistLeft, true)
				} else {
					a.applyAuxiliaryAxisToUI(c.TwistRight, false)
				}
			}
			a.updateContextVisibility()
		}
	}
}

func (a *appState) mappingStatus() string {
	if a.xboxReady.Load() {
		s := a.xboxSlot.Load()
		if s >= 0 {
			return fmt.Sprintf("Mapping ACTIVE. Windows sees Xbox controller in XInput slot %d.\r\nIn Galactic Racer, set your controller scheme to Default.", s+1)
		}
		return "Mapping ACTIVE. Virtual Xbox controller is running.\r\nIn Galactic Racer, set your controller scheme to Default."
	}
	return "Mapping ACTIVE, but Xbox output is not ready yet."
}

func (a *appState) refreshWalkingUI() {
	active := a.mappingActive.Load() && a.xboxReady.Load() && a.walkingActive.Load()
	if h := a.monitorOutputHeading; h != 0 {
		if active {
			setText(h, "Movement Output")
		} else {
			setText(h, "Drive / Steer")
		}
	}
	if h := a.controls[idWalkingToggle]; h != 0 {
		setChecked(h, active)
		if active {
			setText(h, "Walking Mode: ON")
		} else {
			setText(h, "Walking Mode: OFF")
		}
	}
}

func (a *appState) refreshXboxUI() {
	defer a.refreshWalkingUI()
	if a.controls[idStart] == 0 {
		return
	}
	// Called on every virtual-controller start/stop completion message. Make
	// diagnostics track actual readiness, including failed initialization.
	defer a.updateContextVisibility()
	if a.mappingStarting.Load() {
		setText(a.controls[idStart], "Initiating...")
		return
	}
	if a.mappingActive.Load() && a.xboxReady.Load() {
		setText(a.controls[idStart], "Deactivate Xbox Controller")
		return
	}
	setText(a.controls[idStart], "Initiate Xbox Controller")
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

func (a *appState) moveStickDot(id, boxSize int, x, y float64) {
	const dot = 12
	h := a.controls[id]
	base, ok := a.baseRects[h]
	if h == 0 || !ok {
		return
	}
	x = clamp(x, -1, 1)
	y = clamp(y, -1, 1)
	rangePx := float64(boxSize-dot) / 2.0
	// Anchor movement to the dot's actual registered layout position.
	// The twist-axis insert shifts that position down by 210px, while the
	// original hardcoded 542px monitor origin was never updated in v1.98.
	// Using baseRects also keeps dots correctly aligned during scrolling.
	cx := float64(int(base.Left)-a.scrollX) + x*rangePx
	cy := float64(int(base.Top)-a.scrollY) - y*rangePx
	procSetWindowPos.Call(
		h,
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
	a.refreshWalkingUI()
	a.monitorMu.RLock()
	m := a.monitor
	a.monitorMu.RUnlock()
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()

	const boxSize = 126

	// Only the opt-in gameplay monitor refreshes visuals while mapping is active.
	// UI refresh remains at 10 Hz regardless of controller polling frequency.
	if !shouldRenderMonitor(a.mappingActive.Load(), c.MonitorDuringGameplay) {
		if !a.visualsPaused {
			setText(a.controls[idMonitor], "Live visual monitor PAUSED while game mapping is active.\r\nEnable Monitor during gameplay to show live controls (slight CPU impact).")
			a.setButtonTileStates(0)
			a.setOnFootButtonStates(0)
			a.moveStickDot(idLeftDot, boxSize, 0, 0)
			a.moveStickDot(idRightDot, boxSize, 0, 0)
			a.moveStickDot(idDriveSteerDot, boxSize, 0, 0)
			for _, barID := range []int{idLeftThrottleBar, idRightThrottleBar, idWheelThrottleBar, idWheelBrakeBar} {
				if h := a.controls[barID]; h != 0 {
					procSendMessageW.Call(h, 0x0400+2, 0, 0)
				}
			}
			setText(a.controls[idLeftVectorText], "Monitor paused")
			setText(a.controls[idRightVectorText], "Monitor paused")
			setText(a.controls[idDriveSteerVectorText], "Monitor paused")
			a.visualsPaused = true
		}
		return
	}
	a.visualsPaused = false

	mode := "Podracer Style (HOSAS): Average thrust"
	switch normalizeControlMode(c.ControlMode) {
	case "HOTAS":
		mode = "Traditional HOTAS"
	case "SINGLE":
		mode = "Single Stick"
	case "HOTAT":
		mode = "Podracer Style (HOTAT): " + map[bool]string{true: "Hybrid preserve thrust", false: "Average thrust"}[c.PreserveThrust]
	case "WHEEL":
		mode = "Driving Wheel"
	default:
		if c.PreserveThrust {
			mode = "Podracer Style (HOSAS): Hybrid preserve thrust"
		}
	}
	pedal := "Off"
	if c.PedalID >= 0 && normalizeControlMode(c.ControlMode) != "WHEEL" {
		pedal = fmt.Sprintf("%+.2f", m.Pedal)
	}
	camera := "Off"
	if strings.EqualFold(c.PedalMode, "Move Camera") {
		camera = fmt.Sprintf("RX %+.2f", m.CameraX)
	} else if strings.EqualFold(c.PedalMode, "Pitch") {
		camera = fmt.Sprintf("Pitch LY %+.2f", m.Pedal)
	} else if strings.EqualFold(c.PedalMode, "Keyboard keys") {
		camera = fmt.Sprintf("Keys %s / %s @ %d%%", keyName(c.PedalLeftKey), keyName(c.PedalRightKey), c.PedalKeyThreshold)
	}
	var s string
	if normalizeControlMode(c.ControlMode) == "WHEEL" {
		s = fmt.Sprintf("Wheel steer %+.2f  Throttle %.2f  Brake %.2f\r\nAdditional Input 1 %+.2f  Input 2 %+.2f  Pedal 3: %s / Hand Brake: %s\r\n%s",
			m.RightX, m.RawWheelThrottle, m.RawWheelBrake, m.TwistLeft, m.TwistRight, c.DrivingWheel.Pedal3Mode, c.DrivingWheel.HandbrakeMode, mode)
	} else if strings.EqualFold(normalizeControlMode(c.ControlMode), "HOTAS") {
		brakeText := fmt.Sprintf("%.0f%%", m.HotasBrakeAmount*100.0)
		s = fmt.Sprintf(
			"Throttle %.2f  Brake %s  Stick X %+.2f  Stick Y %+.2f\r\nTwist L %+.2f  R %+.2f  Pedal %s  Output %s\r\n%s",
			(m.LeftY+1)/2, brakeText, m.RightX, m.RightY, m.TwistLeft, m.TwistRight, pedal, camera, mode,
		)
	} else {
		s = fmt.Sprintf(
			"Left %+.2f  Right %+.2f  Drive %+.2f  Steer %+.2f\r\nTwist L %+.2f  R %+.2f  Pedal %s  Output %s\r\n%s",
			m.LeftY, m.RightY, m.Drive, m.Steer, m.TwistLeft, m.TwistRight, pedal, camera, mode,
		)
	}
	if a.walkingActive.Load() && a.mappingActive.Load() {
		setText(a.controls[idMonitor], "Walking Mode — left stick moves, right stick looks.\r\nThrottle travel 45-55% = STAND STILL; racing brake zones ignored.\r\n"+mode)
	} else {
		setText(a.controls[idMonitor], s)
	}

	a.moveStickDot(idLeftDot, boxSize, m.LeftX, m.LeftY)
	// Raw physical input only. Vertical fill always rises from 0% to 100%.
	setBar := func(id int, v float64) {
		if h := a.controls[id]; h != 0 {
			procSendMessageW.Call(h, 0x0400+2, uintptr(int(clamp(v, 0, 1)*100+0.5)), 0)
		}
	} // PBM_SETPOS
	setBar(idLeftThrottleBar, (m.LeftY+1)/2)
	setBar(idRightThrottleBar, (m.RightY+1)/2)
	setBar(idWheelThrottleBar, m.RawWheelThrottle)
	setBar(idWheelBrakeBar, m.RawWheelBrake)
	a.moveStickDot(idRightDot, boxSize, m.RightX, m.RightY)
	if a.walkingActive.Load() && a.mappingActive.Load() {
		a.moveStickDot(idDriveSteerDot, boxSize, m.WalkLX, m.WalkLY)
	} else {
		a.moveStickDot(idDriveSteerDot, boxSize, m.Steer, m.Drive)
	}
	selected := normalizeControlMode(c.ControlMode)
	switch selected {
	case "WHEEL":
		setText(a.controls[idLeftVectorText], fmt.Sprintf("Throttle %.0f%%  Brake %.0f%%", m.RawWheelThrottle*100, m.RawWheelBrake*100))
		setText(a.controls[idRightVectorText], fmt.Sprintf("Wheel X %+.2f", m.RightX))
	case "HOTAS":
		if normalizeHotasBrakeBehavior(c.HotasBrakeBehavior) == "Centered Stick" {
			setText(a.controls[idLeftVectorText], fmt.Sprintf("Throttle Y %+.2f", m.LeftY))
		} else {
			setText(a.controls[idLeftVectorText], fmt.Sprintf("Throttle %.0f%%", (m.LeftY+1)*50))
		}
		setText(a.controls[idRightVectorText], fmt.Sprintf("Stick X %+.2f  Y %+.2f", m.RightX, m.RightY))
	case "SINGLE":
		setText(a.controls[idLeftVectorText], fmt.Sprintf("X %+.2f  Y %+.2f", m.LeftX, m.LeftY))
		setText(a.controls[idRightVectorText], "Not used")
	default:
		if selected == "HOTAT" || c.LeftThrottleMode {
			setText(a.controls[idLeftVectorText], fmt.Sprintf("Throttle %.0f%%", (m.LeftY+1)*50))
		} else {
			setText(a.controls[idLeftVectorText], fmt.Sprintf("Side %+.2f  F %+.2f", m.LeftX, m.LeftY))
		}
		if selected == "HOTAT" || c.RightThrottleMode {
			setText(a.controls[idRightVectorText], fmt.Sprintf("Throttle %.0f%%", (m.RightY+1)*50))
		} else {
			setText(a.controls[idRightVectorText], fmt.Sprintf("Side %+.2f  F %+.2f", m.RightX, m.RightY))
		}
	}

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
	if a.walkingActive.Load() && a.mappingActive.Load() {
		setText(a.controls[idDriveSteerVectorText], fmt.Sprintf("Move LX %+.2f  LY %+.2f\r\nLook RX %+.2f  RY %+.2f", m.WalkLX, m.WalkLY, m.WalkRX, m.WalkRY))
	} else {
		setText(a.controls[idDriveSteerVectorText], fmt.Sprintf("Drive %+0.2f  %s\r\nSteer %+0.2f  %s", m.Drive, driveDir, m.Steer, steerDir))
	}
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
				fmt.Sprintf("Detecting %s through DirectInput: MOVE/PRESS THIS INPUT to its maximum now...", side), "", -1,
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
	a.postStatus(fmt.Sprintf("Detecting %s through WinMM: MOVE/PRESS THIS INPUT to its maximum now...", side))
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
		} else if strings.HasPrefix(r.Side, "Wheel") {
			var w *wheelAxisConfig
			switch r.Side {
			case "Wheel Steering":
				w = &a.cfg.DrivingWheel.Wheel
			case "Wheel Pedal 1":
				w = &a.cfg.DrivingWheel.Pedal1
			case "Wheel Pedal 2":
				w = &a.cfg.DrivingWheel.Pedal2
			case "Wheel Pedal 3":
				w = &a.cfg.DrivingWheel.Pedal3
			case "Wheel Hand Brake":
				w = &a.cfg.DrivingWheel.Handbrake
			}
			if w != nil {
				w.DeviceID = r.DeviceID
				if r.UseDirectInput {
					w.DIGUID = r.DISideGUID
				}
				w.Axis = r.Axis
				w.Invert = r.Invert
			}
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
		a.updateContextVisibility() // Newly detected hardware reveals its axis and mode options.
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
	a.cfgMu.RLock()
	backend := normalizeInputBackend(a.cfg.InputBackend)
	a.cfgMu.RUnlock()
	a.postStatus(fmt.Sprintf("Learning %s using %s: press the controller button, trigger, hat, or switch position you want...", label, backend))
	go func() {
		var b binding
		var ok bool
		switch backend {
		case "WinMM":
			b, ok = a.learnWinMM(5 * time.Second)
		case "DirectInput":
			b, ok = a.learnDirectInput(5 * time.Second)
		default:
			b, ok = a.learnAuto(5 * time.Second)
		}
		if ok {
			a.queueLearn(learnResult{Key: key, Binding: b, OK: true})
			return
		}
		a.queueLearn(learnResult{Key: key, Message: "Nothing detected. Try Learn again and press/move the control after clicking Learn. If the device is visible only through one API, select DirectInput or WinMM explicitly."})
	}()
}

func (a *appState) learnWinMM(timeout time.Duration) (binding, bool) {
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
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, d := range devs {
			st, ok := readJoy(d.ID)
			if !ok {
				continue
			}
			bl := base[d.ID]
			pressed := st.Buttons &^ bl.buttons
			if pressed != 0 {
				for i := 0; i < 32; i++ {
					if pressed&(1<<uint(i)) != 0 {
						return binding{Kind: "button", Backend: "WinMM", DeviceID: d.ID, DeviceName: d.Name, Button: i}, true
					}
				}
			}
			if st.POV != JOY_POVCENTERED && st.POV != bl.pov {
				return binding{Kind: "pov", Backend: "WinMM", DeviceID: d.ID, DeviceName: d.Name, POVIndex: 0, POV: int(st.POV)}, true
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	return binding{}, false
}

func detectDirectInputLearn(base map[string]diJoyState, snaps []diFullSnapshot) (binding, bool) {
	for _, snap := range snaps {
		bl, exists := base[snap.GUID]
		if !exists {
			continue
		}
		for i := 0; i < len(snap.State.Buttons); i++ {
			now := snap.State.Buttons[i]&0x80 != 0
			was := bl.Buttons[i]&0x80 != 0
			if now && !was {
				return binding{Kind: "button", Backend: "DirectInput", DeviceID: -1, DIGUID: snap.GUID, DeviceName: snap.Name, Button: i}, true
			}
		}
		for i := 0; i < len(snap.State.POV); i++ {
			now := snap.State.POV[i]
			was := bl.POV[i]
			if now != 0xFFFFFFFF && now != was {
				return binding{Kind: "pov", Backend: "DirectInput", DeviceID: -1, DIGUID: snap.GUID, DeviceName: snap.Name, POVIndex: i, POV: int(now)}, true
			}
		}
	}
	return binding{}, false
}

func (a *appState) learnDirectInput(timeout time.Duration) (binding, bool) {
	base := make(map[string]diJoyState)
	for _, snap := range a.di.FullSnapshots() {
		base[snap.GUID] = snap.State
	}
	if len(base) == 0 {
		return binding{}, false
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if b, ok := detectDirectInputLearn(base, a.di.FullSnapshots()); ok {
			return b, true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return binding{}, false
}

func (a *appState) learnAuto(timeout time.Duration) (binding, bool) {
	// Auto watches both APIs at the same time. DirectInput is checked first on every
	// cycle so a device visible through both APIs is saved as a DirectInput binding.
	diBase := make(map[string]diJoyState)
	for _, snap := range a.di.FullSnapshots() {
		diBase[snap.GUID] = snap.State
	}
	devs := enumerateDevices()
	type winBase struct {
		buttons uint32
		pov     uint32
	}
	wmBase := map[int]winBase{}
	for _, d := range devs {
		if st, ok := readJoy(d.ID); ok {
			wmBase[d.ID] = winBase{st.Buttons, st.POV}
		}
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(diBase) > 0 {
			if b, ok := detectDirectInputLearn(diBase, a.di.FullSnapshots()); ok {
				return b, true
			}
		}
		for _, d := range devs {
			st, ok := readJoy(d.ID)
			if !ok {
				continue
			}
			bl := wmBase[d.ID]
			pressed := st.Buttons &^ bl.buttons
			if pressed != 0 {
				for i := 0; i < 32; i++ {
					if pressed&(1<<uint(i)) != 0 {
						return binding{Kind: "button", Backend: "WinMM", DeviceID: d.ID, DeviceName: d.Name, Button: i}, true
					}
				}
			}
			if st.POV != JOY_POVCENTERED && st.POV != bl.pov {
				return binding{Kind: "pov", Backend: "WinMM", DeviceID: d.ID, DeviceName: d.Name, POVIndex: 0, POV: int(st.POV)}, true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return binding{}, false
}

func (a *appState) queueLearn(r learnResult) {
	a.pendingMu.Lock()
	a.learnQueue = append(a.learnQueue, r)
	a.pendingMu.Unlock()
	procPostMessageW.Call(a.hwnd, WM_APP_LEARN, 0, 0)
}
func (a *appState) clearBinding(key, label string) {
	a.cfgMu.Lock()
	if a.cfg.Bindings != nil {
		delete(a.cfg.Bindings, key)
	}
	a.cfgMu.Unlock()
	saveConfig()
	a.updateBindingLabels()
	a.postStatus(label + " cleared.")
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
		if r.Key == "PROFILECYCLE" {
			a.updateCycleBinding(&r.Binding)
			a.postStatus("Global Profile Cycle button assigned to " + bindingDisplay(r.Binding))
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
	if k == "WALKMODE" {
		return "Walking Mode toggle"
	}
	if k == "PROFILECYCLE" {
		return "Global Profile Cycle button"
	}
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
		a.tooltipForID(idLearnBase+100+i, "Assigned to "+bo.Label+": "+s)
	}
	if h := a.controls[idWalkingLabel]; h != 0 {
		s := "Not assigned"
		if b, ok := c.Bindings["WALKMODE"]; ok {
			s = bindingDisplay(b)
		}
		setText(h, s)
	}
	a.refreshCycleBindingLabel()
}
func bindingDisplay(b binding) string {
	backend := normalizeBindingBackend(b.Backend)
	if backend == "DirectInput" || b.DIGUID != "" {
		name := strings.TrimSpace(b.DeviceName)
		if name == "" {
			name = "DirectInput device"
		}
		if strings.EqualFold(b.Kind, "button") {
			return fmt.Sprintf("%s / Button %d [DI]", name, b.Button+1)
		}
		if strings.EqualFold(b.Kind, "pov") {
			return fmt.Sprintf("%s / POV %d %s [DI]", name, b.POVIndex+1, povName(b.POV))
		}
	}
	if strings.EqualFold(b.Kind, "button") {
		return fmt.Sprintf("Joystick %d / Button %d [WinMM]", b.DeviceID, b.Button+1)
	}
	if strings.EqualFold(b.Kind, "pov") {
		return fmt.Sprintf("Joystick %d / POV %s [WinMM]", b.DeviceID, povName(b.POV))
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
