//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

// Light / Dark / System appearance for the classic Win32 controls this app
// uses. Light Mode is the original look: every hook in this file returns
// early or falls through to the existing drawing while darkThemeActive is
// false. Dark Mode recolours what classic controls let the parent colour
// (labels, checkbox text, edits, dropdown lists, slider and progress-bar
// backgrounds), owner-draws push buttons and paints the tab strip.

var (
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procRegGetValueW          = advapi32.NewProc("RegGetValueW")
	procEnumChildWindows      = user32.NewProc("EnumChildWindows")
	procGetClassNameW         = user32.NewProc("GetClassNameW")
	procGetWindowLongPtrW     = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW     = user32.NewProc("SetWindowLongPtrW")
	procSetClassLongPtrW      = user32.NewProc("SetClassLongPtrW")
	procCallWindowProcW       = user32.NewProc("CallWindowProcW")

	themeChildCallback = syscall.NewCallback(themeChildProc)
	themeTabCallback   = syscall.NewCallback(themeTabProc)
)

const (
	GWL_STYLE          = -16
	GWLP_WNDPROC       = -4
	GCLP_HBRBACKGROUND = -10
	WM_ERASEBKGND      = 0x0014
	WM_SETTINGCHANGE   = 0x001A
	WM_DRAWITEM        = 0x002B
	WM_GETFONT         = 0x0031
	WM_CTLCOLOREDIT    = 0x0133
	WM_CTLCOLORLISTBOX = 0x0134
	WM_CTLCOLORBTN     = 0x0135
	BS_TYPEMASK        = 0x000F
	BS_DEFPUSHBUTTON   = 0x0001
	BS_OWNERDRAW       = 0x000B
	ODT_BUTTON         = 4
	ODS_SELECTED       = 0x0001
	ODS_DISABLED       = 0x0004
	ODS_FOCUS          = 0x0010
	DT_CENTER          = 0x0001
	DT_VCENTER         = 0x0004
	DT_SINGLELINE      = 0x0020
	DT_CALCRECT        = 0x0400
	RDW_FRAME          = 0x0400
	SWP_FRAMECHANGED   = 0x0020
	SWP_NOMOVE         = 0x0002
	PBM_SETBKCOLOR     = 0x2001
	CLR_DEFAULT        = 0xFF000000
	TCM_GETITEMCOUNT   = TCM_FIRST + 4
	TCM_GETITEMRECT    = TCM_FIRST + 10
	TCM_GETITEMW       = TCM_FIRST + 60

	hkeyCurrentUser  = 0x80000001
	rrfRtRegDword    = 0x00000010
	dwmDarkTitleBar  = 20 // DWMWA_USE_IMMERSIVE_DARK_MODE
	dwmDarkTitleOld  = 19 // the same attribute before Windows 10 20H1
	personalizeKey   = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	appsUseLightName = "AppsUseLightTheme"
)

// The Theme dropdown's control ID, outside every other ID range.
const idTheme = 1980

// createThemeSelector places the Theme dropdown just left of the bottom-right
// action buttons, on the same row wherever the footer is laid out.
func (a *appState) createThemeSelector() {
	footer, ok := a.baseRects[a.controls[idTestA]]
	if !ok {
		return
	}
	right, y := int(footer.Left)-14, int(footer.Top)
	a.createLabel("Theme", right-150-56, y+8, 52, 22)
	a.createCombo(right-150, y+4, 150, 120, idTheme)
}

// uiPalette holds every colour the app paints itself, plus Dark Mode colours
// for classic controls.
type uiPalette struct {
	Window, Panel, Text, DisabledText, EditBack, Line, PlotBack        uintptr
	ButtonFace, ButtonPressed, ButtonBorder, ButtonText, TabInactive   uintptr
	HelpShadowOuter, HelpShadowInner, HelpCard, HelpBorder, HelpText   uintptr
	GraphBack, GraphGrid, GraphDiagonal, GraphMarker, MonitorDotBorder uintptr
}

// lightPalette is the original look. Only the colours the app paints itself
// (help panel, Axis Tuning graph, monitor dots) are read in Light Mode.
func lightPalette() uiPalette {
	panel, _, _ := procGetSysColor.Call(COLOR_BTNFACE)
	return uiPalette{
		Window: mainBackgroundColor, Panel: panel, Text: colorRGB(0, 0, 0),
		HelpShadowOuter: colorRGB(207, 207, 207), HelpShadowInner: colorRGB(220, 220, 220),
		HelpCard: colorRGB(255, 255, 255), HelpBorder: colorRGB(199, 199, 199), HelpText: colorRGB(30, 30, 30),
		GraphBack: colorRGB(255, 255, 255), GraphGrid: colorRGB(224, 228, 233), GraphDiagonal: colorRGB(158, 164, 173),
		GraphMarker: colorRGB(255, 255, 255), MonitorDotBorder: colorRGB(0, 0, 0),
	}
}

// darkPalette mirrors the light tones: the main window is a mid-dark grey
// like the light #D4D4D4, and tab pages and the help card are darker, as
// their light versions are lighter.
func darkPalette() uiPalette {
	return uiPalette{
		Window: colorRGB(43, 43, 43), Panel: colorRGB(30, 30, 30),
		Text: colorRGB(230, 230, 230), DisabledText: colorRGB(130, 130, 130),
		EditBack: colorRGB(24, 24, 24), Line: colorRGB(140, 140, 140), PlotBack: colorRGB(30, 30, 30),
		ButtonFace: colorRGB(60, 60, 60), ButtonPressed: colorRGB(38, 79, 120),
		ButtonBorder: colorRGB(100, 100, 100), ButtonText: colorRGB(235, 235, 235), TabInactive: colorRGB(52, 52, 52),
		HelpShadowOuter: colorRGB(36, 36, 36), HelpShadowInner: colorRGB(32, 32, 32),
		HelpCard: colorRGB(24, 24, 24), HelpBorder: colorRGB(70, 70, 70), HelpText: colorRGB(225, 225, 225),
		GraphBack: colorRGB(24, 24, 24), GraphGrid: colorRGB(55, 60, 66), GraphDiagonal: colorRGB(105, 110, 118),
		GraphMarker: colorRGB(24, 24, 24), MonitorDotBorder: colorRGB(235, 235, 235),
	}
}

var (
	currentPalette  = lightPalette()
	darkThemeActive bool
	themeApplied    bool
	themeBrushes    = map[uintptr]uintptr{}
	// Controls changed for Dark Mode, so Light Mode can restore them exactly.
	themeButtonTypes = map[uintptr]uintptr{} // push buttons drawn by themeDrawButton
	themeLines       = map[uintptr]bool{}    // 1px plot grid lines
	themePlotFrames  = map[uintptr]bool{}    // live monitor plot boxes
	themePushLike    = map[uintptr]bool{}    // Profile buttons keep classic drawing
	themeTabProcs    = map[uintptr]uintptr{} // original tab control window procedures
)

func winIndex(i int) uintptr { return uintptr(i) }

func themeBrush(color uintptr) uintptr {
	if b, ok := themeBrushes[color]; ok {
		return b
	}
	b, _, _ := procCreateSolidBrush.Call(color)
	themeBrushes[color] = b
	return b
}

func windowStyle(h uintptr) uintptr {
	s, _, _ := procGetWindowLongPtrW.Call(h, winIndex(GWL_STYLE))
	return s
}

func setWindowStyle(h, style uintptr) {
	procSetWindowLongPtrW.Call(h, winIndex(GWL_STYLE), style)
}

func windowClassName(h uintptr) string {
	var buf [64]uint16
	n, _, _ := procGetClassNameW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

// systemPrefersDark reads Windows Settings > Personalization > Colors >
// "Choose your default app mode". Missing on older Windows: light.
func systemPrefersDark() bool {
	if procRegGetValueW.Find() != nil {
		return false
	}
	var value, size uint32 = 1, 4
	r, _, _ := procRegGetValueW.Call(hkeyCurrentUser,
		uintptr(unsafe.Pointer(utf16ptr(personalizeKey))), uintptr(unsafe.Pointer(utf16ptr(appsUseLightName))),
		rrfRtRegDword, 0, uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&size)))
	return r == 0 && value == 0
}

func setTitleBarDark(hwnd uintptr, dark bool) {
	if procDwmSetWindowAttribute.Find() != nil {
		return
	}
	var v int32
	if dark {
		v = 1
	}
	r, _, _ := procDwmSetWindowAttribute.Call(hwnd, dwmDarkTitleBar, uintptr(unsafe.Pointer(&v)), 4)
	if r != 0 {
		procDwmSetWindowAttribute.Call(hwnd, dwmDarkTitleOld, uintptr(unsafe.Pointer(&v)), 4)
	}
}

// applyTheme switches to the saved theme. It is cheap to call repeatedly:
// nothing happens unless Light/Dark actually changes, so it also handles
// Windows changing its app mode while System Specified is chosen.
func (a *appState) applyTheme() {
	a.cfgMu.RLock()
	theme := a.cfg.Theme
	a.cfgMu.RUnlock()
	dark := useDarkTheme(theme, systemPrefersDark())
	if themeApplied && dark == darkThemeActive {
		return
	}
	first := !themeApplied
	themeApplied = true
	darkThemeActive = dark
	if dark {
		currentPalette = darkPalette()
	} else {
		currentPalette = lightPalette()
	}
	if first && !dark {
		return // the original look needs no changes
	}
	setTitleBarDark(a.hwnd, dark)
	background := mainBackgroundBrush
	if dark {
		background = themeBrush(currentPalette.Window)
	}
	procSetClassLongPtrW.Call(a.hwnd, winIndex(GCLP_HBRBACKGROUND), background)
	if a.tuning != nil && a.tuning.hwnd != 0 {
		tuningBackground := uintptr(COLOR_BTNFACE + 1)
		if dark {
			tuningBackground = themeBrush(currentPalette.Panel)
		}
		procSetClassLongPtrW.Call(a.tuning.hwnd, winIndex(GCLP_HBRBACKGROUND), tuningBackground)
	}
	procEnumChildWindows.Call(a.hwnd, themeChildCallback, 0)
	procRedrawWindow.Call(a.hwnd, 0, 0, RDW_INVALIDATE|RDW_ERASE|RDW_ALLCHILDREN|RDW_FRAME|RDW_UPDATENOW)
}

// themeChildProc adjusts one control (EnumChildWindows also visits the
// Axis Tuning page's controls) for the current theme.
func themeChildProc(h, _ uintptr) uintptr {
	style := windowStyle(h)
	switch class := windowClassName(h); {
	case strings.EqualFold(class, "Button"):
		if style&BS_PUSHLIKE != 0 {
			themePushLike[h] = true
		}
		original, switched := themeButtonTypes[h]
		kind := style & BS_TYPEMASK
		if darkThemeActive && !switched && (kind == BS_PUSHBUTTON || kind == BS_DEFPUSHBUTTON) {
			themeButtonTypes[h] = kind
			setWindowStyle(h, style&^BS_TYPEMASK|BS_OWNERDRAW)
		} else if !darkThemeActive && switched {
			setWindowStyle(h, style&^BS_TYPEMASK|original)
			delete(themeButtonTypes, h)
		}
	case strings.EqualFold(class, "Static"):
		if !themeLines[h] && !themePlotFrames[h] && style&WS_BORDER != 0 {
			var rc rect
			procGetWindowRect.Call(h, uintptr(unsafe.Pointer(&rc)))
			if rc.Right-rc.Left <= 1 || rc.Bottom-rc.Top <= 1 {
				themeLines[h] = true
			} else {
				themePlotFrames[h] = true
			}
		}
		// A black WS_BORDER line would vanish on a dark plot, so Dark Mode
		// drops the border and fills the 1px control with the line colour.
		if themeLines[h] {
			if darkThemeActive {
				setWindowStyle(h, style&^WS_BORDER)
			} else {
				setWindowStyle(h, style|WS_BORDER)
			}
			procSetWindowPos.Call(h, 0, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE|SWP_FRAMECHANGED)
		}
	case strings.EqualFold(class, "msctls_progress32"):
		background := uintptr(CLR_DEFAULT)
		if darkThemeActive {
			background = currentPalette.PlotBack
		}
		procSendMessageW.Call(h, PBM_SETBKCOLOR, 0, background)
	case strings.EqualFold(class, "SysTabControl32"):
		if _, ok := themeTabProcs[h]; !ok {
			old, _, _ := procSetWindowLongPtrW.Call(h, winIndex(GWLP_WNDPROC), themeTabCallback)
			themeTabProcs[h] = old
		}
	}
	return 1
}

// themeCtlColor handles WM_CTLCOLOR* in Dark Mode. It returns 0 in Light
// Mode, or to keep a control's default drawing.
func themeCtlColor(msg uint32, dc, ctrl uintptr, panel bool) uintptr {
	if !darkThemeActive {
		return 0
	}
	p := currentPalette
	background := p.Window
	if panel {
		background = p.Panel
	}
	switch msg {
	case WM_CTLCOLORSTATIC:
		switch {
		case themePushLike[ctrl]:
			return 0 // classic push-like face is light, so keep its dark text
		case themeLines[ctrl]:
			procSetBkColor.Call(dc, p.Line)
			return themeBrush(p.Line)
		case themePlotFrames[ctrl]:
			procSetBkColor.Call(dc, p.PlotBack)
			return themeBrush(p.PlotBack)
		}
		procSetTextColor.Call(dc, p.Text)
		procSetBkMode.Call(dc, TRANSPARENT)
		procSetBkColor.Call(dc, background)
		return themeBrush(background)
	case WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		procSetTextColor.Call(dc, p.Text)
		procSetBkColor.Call(dc, p.EditBack)
		return themeBrush(p.EditBack)
	case WM_CTLCOLORBTN:
		procSetBkColor.Call(dc, background)
		return themeBrush(background)
	}
	return 0
}

type drawItemStruct struct {
	CtlType, CtlID, ItemID, ItemAction, ItemState uint32
	HwndItem, HDC                                 uintptr
	RcItem                                        rect
	ItemData                                      uintptr
}

// themeDrawButton paints a Dark Mode push button for WM_DRAWITEM.
func themeDrawButton(lParam uintptr) bool {
	if lParam == 0 {
		return false
	}
	d := (*drawItemStruct)(unsafe.Pointer(lParam))
	if d.CtlType != ODT_BUTTON {
		return false
	}
	if _, ok := themeButtonTypes[d.HwndItem]; !ok {
		return false
	}
	p := currentPalette
	rc := d.RcItem
	face, text, border := p.ButtonFace, p.ButtonText, p.ButtonBorder
	if d.ItemState&ODS_SELECTED != 0 {
		face = p.ButtonPressed // also the lit state of the mapping tiles
	}
	if d.ItemState&ODS_DISABLED != 0 {
		text = p.DisabledText
	}
	if d.ItemState&ODS_FOCUS != 0 {
		border = p.Text
	}
	procFillRect.Call(d.HDC, uintptr(unsafe.Pointer(&rc)), themeBrush(face))
	procFrameRect.Call(d.HDC, uintptr(unsafe.Pointer(&rc)), themeBrush(border))
	var buf [256]uint16
	n, _, _ := procGetWindowTextW.Call(d.HwndItem, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return true
	}
	if font, _, _ := procSendMessageW.Call(d.HwndItem, WM_GETFONT, 0, 0); font != 0 {
		old, _, _ := procSelectObject.Call(d.HDC, font)
		defer procSelectObject.Call(d.HDC, old)
	}
	procSetBkMode.Call(d.HDC, TRANSPARENT)
	procSetTextColor.Call(d.HDC, text)
	inner := rect{Left: rc.Left + 4, Top: rc.Top + 2, Right: rc.Right - 4, Bottom: rc.Bottom - 2}
	if windowStyle(d.HwndItem)&BS_MULTILINE != 0 {
		measured := inner
		procDrawTextW.Call(d.HDC, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&measured)), DT_CENTER|DT_WORDBREAK|DT_CALCRECT)
		if h := measured.Bottom - measured.Top; h < inner.Bottom-inner.Top {
			inner.Top += (inner.Bottom - inner.Top - h) / 2
		}
		procDrawTextW.Call(d.HDC, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&inner)), DT_CENTER|DT_WORDBREAK)
	} else {
		procDrawTextW.Call(d.HDC, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&inner)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}
	return true
}

// themeTabProc paints the Mappings / Axis Tuning tab strip in Dark Mode and
// leaves every message to the original control in Light Mode.
func themeTabProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if darkThemeActive {
		switch msg {
		case WM_ERASEBKGND:
			return 1
		case WM_PAINT:
			paintDarkTabs(hwnd)
			return 0
		}
	}
	r, _, _ := procCallWindowProcW.Call(themeTabProcs[hwnd], hwnd, uintptr(msg), wParam, lParam)
	return r
}

func paintDarkTabs(hwnd uintptr) {
	var ps tuningPaintStruct
	dc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	p := currentPalette
	var client rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	procFillRect.Call(dc, uintptr(unsafe.Pointer(&client)), themeBrush(p.Window))
	count, _, _ := procSendMessageW.Call(hwnd, TCM_GETITEMCOUNT, 0, 0)
	selected, _, _ := procSendMessageW.Call(hwnd, TCM_GETCURSEL, 0, 0)
	tabs := make([]rect, count)
	body := client
	for i := range tabs {
		procSendMessageW.Call(hwnd, TCM_GETITEMRECT, uintptr(i), uintptr(unsafe.Pointer(&tabs[i])))
		if tabs[i].Bottom > body.Top {
			body.Top = tabs[i].Bottom
		}
	}
	procFillRect.Call(dc, uintptr(unsafe.Pointer(&body)), themeBrush(p.Panel))
	procFrameRect.Call(dc, uintptr(unsafe.Pointer(&body)), themeBrush(p.Line))
	if font, _, _ := procSendMessageW.Call(hwnd, WM_GETFONT, 0, 0); font != 0 {
		old, _, _ := procSelectObject.Call(dc, font)
		defer procSelectObject.Call(dc, old)
	}
	procSetBkMode.Call(dc, TRANSPARENT)
	procSetTextColor.Call(dc, p.Text)
	for i, r := range tabs {
		fill := p.TabInactive
		if uintptr(i) == selected {
			fill = p.Panel
			r.Top -= 2 // the selected tab stands out, as in the classic strip
		}
		procFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), themeBrush(fill))
		procFrameRect.Call(dc, uintptr(unsafe.Pointer(&r)), themeBrush(p.Line))
		if uintptr(i) == selected {
			join := rect{Left: r.Left + 1, Top: r.Bottom - 1, Right: r.Right - 1, Bottom: r.Bottom + 1}
			procFillRect.Call(dc, uintptr(unsafe.Pointer(&join)), themeBrush(p.Panel))
		}
		var buf [64]uint16
		item := tabControlItem{Mask: TCIF_TEXT, Text: &buf[0], TextMax: int32(len(buf))}
		procSendMessageW.Call(hwnd, TCM_GETITEMW, uintptr(i), uintptr(unsafe.Pointer(&item)))
		label := syscall.UTF16ToString(buf[:])
		if label != "" {
			chars, _ := syscall.UTF16FromString(label)
			procDrawTextW.Call(dc, uintptr(unsafe.Pointer(&chars[0])), uintptr(len(chars)-1), uintptr(unsafe.Pointer(&r)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
	}
}
