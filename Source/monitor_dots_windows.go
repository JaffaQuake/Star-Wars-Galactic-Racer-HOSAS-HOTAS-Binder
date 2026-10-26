//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// A window region clips away the corners, leaving the graph/grid visible
// behind every moving indicator. No new buttons, inputs, or capture behavior.
// Two nested filled ellipses produce a *true*, symmetrical black ring.
// A clipped single-pixel GDI pen can appear as a one-sided shadow at 12px.
const monitorDotDiameter = 12
const monitorDotBorder = 2
const monitorDotNullPenStockID = 8 // NULL_PEN (Win32 GetStockObject)

var (
	monitorDotWndProcCallback = syscall.NewCallback(monitorDotWndProc)
	monitorDotClassRegistered bool
	procCreateEllipticRgn     = gdi32.NewProc("CreateEllipticRgn")
	procSetWindowRgn          = user32.NewProc("SetWindowRgn")
	procGetDlgCtrlID          = user32.NewProc("GetDlgCtrlID")
)

func (a *appState) createLiveMonitorDot(x, y, id int) uintptr {
	if !monitorDotClassRegistered {
		hInstance, _, _ := procGetModuleHandleW.Call(0)
		className := utf16ptr("GalacticRacerMonitorDot")
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
		wc := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			LpfnWndProc:   monitorDotWndProcCallback,
			HInstance:     hInstance,
			HCursor:       cursor,
			HbrBackground: 0, // drawn by the child itself
			LpszClassName: className,
		}
		if registered, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); registered == 0 {
			return 0
		}
		monitorDotClassRegistered = true
	}
	h := a.createControl("GalacticRacerMonitorDot", "", WS_CHILD|WS_VISIBLE,
		x, y, monitorDotDiameter, monitorDotDiameter, id)
	if h != 0 {
		region, _, _ := procCreateEllipticRgn.Call(0, 0, monitorDotDiameter, monitorDotDiameter)
		if region != 0 {
			if ok, _, _ := procSetWindowRgn.Call(h, region, 1); ok == 0 {
				procDeleteObject.Call(region) // successful SetWindowRgn transfers ownership
			}
		}
	}
	return h
}

func monitorDotColor(id uintptr) uintptr {
	switch int(id) {
	case idLeftDot:
		return colorRGB(216, 74, 74) // red #D84A4A
	case idRightDot:
		return colorRGB(52, 126, 212) // blue #347ED4
	case idDriveSteerDot:
		return colorRGB(40, 151, 83) // green #289753
	default:
		return colorRGB(90, 90, 90)
	}
}

func monitorDotWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case 0x0014: // WM_ERASEBKGND: circle fully painted by WM_PAINT
		return 1
	case WM_PAINT:
		var ps tuningPaintStruct
		dc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if dc == 0 {
			return 0
		}
		var bounds rect
		procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds)))
		id, _, _ := procGetDlgCtrlID.Call(hwnd)
		// Do not stroke the window-region boundary: GDI clips that stroke
		// unevenly on some Windows renderers, leaving a side-only shadow.
		// Fill a black outer ellipse and then an inset colored ellipse;
		// the 2px-wide ring is fully inside the round indicator window.
		outer, _, _ := procCreateSolidBrush.Call(currentPalette.MonitorDotBorder)
		inner, _, _ := procCreateSolidBrush.Call(monitorDotColor(id))
		nullPen, _, _ := procGetStockObject.Call(monitorDotNullPenStockID)
		if outer != 0 && inner != 0 && nullPen != 0 {
			priorBrush, _, _ := procSelectObject.Call(dc, outer)
			priorPen, _, _ := procSelectObject.Call(dc, nullPen)
			procEllipse.Call(dc, 0, 0, uintptr(bounds.Right), uintptr(bounds.Bottom))
			procSelectObject.Call(dc, inner)
			procEllipse.Call(dc,
				monitorDotBorder, monitorDotBorder,
				uintptr(bounds.Right-monitorDotBorder),
				uintptr(bounds.Bottom-monitorDotBorder))
			procSelectObject.Call(dc, priorPen)
			procSelectObject.Call(dc, priorBrush)
		}
		if outer != 0 {
			procDeleteObject.Call(outer)
		}
		if inner != 0 {
			procDeleteObject.Call(inner)
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}
