//go:build windows

package main

import "testing"

func TestUIBackgroundAndMonitorPalette(t *testing.T) {
	if mainBackgroundColor != colorRGB(212, 212, 212) {
		t.Fatalf("main background: got %#x, want #D4D4D4", mainBackgroundColor)
	}
	if monitorDotDiameter != 12 || monitorDotBorder != 2 || monitorDotNullPenStockID != 8 {
		t.Fatalf("unexpected monitor dot geometry / outline settings")
	}
	for _, tc := range []struct {
		id   int
		want uintptr
	}{
		{idLeftDot, colorRGB(216, 74, 74)},
		{idRightDot, colorRGB(52, 126, 212)},
		{idDriveSteerDot, colorRGB(40, 151, 83)},
	} {
		if got := monitorDotColor(uintptr(tc.id)); got != tc.want {
			t.Errorf("marker id %d: got %#x, want %#x", tc.id, got, tc.want)
		}
	}
}
