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
