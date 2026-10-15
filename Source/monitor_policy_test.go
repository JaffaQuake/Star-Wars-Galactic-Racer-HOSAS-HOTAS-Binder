package main

import "testing"

func TestMonitorPolicy(t *testing.T) {
	cases := []struct{ active, optIn, visible bool }{
		{false, false, true}, {false, true, true}, {true, false, false}, {true, true, true},
	}
	for _, tc := range cases {
		if got := shouldRenderMonitor(tc.active, tc.optIn); got != tc.visible {
			t.Errorf("active=%v optIn=%v: got %v want %v", tc.active, tc.optIn, got, tc.visible)
		}
	}
}
