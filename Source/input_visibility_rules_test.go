package main

import "testing"

func TestInputDeviceSelected(t *testing.T) {
	for _, tc := range []struct {
		name, backend string
		id            int
		guid          string
		want          bool
	}{
		{"winmm_none", "WinMM", -1, "", false},
		{"winmm_first_device", "WinMM", 0, "", true},
		{"winmm_second_device", "WinMM", 1, "", true},
		{"di_none", "DirectInput", -1, "", false},
		{"di_device", "DirectInput", -1, "{device-guid}", true},
		{"di_blank_guid", "DirectInput", 0, " ", false},
		{"auto_none", "Auto", -1, "", false},
		{"auto_device", "Auto", -1, "{device-guid}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := inputDeviceSelected(tc.backend, tc.id, tc.guid)
			if got != tc.want {
				t.Fatalf("selected=%v, want %v", got, tc.want)
			}
		})
	}
}
