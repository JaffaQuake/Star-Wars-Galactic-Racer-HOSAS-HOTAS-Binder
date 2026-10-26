//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// The Additional Inputs scroll inside their own bordered section, so all of
// them fit in the space Inputs 1 and 2 originally used. The rows are child
// windows of the section. It forwards their notifications and colour requests
// to the main window, so onCommand and the existing colour handling apply
// unchanged.

var (
	auxPanelCallback   = syscall.NewCallback(auxPanelWndProc)
	auxPanelClassReady bool
	procSetScrollInfo  = user32.NewProc("SetScrollInfo")
)

const (
	sifRange           = 0x1
	sifPage            = 0x2
	sifPos             = 0x4
	sifDisableNoScroll = 0x8
	wsClipChildren     = 0x02000000
	auxWMEraseBkgnd    = 0x0014
	auxWMDrawItem      = 0x002B
	auxWMCtlColorFirst = 0x0132 // WM_CTLCOLORMSGBOX; WM_CTLCOLORSTATIC is the last
)

type scrollInfo struct {
	CbSize, FMask   uint32
	NMin, NMax      int32
	NPage           uint32
	NPos, NTrackPos int32
}

type auxInputPanel struct {
	hwnd             uintptr
	originX, originY int              // main-window position of the section's client area
	rects            map[uintptr]rect // row controls in section coordinates, unscrolled
	scrollY          int
	contentHeight    int
	viewHeight       int
	creating         bool // createControl puts new controls in the section
}

// createAuxInputPanel makes the section. Until finishAuxInputPanel, controls
// created with main-window coordinates are placed inside it instead.
func (a *appState) createAuxInputPanel(x, y, w, h int) {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	className := utf16ptr("GalacticRacerAdditionalInputs")
	if !auxPanelClassReady {
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
		wc := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			LpfnWndProc:   auxPanelCallback,
			HInstance:     hInst,
			HCursor:       cursor,
			HbrBackground: 0, // painted with the main window's label background
			LpszClassName: className,
		}
		if ok, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); ok == 0 {
			return
		}
		auxPanelClassReady = true
	}
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16ptr(""))),
		WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|wsClipChildren,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), a.hwnd, 0, hInst, 0)
	if hwnd == 0 {
		return
	}
	a.baseRects[hwnd] = rect{Left: int32(x), Top: int32(y), Right: int32(x + w), Bottom: int32(y + h)}
	a.auxPanel = &auxInputPanel{hwnd: hwnd, originX: x + 1, originY: y + 1, rects: make(map[uintptr]rect), creating: true}
}

// finishAuxInputPanel stops placing controls in the section and sets up its
// scrollbar for content rows of total height content.
func (a *appState) finishAuxInputPanel(content int) {
	p := a.auxPanel
	if p == nil {
		return
	}
	p.creating = false
	p.contentHeight = content
	var rc rect
	procGetClientRect.Call(p.hwnd, uintptr(unsafe.Pointer(&rc)))
	p.viewHeight = int(rc.Bottom - rc.Top)
	si := scrollInfo{FMask: sifRange | sifPage | sifPos | sifDisableNoScroll, NMax: int32(content - 1), NPage: uint32(p.viewHeight)}
	si.CbSize = uint32(unsafe.Sizeof(si))
	procSetScrollInfo.Call(p.hwnd, SB_VERT, uintptr(unsafe.Pointer(&si)), 1)
}

func (a *appState) scrollAuxPanel(pos int) {
	p := a.auxPanel
	if p == nil {
		return
	}
	pos = clampScroll(pos, p.contentHeight, p.viewHeight)
	if pos == p.scrollY {
		return
	}
	a.closeCombo(false)
	p.scrollY = pos
	hdwp, _, _ := procBeginDeferWindowPos.Call(uintptr(len(p.rects)))
	for h, r := range p.rects {
		y := int(r.Top) - pos
		if hdwp != 0 {
			if next, _, _ := procDeferWindowPos.Call(hdwp, h, 0, uintptr(r.Left), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE); next != 0 {
				hdwp = next
			}
		} else {
			procSetWindowPos.Call(h, 0, uintptr(r.Left), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
		}
	}
	if hdwp != 0 {
		procEndDeferWindowPos.Call(hdwp)
	}
	si := scrollInfo{FMask: sifPos, NPos: int32(pos)}
	si.CbSize = uint32(unsafe.Sizeof(si))
	procSetScrollInfo.Call(p.hwnd, SB_VERT, uintptr(unsafe.Pointer(&si)), 1)
	procRedrawWindow.Call(p.hwnd, 0, 0, RDW_INVALIDATE|RDW_ERASE|RDW_ALLCHILDREN)
}

// pointInWindow reports whether screen point (x, y) is over window h.
func pointInWindow(h uintptr, x, y int32) bool {
	var rc rect
	if r, _, _ := procGetWindowRect.Call(h, uintptr(unsafe.Pointer(&rc))); r == 0 {
		return false
	}
	return x >= rc.Left && x < rc.Right && y >= rc.Top && y < rc.Bottom
}

// routeMouseWheel scrolls the Additional Inputs section while the pointer is
// over it, and the whole window once the section reaches its top or bottom.
func (a *appState) routeMouseWheel(wParam, lParam uintptr) {
	if p := a.auxPanel; p != nil && pointInWindow(p.hwnd, int32(int16(loword(lParam))), int32(int16(hiword(lParam)))) {
		if next, moved := auxWheelScroll(int(int16(hiword(wParam))), p.scrollY, p.contentHeight, p.viewHeight, auxiliaryRowPitch); moved {
			a.scrollAuxPanel(next)
			return
		}
	}
	a.handleMouseWheel(wParam)
}

// hiddenInAuxPanel reports whether control h is an Additional Inputs row
// control scrolled out of view, so hover help ignores it.
func (a *appState) hiddenInAuxPanel(h uintptr, pt point) bool {
	p := a.auxPanel
	if p == nil {
		return false
	}
	if _, inPanel := p.rects[h]; !inPanel {
		return false
	}
	return !pointInWindow(p.hwnd, pt.X, pt.Y)
}

func auxPanelWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch {
	case msg == WM_COMMAND || msg == WM_NOTIFY || msg == auxWMDrawItem || (msg >= auxWMCtlColorFirst && msg <= WM_CTLCOLORSTATIC):
		r, _, _ := procSendMessageW.Call(app.hwnd, uintptr(msg), wParam, lParam)
		return r
	case msg == auxWMEraseBkgnd:
		// Same background as the main window's labels, in any theme.
		if brush, _, _ := procSendMessageW.Call(app.hwnd, WM_CTLCOLORSTATIC, wParam, hwnd); brush != 0 {
			var rc rect
			procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
			procFillRect.Call(wParam, uintptr(unsafe.Pointer(&rc)), brush)
			return 1
		}
	case msg == WM_VSCROLL:
		if p := app.auxPanel; p != nil {
			app.scrollAuxPanel(auxScrollAfter(int(loword(wParam)), int(hiword(wParam)), p.scrollY, p.contentHeight, p.viewHeight, auxiliaryRowPitch))
		}
		return 0
	case msg == WM_MOUSEWHEEL:
		app.closeCombo(false)
		app.routeMouseWheel(wParam, lParam)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}
