package main

import "testing"

func TestWheelButtonActions(t *testing.T) {
	for _, tc := range []struct {
		mode string
		v    float64
		want string
	}{
		{"Boost", 0, ""}, {"Boost", .001, "Boost"}, {"Boost", -.05, "Boost"},
		{"Drift", .001, "Drift"}, {"Drift", -.4, "Drift"},
		{"Boost/Drift", -.25, "Drift"}, {"Boost/Drift", .25, "Boost"}, {"Boost/Drift", 0, ""},
		{"Boost/Boost", -.25, "Boost"}, {"Boost/Boost", .25, "Boost"}, {"Boost/Boost", 0, ""},
		{"Drift/Drift", -.25, "Drift"}, {"Drift/Drift", .25, "Drift"}, {"Drift/Drift", 0, ""},
		{"Off", 1, ""},
	} {
		if got := digitalAxisAction(tc.mode, tc.v); got != tc.want {
			t.Errorf("%s / %v -> %s, expected %s", tc.mode, tc.v, got, tc.want)
		}
	}
}

func TestWheelTriggers(t *testing.T) {
	for _, tc := range []struct {
		input float64
		want  byte
	}{
		{-1, 0}, {0, 0}, {.5, 128}, {1, 255}, {2, 255},
	} {
		if got := pedalTriggerByte(tc.input); got != tc.want {
			t.Errorf("%v trigger=%v wanted %v", tc.input, got, tc.want)
		}
	}
}

func TestThresholdedButtonValue(t *testing.T) {
	for _, tc := range []struct {
		normalized, tuned float64
		threshold         int
		want              float64
	}{
		{0.19, 0.14, 20, 0},
		{0.20, 0.15, 20, 0.15},
		{-0.50, -0.47, 20, -0.47},
		{-0.10, -0.04, 20, 0},
		{0.30, 0, 20, 0}, // still inside the deadzone
		{0.06, 0.01, 5, 0.01},
	} {
		if got := thresholdedButtonValue(tc.normalized, tc.tuned, tc.threshold); got != tc.want {
			t.Errorf("%.2f @ %d%% -> %.2f, expected %.2f", tc.normalized, tc.threshold, got, tc.want)
		}
	}
	for _, mode := range []string{"Boost", "Drift", "Boost/Drift", "Boost/Boost", "Drift/Drift"} {
		if !isAuxiliaryButtonMode(mode) {
			t.Errorf("%s should show Threshold %%", mode)
		}
	}
	for _, mode := range []string{"Off", "Keyboard keys", "Drive", "Pitch"} {
		if isAuxiliaryButtonMode(mode) {
			t.Errorf("%s should not show the button Threshold %%", mode)
		}
	}
}
