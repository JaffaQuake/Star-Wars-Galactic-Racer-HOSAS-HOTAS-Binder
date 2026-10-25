//go:build windows

package main

import (
	"fmt"
	"math"
	"syscall"
	"time"
	"unsafe"
)

const (
	tuningComboID           = 2100
	tuningDeadzoneID        = 2101
	tuningCurveID           = 2102
	tuningFullID            = 2103
	tuningTestID            = 2104
	tuningResetID           = 2105
	tuningApplyID           = 2106
	tuningCloseID           = 2107
	tuningCurveLabelID      = 2116
	tuningFullLabelID       = 2117
	tuningGraphLabelID      = 2118
	tuningTestLabelID       = 2119
	tuningDeadzoneValueID   = 2111
	tuningCurveValueID      = 2112
	tuningFullValueID       = 2113
	tuningTestValueID       = 2114
	tuningLiveCheckID       = 2120
	tuningEmptyLabelID      = 2121
	tuningDeadzoneLabelID   = 2122
	tuningDescriptionID     = 2115
	tuningWeightID          = 2130
	tuningWeightValueID     = 2131
	tuningWeightLabelID     = 2132
	tuningTwoStageID        = 2133
	tuningMidpointLabelID   = 2134
	tuningMidpointSliderID  = 2135
	tuningMidpointPercentID = 2151
	tuningCustomMidID       = 2136
	tuningMidXLabelID       = 2139
	tuningMidXID            = 2140
	tuningMidXValueID       = 2141
	tuningMidYLabelID       = 2142
	tuningMidYID            = 2143
	tuningMidYValueID       = 2144
	tuningStage2LabelID     = 2145
	tuningStage2ID          = 2146
	tuningStage2ValueID     = 2147
	tuningStage2WLabelID    = 2148
	tuningStage2WID         = 2149
	tuningStage2WValueID    = 2150
	TBM_GETPOS              = 0x0400
	TBM_SETPOS              = 0x0405
	TBM_SETRANGE            = 0x0406
	WM_PAINT                = 0x000F
)

type tuningPaintStruct struct {
	Hdc       uintptr
	Erase     int32
	Paint     rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

type axisTuningPopup struct {
	hwnd        uintptr
	selected    int
	isHOTAS     bool
	controlMode string
	entries     []axisTuneEntry
	layoutShift int
	draft       map[string]axisTuneProfile
	controls    map[int]uintptr
	embedded    bool
	liveSign    float64 // centered signed travel; graph plots magnitude
	graphTop    int
	graphBottom int
	suppressUI  bool // ignore change notifications while populating controls
}

var (
	axisTuningCallback   = syscall.NewCallback(axisTuningWndProc)
	axisTuningClassReady bool
	procBeginPaint       = user32.NewProc("BeginPaint")
	procEndPaint         = user32.NewProc("EndPaint")
	procInvalidateRect   = user32.NewProc("InvalidateRect")
	procEnableWindow     = user32.NewProc("EnableWindow")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procMoveToEx         = gdi32.NewProc("MoveToEx")
	procLineTo           = gdi32.NewProc("LineTo")
	procEllipse          = gdi32.NewProc("Ellipse")
	procFillRect         = user32.NewProc("FillRect")
)

// createEmbeddedAxisTuning builds the former popup as an in-window tab page.
// It keeps the same curve math and axis selection widgets, without a second window.
func (a *appState) createEmbeddedAxisTuning(x, y, width, height int) {
	a.cfgMu.RLock()
	c := a.cfg
	a.cfgMu.RUnlock()
	c.Tuning = cloneAxisProfiles(c.Tuning)
	initAxisProfiles(&c)
	p := &axisTuningPopup{draft: make(map[string]axisTuneProfile), controls: make(map[int]uintptr), embedded: true}
	p.controlMode = normalizeControlMode(c.ControlMode)
	p.isHOTAS = p.controlMode == "HOTAS"
	p.entries = nil // filtered from assigned/active hardware after the UI is ready
	for k, v := range c.Tuning {
		p.draft[k] = v
	}
	hInst, _, _ := procGetModuleHandleW.Call(0)
	className := utf16ptr("GalacticRacerAxisTuningPopup")
	if !axisTuningClassReady {
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
		wc := wndClassEx{CbSize: uint32(unsafe.Sizeof(wndClassEx{})), LpfnWndProc: axisTuningCallback, HInstance: hInst, HCursor: cursor, HbrBackground: uintptr(COLOR_BTNFACE + 1), LpszClassName: className}
		if ok, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); ok == 0 {
			a.postStatus("Unable to create embedded Axis Tuning page.")
			return
		}
		axisTuningClassReady = true
	}
	a.tuning = p
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16ptr(""))),
		WS_CHILD|WS_VISIBLE, uintptr(x), uintptr(y), uintptr(width), uintptr(height), a.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		a.tuning = nil
		a.postStatus("Unable to create Axis Tuning tab.")
		return
	}
	p.hwnd = hwnd
	a.baseRects[hwnd] = rect{Left: int32(x), Top: int32(y), Right: int32(x + width), Bottom: int32(y + height)}
	p.createControls()
	p.setSlider(tuningTestID, 50)
	p.refreshEntriesFromConfig()
}

func (p *axisTuningPopup) control(class, label string, style uintptr, x, y, w, h, id int) uintptr {
	hnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(utf16ptr(class))), uintptr(unsafe.Pointer(utf16ptr(label))), style, uintptr(x), uintptr(y), uintptr(w), uintptr(h), p.hwnd, uintptr(id), 0, 0)
	if hnd != 0 && app.font != 0 {
		procSendMessageW.Call(hnd, WM_SETFONT, app.font, 1)
	}
	if id != 0 {
		p.controls[id] = hnd
	}
	return hnd
}
func (p *axisTuningPopup) label(label string, x, y, w, h, id int) {
	p.control("STATIC", label, WS_CHILD|WS_VISIBLE|SS_LEFT, x, y, w, h, id)
}
func (p *axisTuningPopup) button(label string, x, y, w, h, id int) {
	p.control("BUTTON", label, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, x, y, w, h, id)
}
func (p *axisTuningPopup) slider(x, y, w, h, id, lo, hi int) {
	ctrl := p.control("msctls_trackbar32", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP, x, y, w, h, id)
	packed := uint32(uint16(lo)) | uint32(uint16(hi))<<16
	procSendMessageW.Call(ctrl, TBM_SETRANGE, 1, uintptr(packed))
}
func (p *axisTuningPopup) sliderValue(id int) int {
	v, _, _ := procSendMessageW.Call(p.controls[id], TBM_GETPOS, 0, 0)
	return int(int32(v))
}
func (p *axisTuningPopup) setSlider(id, val int) {
	procSendMessageW.Call(p.controls[id], TBM_SETPOS, 1, uintptr(val))
}

// Embedded controls now begin flush with the shared Calibration section header,
// rather than inheriting the old detached dialog's 24px left gutter.
func (p *axisTuningPopup) createControls() {
	p.label("INPUT TO CONFIGURE", 0, 19, 300, 22, 0)
	p.label("No assigned, active analog inputs to configure.", 0, 96, 540, 26, tuningEmptyLabelID)
	combo := p.control("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 0, 45, 594, 270, tuningComboID)
	for _, entry := range p.entries {
		procSendMessageW.Call(combo, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(entry.Name))))
	}
	p.label("Deadzone", 0, 101, 145, 25, tuningDeadzoneLabelID)
	p.slider(168, 96, 300, 38, tuningDeadzoneID, 0, 40)
	p.label("6%", 482, 102, 127, 25, tuningDeadzoneValueID)
	p.label("Response curve", 0, 158, 160, 25, tuningCurveLabelID)
	p.slider(168, 153, 300, 38, tuningCurveID, 20, 300)
	p.label("Linear", 482, 159, 127, 25, tuningCurveValueID)
	p.control("BUTTON", "Two Stage Curve", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 0, 117, 190, 26, tuningTwoStageID)
	p.label("Weighting", 0, 191, 145, 25, tuningWeightLabelID)
	p.slider(168, 187, 300, 32, tuningWeightID, 0, 100)
	p.label("Center", 482, 192, 114, 25, tuningWeightValueID)
	// Stage 2 follows Stage 1; Midpoint controls appear after both stages.
	p.label("Curve 2", 0, 225, 150, 25, tuningStage2LabelID)
	p.slider(168, 220, 300, 34, tuningStage2ID, 20, 300)
	p.label("Linear", 482, 225, 113, 25, tuningStage2ValueID)
	p.label("Weighting", 0, 262, 150, 25, tuningStage2WLabelID)
	p.slider(168, 257, 300, 34, tuningStage2WID, 0, 100)
	p.label("Center", 482, 262, 113, 25, tuningStage2WValueID)
	p.control("BUTTON", "Custom Midpoint", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 0, 331, 200, 27, tuningCustomMidID)
	// The normal diagonal midpoint slider is directly beneath the checkbox.
	// Custom mode replaces that same row with X/Y sliders, without preset UI.
	p.label("Midpoint Position", 0, 371, 165, 25, tuningMidpointLabelID)
	p.slider(168, 366, 300, 34, tuningMidpointSliderID, 10, 90)
	p.label("50%", 482, 371, 110, 25, tuningMidpointPercentID)
	p.label("X — Input", 0, 371, 150, 25, tuningMidXLabelID)
	p.slider(168, 366, 300, 34, tuningMidXID, 10, 90)
	p.label("50%", 482, 371, 113, 25, tuningMidXValueID)
	p.label("Y — Output", 0, 407, 150, 25, tuningMidYLabelID)
	p.slider(168, 402, 300, 34, tuningMidYID, 10, 90)
	p.label("50%", 482, 407, 113, 25, tuningMidYValueID)
	p.label("Full Output Point", 0, 215, 160, 25, tuningFullLabelID)
	p.slider(168, 210, 300, 38, tuningFullID, 50, 100)
	p.label("100%", 482, 216, 127, 25, tuningFullValueID)
	p.label("Response preview — travel (%) versus output (%)", 0, 264, 540, 23, tuningGraphLabelID)
	p.label("", 0, 516, 592, 37, tuningDescriptionID)
	p.label("Test travel", 0, 558, 115, 25, tuningTestLabelID)
	p.slider(116, 551, 355, 37, tuningTestID, 0, 100)
	p.label("0% -> 0%", 481, 559, 127, 25, tuningTestValueID)
	p.control("BUTTON", "Live Input", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 0, 598, 150, 28, tuningLiveCheckID)
	p.button("Reset This Axis", 0, 640, 150, 30, tuningResetID)

}

func (p *axisTuningPopup) entry() (string, bool, bool) {
	i := p.selected
	if i < 0 || i >= len(p.entries) {
		i = 0
	}
	if len(p.entries) == 0 {
		return "", true, false
	}
	e := p.entries[i]
	return e.Key, e.DeadzoneOnly, e.OneWay
}

// Display only meaningful tuning controls. Digital-output axes need deadzone
// only; the response graph and preview slider would imply analog behavior.
// Compact reflow keeps the graph, both stage sliders and custom midpoint
// INSIDE the tab. Controls not applicable to the active axis are hidden.
func (p *axisTuningPopup) layoutProfile(deadzoneOnly, digitalOnly bool) {
	key, _, _ := p.entry()
	profile := checkedAxisTune(p.draft[key])
	stage := !deadzoneOnly && profile.TwoStage
	custom := stage && profile.CustomMidpoint
	move := func(id, x, y, width, height int) {
		if h := p.controls[id]; h != 0 {
			procSetWindowPos.Call(h, 0, uintptr(x), uintptr(y), uintptr(width), uintptr(height), SWP_NOZORDER|SWP_NOACTIVATE)
		}
	}
	show := func(id int, visible bool) {
		if h := p.controls[id]; h != 0 {
			if visible {
				procShowWindow.Call(h, SW_SHOW)
			} else {
				procShowWindow.Call(h, SW_HIDE)
			}
		}
	}
	for _, id := range []int{tuningCurveLabelID, tuningCurveID, tuningCurveValueID,
		tuningWeightLabelID, tuningWeightID, tuningWeightValueID,
		tuningFullLabelID, tuningFullID, tuningFullValueID, tuningTwoStageID} {
		show(id, !deadzoneOnly)
	}
	for _, id := range []int{tuningCustomMidID,
		tuningStage2LabelID, tuningStage2ID, tuningStage2ValueID,
		tuningStage2WLabelID, tuningStage2WID, tuningStage2WValueID} {
		show(id, stage)
	}
	// The diagonal midpoint slider is replaced entirely by custom X/Y controls.
	for _, id := range []int{tuningMidpointLabelID, tuningMidpointSliderID, tuningMidpointPercentID} {
		show(id, stage && !custom)
	}
	for _, id := range []int{tuningMidXLabelID,
		tuningMidXID, tuningMidXValueID, tuningMidYLabelID, tuningMidYID, tuningMidYValueID} {
		show(id, custom)
	}
	for _, id := range []int{tuningGraphLabelID, tuningDescriptionID, tuningTestLabelID,
		tuningTestID, tuningTestValueID, tuningLiveCheckID} {
		show(id, !digitalOnly)
	}
	if digitalOnly {
		move(tuningResetID, 0, 122, 150, 30)
		p.graphTop, p.graphBottom = 0, 0
		procInvalidateRect.Call(p.hwnd, 0, 1)
		return
	}
	move(tuningDeadzoneLabelID, 0, 72, 150, 25)
	move(tuningDeadzoneID, 168, 67, 300, 34)
	move(tuningDeadzoneValueID, 482, 73, 110, 25)
	// Full Output Point is global for every analog curve and sits directly
	// underneath Deadzone. It no longer follows the Two Stage midpoint block.
	move(tuningFullLabelID, 0, 111, 160, 25)
	move(tuningFullID, 168, 106, 300, 34)
	move(tuningFullValueID, 482, 111, 110, 25)
	move(tuningTwoStageID, 0, 150, 205, 26)
	move(tuningCurveLabelID, 0, 190, 160, 25)
	move(tuningCurveID, 168, 185, 300, 34)
	move(tuningCurveValueID, 482, 190, 110, 25)
	move(tuningWeightLabelID, 0, 229, 160, 25)
	move(tuningWeightID, 168, 225, 300, 34)
	move(tuningWeightValueID, 482, 229, 110, 25)
	var graphLabelY, graphHeight int
	if deadzoneOnly {
		// HOSAS physical-stick tuning exposes deadzone only, as before.
		graphLabelY, graphHeight = 125, 250
	} else if stage {
		// Curve 1 -> Curve 2 -> Custom Midpoint checkbox. There is no
		// redundant standalone Midpoint heading and no preset selector.
		move(tuningStage2LabelID, 0, 268, 160, 25)
		move(tuningStage2ID, 168, 263, 300, 34)
		move(tuningStage2ValueID, 482, 268, 110, 25)
		move(tuningStage2WLabelID, 0, 307, 160, 25)
		move(tuningStage2WID, 168, 303, 300, 34)
		move(tuningStage2WValueID, 482, 307, 110, 25)
		move(tuningCustomMidID, 0, 349, 200, 27)
		if custom {
			move(tuningMidXLabelID, 0, 388, 160, 25)
			move(tuningMidXID, 168, 383, 300, 34)
			move(tuningMidXValueID, 482, 388, 110, 25)
			move(tuningMidYLabelID, 0, 424, 160, 25)
			move(tuningMidYID, 168, 419, 300, 34)
			move(tuningMidYValueID, 482, 424, 110, 25)
			graphLabelY, graphHeight = 462, 134
		} else {
			move(tuningMidpointLabelID, 0, 388, 160, 25)
			move(tuningMidpointSliderID, 168, 383, 300, 34)
			move(tuningMidpointPercentID, 482, 388, 110, 25)
			graphLabelY, graphHeight = 426, 170
		}
	} else {
		graphLabelY, graphHeight = 270, 230
	}
	move(tuningGraphLabelID, 0, graphLabelY, 550, 25)
	p.graphTop = graphLabelY + 24
	p.graphBottom = p.graphTop + graphHeight
	move(tuningDescriptionID, 0, p.graphBottom+6, 592, 37)
	move(tuningTestLabelID, 0, p.graphBottom+49, 116, 25)
	move(tuningTestID, 116, p.graphBottom+45, 355, 34)
	move(tuningTestValueID, 481, p.graphBottom+49, 110, 25)
	move(tuningLiveCheckID, 0, p.graphBottom+84, 150, 27)
	move(tuningResetID, 0, p.graphBottom+121, 150, 30)
	p.setManualTravelEnabled()
	procInvalidateRect.Call(p.hwnd, 0, 1)
}

// Refresh only when the configured/active input list actually changes, so a
// timer or unrelated UI update cannot reset a slider mid-edit.
func (p *axisTuningPopup) refreshEntriesFromConfig() {
	if p == nil || p.hwnd == 0 {
		return
	}
	app.cfgMu.RLock()
	c := app.cfg
	app.cfgMu.RUnlock()
	mode := normalizeControlMode(c.ControlMode)
	all := axisTuneEntriesForControlMode(mode)
	backend := normalizeInputBackend(c.InputBackend)
	assigned := func(id int, guid string) bool {
		if backend == "WinMM" {
			return id >= 0
		}
		if backend == "DirectInput" {
			return guid != ""
		}
		return id >= 0 || guid != ""
	}
	// Once the device pickers exist, use their actual selected identities.
	// A legacy saved WinMM ID can exist even when the picker says None.
	picked := func(picker, id int, guid string) bool {
		if app.uiReady {
			return assigned(comboDevice(picker), comboGUID(picker))
		}
		return assigned(id, guid)
	}
	left := picked(idLeftDevice, c.LeftID, c.LeftDIGUID)
	right := picked(idRightDevice, c.RightID, c.RightDIGUID)
	stick := picked(idHotasStickDevice, c.HotasStickID, c.HotasStickDIGUID)
	infos := map[string]tuneInputInfo{
		profileHOSASLeft: {Enabled: left}, profileHOSASRight: {Enabled: right},
		profileHOSASThrust: {Enabled: left || right}, profileHOSASSteer: {Enabled: left || right},
		profileHOTASX: {Enabled: stick}, profileHOTASY: {Enabled: stick},
		profileHOTASThrottle:  {Enabled: picked(idHotasThrottleDevice, c.HotasThrottleID, c.HotasThrottleDIGUID)},
		profilePedal:          {Enabled: picked(idPedalDevice, c.PedalID, c.PedalDIGUID) && c.PedalMode != "Off", DigitalOnly: isTuningDigitalMode(c.PedalMode)},
		profileTwistLeft:      {Enabled: picked(idTwistLeftDevice, c.TwistLeft.DeviceID, c.TwistLeft.DIGUID) && c.TwistLeft.Mode != "Off", DigitalOnly: isTuningDigitalMode(c.TwistLeft.Mode), OneWay: normalizeAxisType(c.TwistLeft.AxisType) == "Lever"},
		profileTwistRight:     {Enabled: picked(idTwistRightDevice, c.TwistRight.DeviceID, c.TwistRight.DIGUID) && c.TwistRight.Mode != "Off", DigitalOnly: isTuningDigitalMode(c.TwistRight.Mode), OneWay: normalizeAxisType(c.TwistRight.AxisType) == "Lever"},
		profileTwist3:         {Enabled: picked(idTwist3Device, c.Twist3.DeviceID, c.Twist3.DIGUID) && c.Twist3.Mode != "Off", DigitalOnly: isTuningDigitalMode(c.Twist3.Mode), OneWay: normalizeAxisType(c.Twist3.AxisType) == "Lever"},
		profileTwist4:         {Enabled: picked(idTwist4Device, c.Twist4.DeviceID, c.Twist4.DIGUID) && c.Twist4.Mode != "Off", DigitalOnly: isTuningDigitalMode(c.Twist4.Mode), OneWay: normalizeAxisType(c.Twist4.AxisType) == "Lever"},
		profileTwist5:         {Enabled: picked(idTwist5Device, c.Twist5.DeviceID, c.Twist5.DIGUID) && c.Twist5.Mode != "Off", DigitalOnly: isTuningDigitalMode(c.Twist5.Mode), OneWay: normalizeAxisType(c.Twist5.AxisType) == "Lever"},
		profileTwist6:         {Enabled: picked(idTwist6Device, c.Twist6.DeviceID, c.Twist6.DIGUID) && c.Twist6.Mode != "Off", DigitalOnly: isTuningDigitalMode(c.Twist6.Mode), OneWay: normalizeAxisType(c.Twist6.AxisType) == "Lever"},
		profileWheelSteering:  {Enabled: picked(idWheelDevice, c.DrivingWheel.Wheel.DeviceID, c.DrivingWheel.Wheel.DIGUID)},
		profileWheelPedal1:    {Enabled: picked(idWheelPedal1Device, c.DrivingWheel.Pedal1.DeviceID, c.DrivingWheel.Pedal1.DIGUID)},
		profileWheelPedal2:    {Enabled: picked(idWheelPedal2Device, c.DrivingWheel.Pedal2.DeviceID, c.DrivingWheel.Pedal2.DIGUID)},
		profileWheelPedal3:    {Enabled: picked(idWheelPedal3Device, c.DrivingWheel.Pedal3.DeviceID, c.DrivingWheel.Pedal3.DIGUID) && c.DrivingWheel.Pedal3Mode != "Off", DigitalOnly: isTuningDigitalMode(c.DrivingWheel.Pedal3Mode)},
		profileWheelHandbrake: {Enabled: !c.DrivingWheel.HandbrakeButton && picked(idPedalDevice, c.DrivingWheel.Handbrake.DeviceID, c.DrivingWheel.Handbrake.DIGUID), DigitalOnly: true},
	}
	next := filterAxisTuneEntries(all, infos)
	identical := mode == p.controlMode && len(next) == len(p.entries)
	if identical {
		for i := range next {
			if next[i] != p.entries[i] {
				identical = false
				break
			}
		}
	}
	if identical {
		return
	}
	oldKey, _, _ := p.entry()
	if len(p.entries) > 0 {
		p.storeCurrent()
	}
	p.controlMode = mode
	p.isHOTAS = mode == "HOTAS"
	p.entries = next
	combo := p.controls[tuningComboID]
	procSendMessageW.Call(combo, CB_RESETCONTENT, 0, 0)
	for _, entry := range next {
		procSendMessageW.Call(combo, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(entry.Name))))
	}
	if len(next) == 0 {
		procEnableWindow.Call(combo, 0)
		procShowWindow.Call(p.controls[tuningEmptyLabelID], SW_SHOW)
		for id, h := range p.controls {
			if id != tuningComboID && id != tuningEmptyLabelID {
				procShowWindow.Call(h, SW_HIDE)
			}
		}
		p.selected = -1
		procInvalidateRect.Call(p.hwnd, 0, 1)
		return
	}
	procEnableWindow.Call(combo, 1)
	procShowWindow.Call(p.controls[tuningEmptyLabelID], SW_HIDE)
	for id, h := range p.controls {
		if id != tuningEmptyLabelID {
			procShowWindow.Call(h, SW_SHOW)
		}
	}
	idx := 0
	for i, e := range next {
		if e.Key == oldKey {
			idx = i
			break
		}
	}
	p.selectProfile(idx)
}

func (p *axisTuningPopup) switchMode(isHOTAS bool) {
	p.refreshEntriesFromConfig()
}
func (p *axisTuningPopup) switchModeName(mode string) {
	p.refreshEntriesFromConfig()
}

func (p *axisTuningPopup) storeCurrent() {
	if len(p.entries) == 0 {
		return
	}
	key, deadzoneOnly, _ := p.entry()
	old := checkedAxisTune(p.draft[key])
	old.Deadzone = p.sliderValue(tuningDeadzoneID)
	if !deadzoneOnly {
		old.FullOutput = p.sliderValue(tuningFullID)
		old.TwoStage = isChecked(p.controls[tuningTwoStageID])
		if old.TwoStage {
			old.Stage1Curve = float64(p.sliderValue(tuningCurveID)) / 100
			old.Stage1Weight = axisInt(p.sliderValue(tuningWeightID))
			old.Stage2Curve = float64(p.sliderValue(tuningStage2ID)) / 100
			old.Stage2Weight = axisInt(p.sliderValue(tuningStage2WID))
			old.CustomMidpoint = isChecked(p.controls[tuningCustomMidID])
			if old.CustomMidpoint {
				old.MidpointX = p.sliderValue(tuningMidXID)
				old.MidpointY = p.sliderValue(tuningMidYID)
			} else {
				old.Midpoint = p.sliderValue(tuningMidpointSliderID)
			}
		} else {
			old.Curve = float64(p.sliderValue(tuningCurveID)) / 100
			old.Weighting = axisInt(p.sliderValue(tuningWeightID))
		}
	}
	p.draft[key] = checkedAxisTune(old)
}
func (p *axisTuningPopup) selectProfile(i int) {
	if len(p.entries) == 0 {
		return
	}
	if i < 0 || i >= len(p.entries) {
		i = 0
	}
	p.suppressUI = true
	defer func() { p.suppressUI = false }()
	p.selected = i
	p.liveSign = 1
	setChecked(p.controls[tuningLiveCheckID], false)
	procSendMessageW.Call(p.controls[tuningComboID], CB_SETCURSEL, uintptr(i), 0)
	key, deadzoneOnly, _ := p.entry()
	tune := checkedAxisTune(p.draft[key])
	p.draft[key] = tune
	p.setSlider(tuningDeadzoneID, tune.Deadzone)
	setChecked(p.controls[tuningTwoStageID], tune.TwoStage)
	setChecked(p.controls[tuningCustomMidID], tune.CustomMidpoint)
	if tune.TwoStage {
		p.setSlider(tuningCurveID, int(math.Round(tune.Stage1Curve*100)))
		p.setSlider(tuningWeightID, displayAxisWeight(tune.Stage1Weight))
	} else {
		p.setSlider(tuningCurveID, int(math.Round(tune.Curve*100)))
		p.setSlider(tuningWeightID, displayAxisWeight(tune.Weighting))
	}
	p.setSlider(tuningStage2ID, int(math.Round(tune.Stage2Curve*100)))
	p.setSlider(tuningStage2WID, displayAxisWeight(tune.Stage2Weight))
	p.setSlider(tuningFullID, tune.FullOutput)
	p.setSlider(tuningMidXID, tune.MidpointX)
	p.setSlider(tuningMidYID, tune.MidpointY)
	p.setSlider(tuningMidpointSliderID, tune.Midpoint)
	app.cfgMu.RLock()
	selectedConfig := app.cfg
	app.cfgMu.RUnlock()
	digitalOnly := isTuningDigitalModeForEntry(key, p.controlMode, selectedConfig)
	p.layoutProfile(deadzoneOnly, digitalOnly)
	if digitalOnly {
		setText(p.controls[tuningDescriptionID], "Digital output: only the activation deadzone applies.")
	} else if deadzoneOnly {
		setText(p.controls[tuningDescriptionID], "Independent input deadzone only. Curves belong to combined Thrust and Steering outputs.")
	} else if key == profileHOTASThrottle {
		setText(p.controls[tuningDescriptionID], "Throttle shaping affects acceleration; existing brake threshold and brake zone remain unchanged.")
	} else if key == profileHOSASThrust || key == profileHOSASSteer {
		setText(p.controls[tuningDescriptionID], "Applied after differential mixing so matched stick movement does not cause unintended turning.")
	} else {
		setText(p.controls[tuningDescriptionID], "Smooth response curves apply after deadzone; direction and inversion are preserved.")
	}
	p.updateValues()
}
func (p *axisTuningPopup) currentValue() axisTuneProfile {
	if len(p.entries) == 0 {
		return defaultAxisTune(0)
	}
	key, _, _ := p.entry()
	// The draft is committed on every calibration edit, including text edits.
	return checkedAxisTune(p.draft[key])
}
func (p *axisTuningPopup) updateValues() {
	if len(p.entries) == 0 {
		return
	}
	profile := p.currentValue()
	setText(p.controls[tuningDeadzoneValueID], fmt.Sprintf("%d%%", profile.Deadzone))
	value := profile.Curve
	weight := displayAxisWeight(profile.Weighting)
	if profile.TwoStage {
		value = profile.Stage1Curve
		weight = displayAxisWeight(profile.Stage1Weight)
	}
	setText(p.controls[tuningCurveLabelID], map[bool]string{true: "Curve 1", false: "Response Curve"}[profile.TwoStage])
	setText(p.controls[tuningCurveValueID], fmt.Sprintf("%.2f", value))
	setText(p.controls[tuningWeightValueID], fmt.Sprintf("%d%%", weight))
	setText(p.controls[tuningStage2ValueID], fmt.Sprintf("%.2f", profile.Stage2Curve))
	setText(p.controls[tuningStage2WValueID], fmt.Sprintf("%d%%", displayAxisWeight(profile.Stage2Weight)))
	setText(p.controls[tuningMidpointPercentID], fmt.Sprintf("%d%%", profile.Midpoint))
	setText(p.controls[tuningMidXValueID], fmt.Sprintf("%d%%", profile.MidpointX))
	setText(p.controls[tuningMidYValueID], fmt.Sprintf("%d%%", profile.MidpointY))
	setText(p.controls[tuningFullValueID], fmt.Sprintf("%d%%", profile.FullOutput))
	test := p.sliderValue(tuningTestID)
	_, _, oneWay := p.entry()
	val := float64(test) / 100
	if isChecked(p.controls[tuningLiveCheckID]) && !oneWay && p.liveSign < 0 {
		val = -val
	}
	out := tunedCentered(val, profile)
	if oneWay {
		out = tunedOneWay(val, profile)
	}
	setText(p.controls[tuningTestValueID], fmt.Sprintf("%.0f%% -> %.0f%%", val*100, out*100))
	procInvalidateRect.Call(p.hwnd, 0, 0)
}
func (p *axisTuningPopup) applyDraft() {
	if len(p.entries) == 0 {
		return
	}
	p.storeCurrent()
	app.cfgMu.Lock()
	updated := app.cfg
	updated.Tuning = cloneAxisProfiles(app.cfg.Tuning)
	initAxisProfiles(&updated)
	for key, value := range p.draft {
		updated.Tuning[key] = checkedAxisTune(value)
	}
	app.cfg.Tuning = updated.Tuning
	// Retain the legacy fields so a downgrade can still read safe values.
	app.cfg.Deadzone = app.cfg.Tuning[profileHOSASLeft].Deadzone
	app.cfg.TwistLeft.Deadzone = app.cfg.Tuning[profileTwistLeft].Deadzone
	app.cfg.TwistRight.Deadzone = app.cfg.Tuning[profileTwistRight].Deadzone
	app.cfg.Twist3.Deadzone = app.cfg.Tuning[profileTwist3].Deadzone
	app.cfg.Twist4.Deadzone = app.cfg.Tuning[profileTwist4].Deadzone
	app.cfg.Twist5.Deadzone = app.cfg.Tuning[profileTwist5].Deadzone
	app.cfg.Twist6.Deadzone = app.cfg.Tuning[profileTwist6].Deadzone
	app.cfgMu.Unlock()
	// Save to memory on every edit and commit the JSON after the slider settles.
	app.tuningDirty = true
	app.tuningSaveDue = time.Now().Add(450 * time.Millisecond)
}
func (p *axisTuningPopup) command(id, notify int) {
	if p.suppressUI {
		return
	}
	switch id {
	case tuningLiveCheckID:
		if notify == BN_CLICKED {
			p.liveSign = 1
			p.setManualTravelEnabled()
			p.updateValues()
		}
	case tuningComboID:
		if notify == CBN_SELCHANGE {
			p.storeCurrent()
			v, _, _ := procSendMessageW.Call(p.controls[tuningComboID], CB_GETCURSEL, 0, 0)
			p.selectProfile(int(v))
		}
	case tuningTwoStageID, tuningCustomMidID:
		if notify == BN_CLICKED && len(p.entries) > 0 {
			key, _, _ := p.entry()
			draft := checkedAxisTune(p.draft[key])
			if id == tuningTwoStageID {
				draft.TwoStage = isChecked(p.controls[tuningTwoStageID])
			} else {
				draft.CustomMidpoint = isChecked(p.controls[tuningCustomMidID])
				if !draft.CustomMidpoint {
					draft.Midpoint = draft.MidpointX
				}
				// Re-enabling Custom restores the previous saved X/Y positions.
			}
			p.draft[key] = draft
			p.selectProfile(p.selected)
			p.applyDraft()
		}
	case tuningResetID:
		if notify == BN_CLICKED && len(p.entries) > 0 {
			key, _, _ := p.entry()
			dz := 0
			switch key {
			case profileHOSASLeft, profileHOSASRight, profileHOTASX, profileHOTASY, profilePedal, profileTwistLeft, profileTwistRight, profileTwist3, profileTwist4, profileTwist5, profileTwist6:
				dz = 6
			}
			p.draft[key] = defaultAxisTune(dz)
			p.selectProfile(p.selected)
			p.applyDraft()
		}
	}
}
func axisTuningWndProc(hwnd uintptr, m uint32, w, l uintptr) uintptr {
	p := app.tuning
	switch m {
	case WM_CTLCOLORSTATIC:
		// Keep embedded curve-page label text consistent with its system
		// button-face background, just like the outer tab's STATIC labels.
		procSetBkMode.Call(w, TRANSPARENT)
		face, _, _ := procGetSysColorBrush.Call(COLOR_BTNFACE)
		color, _, _ := procGetSysColor.Call(COLOR_BTNFACE)
		procSetBkColor.Call(w, color)
		return face
	case WM_COMMAND:
		if p != nil && p.hwnd == hwnd {
			p.command(loword(w), hiword(w))
		}
		return 0
	case WM_HSCROLL:
		if p != nil && p.hwnd == hwnd && len(p.entries) > 0 && !p.suppressUI {
			// Test travel is preview-only; all other sliders auto-apply.
			if l != p.controls[tuningTestID] {
				p.applyDraft()
			}
			p.updateValues()
		}
		return 0
	case WM_PAINT:
		if p != nil && p.hwnd == hwnd {
			p.paintGraph()
			return 0
		}
	case WM_CLOSE:
		procDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		if p != nil && p.hwnd == hwnd {
			app.tuning = nil
		}
		return 0
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(m), w, l)
	return result
}

func colorRGB(r, g, b uint32) uintptr { return uintptr(r | (g << 8) | (b << 16)) }
func (p *axisTuningPopup) paintGraph() {
	var ps tuningPaintStruct
	dc, _, _ := procBeginPaint.Call(p.hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	defer procEndPaint.Call(p.hwnd, uintptr(unsafe.Pointer(&ps)))
	if len(p.entries) == 0 {
		return
	}
	key, _, _ := p.entry()
	app.cfgMu.RLock()
	c := app.cfg
	app.cfgMu.RUnlock()
	if isTuningDigitalModeForEntry(key, p.controlMode, c) {
		return
	}
	graph := rect{Left: 0, Top: int32(p.graphTop), Right: 557, Bottom: int32(p.graphBottom)}
	if p.graphBottom <= p.graphTop {
		return
	}
	white, _, _ := procCreateSolidBrush.Call(colorRGB(255, 255, 255))
	procFillRect.Call(dc, uintptr(unsafe.Pointer(&graph)), white)
	procDeleteObject.Call(white)
	draw := func(points [][2]int, clr uintptr, width int) {
		pen, _, _ := procCreatePen.Call(0, uintptr(width), clr)
		old, _, _ := procSelectObject.Call(dc, pen)
		for i, pt := range points {
			if i == 0 {
				procMoveToEx.Call(dc, uintptr(pt[0]), uintptr(pt[1]), 0)
			} else {
				procLineTo.Call(dc, uintptr(pt[0]), uintptr(pt[1]))
			}
		}
		procSelectObject.Call(dc, old)
		procDeleteObject.Call(pen)
	}
	x0, y0, x1, y1 := int(graph.Left), int(graph.Bottom), int(graph.Right), int(graph.Top)
	// Grid and axes. X is travel 0..100%, Y is output 0..100%.
	for i := 0; i <= 4; i++ {
		x := x0 + (x1-x0)*i/4
		y := y0 - (y0-y1)*i/4
		draw([][2]int{{x, y1}, {x, y0}}, colorRGB(224, 228, 233), 1)
		draw([][2]int{{x0, y}, {x1, y}}, colorRGB(224, 228, 233), 1)
	}
	draw([][2]int{{x0, y0}, {x1, y1}}, colorRGB(158, 164, 173), 1)
	tune := p.currentValue()
	_, _, oneWay := p.entry()
	getOutput := func(frac float64) float64 {
		if oneWay {
			return tunedOneWay(frac, tune)
		}
		return tunedCentered(frac, tune)
	}
	// Single curve is blue. Two stage retains the same blue, with the second
	// segment the user-selected orangey yellow (#E08D33).
	curvePoints := func(from, to float64) [][2]int {
		path := make([][2]int, 0, 151)
		for i := 0; i <= 150; i++ {
			frac := from + (to-from)*float64(i)/150
			x := x0 + int(math.Round(float64(x1-x0)*frac))
			y := y0 - int(math.Round(float64(y0-y1)*getOutput(frac)))
			path = append(path, [2]int{x, y})
		}
		return path
	}
	if tune.TwoStage {
		mx := tune.Midpoint
		if tune.CustomMidpoint {
			mx = tune.MidpointX
		}
		cut := float64(tune.FullOutput) * float64(mx) / 10000
		draw(curvePoints(0, cut), colorRGB(52, 126, 212), 3)
		draw(curvePoints(cut, 1), colorRGB(224, 141, 51), 3)
		markerX := x0 + int(math.Round(float64(x1-x0)*cut))
		markerY := y0 - int(math.Round(float64(y0-y1)*getOutput(cut)))
		brush, _, _ := procCreateSolidBrush.Call(colorRGB(255, 255, 255))
		old, _, _ := procSelectObject.Call(dc, brush)
		procEllipse.Call(dc, uintptr(markerX-4), uintptr(markerY-4), uintptr(markerX+4), uintptr(markerY+4))
		procSelectObject.Call(dc, old)
		procDeleteObject.Call(brush)
	} else {
		draw(curvePoints(0, 1), colorRGB(52, 126, 212), 3)
	}
	test := p.sliderValue(tuningTestID)
	markerX := x0 + (x1-x0)*test/100
	markerY := y0 - int(math.Round(float64(y0-y1)*getOutput(float64(test)/100)))
	brush, _, _ := procCreateSolidBrush.Call(colorRGB(20, 113, 189))
	old, _, _ := procSelectObject.Call(dc, brush)
	procEllipse.Call(dc, uintptr(markerX-5), uintptr(markerY-5), uintptr(markerX+5), uintptr(markerY+5))
	procSelectObject.Call(dc, old)
	procDeleteObject.Call(brush)
}

// Keep the manual travel slider available only when live mode is off.
func (p *axisTuningPopup) setManualTravelEnabled() {
	if p == nil {
		return
	}
	manual := !isChecked(p.controls[tuningLiveCheckID])
	v := uintptr(0)
	if manual {
		v = 1
	}
	procEnableWindow.Call(p.controls[tuningTestID], v)
}

// Called by the existing 10 Hz UI timer. Neither the selected input's state
// nor calibration JSON is changed by the preview.
func (p *axisTuningPopup) refreshLiveInput() {
	if p == nil || p.hwnd == 0 || len(p.entries) == 0 || app.activeRightTab != 1 || !isChecked(p.controls[tuningLiveCheckID]) {
		return
	}
	key, _, oneWay := p.entry()
	app.cfgMu.RLock()
	c := app.cfg
	app.cfgMu.RUnlock()
	if isTuningDigitalModeForEntry(key, p.controlMode, c) {
		return
	}
	var v float64
	var ok bool
	if key == profileHOSASThrust || key == profileHOSASSteer {
		app.monitorMu.RLock()
		m := app.monitor
		app.monitorMu.RUnlock()
		if time.Since(m.LastUpdate) < 500*time.Millisecond {
			if key == profileHOSASThrust {
				v = calculateDrive(m.Left, m.Right, c.PreserveThrust)
				if normalizeControlMode(c.ControlMode) == "HOSAS" {
					v = hosasDrive(m.Left, m.Right, v, c.LeftAvoidDrive, c.RightAvoidDrive)
				}
			} else {
				v = clamp(m.Left-m.Right, -1, 1)
			}
			ok = true
		}
	} else {
		backend := normalizeInputBackend(c.InputBackend)
		guid := ""
		id, axis := -1, 0
		invert := false
		throttleHOSAS := false
		var aux *auxiliaryAxisConfig
		switch key {
		case profileHOSASLeft:
			guid, id, axis, invert, throttleHOSAS = c.LeftDIGUID, c.LeftID, c.LeftAxis, c.LeftInvert, c.LeftThrottleMode
		case profileHOSASRight:
			guid, id, axis, invert, throttleHOSAS = c.RightDIGUID, c.RightID, c.RightAxis, c.RightInvert, c.RightThrottleMode
		case profileHOTASX:
			guid, id, axis, invert = c.HotasStickDIGUID, c.HotasStickID, c.HotasStickXAxis, c.HotasStickInvertX
		case profileHOTASY:
			guid, id, axis, invert = c.HotasStickDIGUID, c.HotasStickID, c.HotasStickYAxis, c.HotasStickInvertY
		case profileHOTASThrottle:
			guid, id, axis, invert = c.HotasThrottleDIGUID, c.HotasThrottleID, c.HotasThrottleAxis, c.HotasThrottleInvert
		case profilePedal:
			guid, id, axis, invert = c.PedalDIGUID, c.PedalID, c.PedalAxis, c.PedalInvert
		case profileTwistLeft:
			aux = &c.TwistLeft
		case profileTwistRight:
			aux = &c.TwistRight
		case profileTwist3:
			aux = &c.Twist3
		case profileTwist4:
			aux = &c.Twist4
		case profileTwist5:
			aux = &c.Twist5
		case profileTwist6:
			aux = &c.Twist6
		case profileWheelSteering:
			guid, id, axis, invert = c.DrivingWheel.Wheel.DIGUID, c.DrivingWheel.Wheel.DeviceID, c.DrivingWheel.Wheel.Axis, c.DrivingWheel.Wheel.Invert
		case profileWheelPedal1:
			guid, id, axis, invert = c.DrivingWheel.Pedal1.DIGUID, c.DrivingWheel.Pedal1.DeviceID, c.DrivingWheel.Pedal1.Axis, c.DrivingWheel.Pedal1.Invert
		case profileWheelPedal2:
			guid, id, axis, invert = c.DrivingWheel.Pedal2.DIGUID, c.DrivingWheel.Pedal2.DeviceID, c.DrivingWheel.Pedal2.Axis, c.DrivingWheel.Pedal2.Invert
		case profileWheelPedal3:
			guid, id, axis, invert = c.DrivingWheel.Pedal3.DIGUID, c.DrivingWheel.Pedal3.DeviceID, c.DrivingWheel.Pedal3.Axis, c.DrivingWheel.Pedal3.Invert
		case profileWheelHandbrake:
			guid, id, axis, invert = c.DrivingWheel.Handbrake.DIGUID, c.DrivingWheel.Handbrake.DeviceID, c.DrivingWheel.Handbrake.Axis, c.DrivingWheel.Handbrake.Invert
		}
		if aux != nil {
			guid, id, axis = aux.DIGUID, aux.DeviceID, aux.Axis
		}
		states := map[int]joyInfoEx{}
		valid := map[int]bool{}
		if id >= 0 {
			states[id], valid[id] = readJoy(id)
		}
		if throttleHOSAS {
			v, ok = app.readConfiguredThrottleCentered(backend, guid, id, axis, invert, states, valid)
		} else {
			v, ok = app.readConfiguredAxis(backend, guid, id, axis, invert, states, valid)
		}
		if aux != nil && ok {
			v = normalizeAuxiliaryValue(v, *aux, 0)
		}
		if ok && oneWay && aux == nil {
			v = clamp((v+1)/2, 0, 1)
		}
	}
	if !ok {
		setText(p.controls[tuningTestValueID], "Waiting for assigned device...")
		return
	}
	v = clamp(v, -1, 1)
	p.liveSign = 1
	if v < 0 {
		p.liveSign = -1
	}
	p.setSlider(tuningTestID, int(math.Round(math.Abs(v)*100)))
	p.updateValues()
}

func isTuningDigitalModeForEntry(key, mode string, c config) bool {
	switch key {
	case profilePedal:
		return isTuningDigitalMode(c.PedalMode)
	case profileTwistLeft:
		return isTuningDigitalMode(c.TwistLeft.Mode)
	case profileTwistRight:
		return isTuningDigitalMode(c.TwistRight.Mode)
	case profileTwist3:
		return isTuningDigitalMode(c.Twist3.Mode)
	case profileTwist4:
		return isTuningDigitalMode(c.Twist4.Mode)
	case profileTwist5:
		return isTuningDigitalMode(c.Twist5.Mode)
	case profileTwist6:
		return isTuningDigitalMode(c.Twist6.Mode)
	case profileWheelPedal3:
		return isTuningDigitalMode(c.DrivingWheel.Pedal3Mode)
	case profileWheelHandbrake:
		return true
	}
	return false
}
