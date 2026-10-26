//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	helpPanelCallback   = syscall.NewCallback(helpPanelWndProc)
	helpPanelClassReady bool
	procDrawTextW       = user32.NewProc("DrawTextW")
	procFrameRect       = user32.NewProc("FrameRect")
	procSetTextColor    = gdi32.NewProc("SetTextColor")
	procSetBkMode       = gdi32.NewProc("SetBkMode")
)

const (
	DT_LEFT      = 0x0000
	DT_WORDBREAK = 0x0010
	DT_NOPREFIX  = 0x0800
	TRANSPARENT  = 1
)

// The owner is the main scrollable window, not the calibration tab. Paint a
// modest shadow *inside* the child's bounds: it cannot overlap neighboring UI
// and all its padding is explicit and DPI-scalable with the existing UI.
func (a *appState) createHelpPanel(x, y, w, h int) {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	name := utf16ptr("GalacticRacerHelpPanel")
	if !helpPanelClassReady {
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
		wc := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			LpfnWndProc:   helpPanelCallback,
			HInstance:     hInst,
			HCursor:       cursor,
			HbrBackground: 0, // WM_PAINT fills the whole surface
			LpszClassName: name,
		}
		if success, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); success == 0 {
			return
		}
		helpPanelClassReady = true
	}
	a.createControl("GalacticRacerHelpPanel", "Help: hover over a control to see what it does.",
		WS_CHILD|WS_VISIBLE, x, y, w, h, idHelpText)
}

func helpPanelWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if message == WM_SETTEXT {
		result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		procInvalidateRect.Call(hwnd, 0, 0)
		return result
	}
	if message == WM_PAINT {
		var ps tuningPaintStruct
		dc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if dc == 0 {
			return 0
		}
		var bounds rect
		procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds)))
		right, bottom := bounds.Right, bounds.Bottom
		// Panel's own background matches the neutral main window.
		paint := func(rc rect, c uintptr) {
			brush, _, _ := procCreateSolidBrush.Call(c)
			if brush != 0 {
				procFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brush)
				procDeleteObject.Call(brush)
			}
		}
		pal := currentPalette
		paint(bounds, pal.Window)
		if right > 10 && bottom > 10 {
			paint(rect{Left: 4, Top: 4, Right: right, Bottom: bottom}, pal.HelpShadowOuter)
			paint(rect{Left: 2, Top: 2, Right: right - 2, Bottom: bottom - 2}, pal.HelpShadowInner)
			card := rect{Left: 0, Top: 0, Right: right - 5, Bottom: bottom - 5}
			paint(card, pal.HelpCard)
			border, _, _ := procCreateSolidBrush.Call(pal.HelpBorder)
			if border != 0 {
				procFrameRect.Call(dc, uintptr(unsafe.Pointer(&card)), border)
				procDeleteObject.Call(border)
			}
			var chars [1024]uint16
			n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&chars[0])), uintptr(len(chars)))
			if n > 0 {
				if app != nil && app.font != 0 {
					old, _, _ := procSelectObject.Call(dc, app.font)
					defer procSelectObject.Call(dc, old)
				}
				procSetBkMode.Call(dc, TRANSPARENT)
				procSetTextColor.Call(dc, pal.HelpText)
				textBounds := rect{Left: 12, Top: 10, Right: right - 19, Bottom: bottom - 14}
				procDrawTextW.Call(dc, uintptr(unsafe.Pointer(&chars[0])), n,
					uintptr(unsafe.Pointer(&textBounds)), DT_LEFT|DT_WORDBREAK|DT_NOPREFIX)
			}
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	}
	returnVal, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return returnVal
}
