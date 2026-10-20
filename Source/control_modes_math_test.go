package main

import (
	"math"
	"testing"
)

func nearMode(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
}
func TestCenteredStickFixed50Percent(t *testing.T) {
	for _, tc := range []struct{ in, out float64 }{
		{0, -1}, {.25, -.5}, {.5, 0}, {.75, .5}, {1, 1},
	} {
		nearMode(t, "centered", leverDrive(tc.in, "Centered Stick", 5, 20), tc.out)
	}
	nearMode(t, "fixed regardless of legacy zone", leverDrive(.25, "Centered Stick", 5, 40), -.5)
}
func TestHOTATLeverBrakeOptions(t *testing.T) {
	nearMode(t, "off idle", leverDrive(0, "Off", 5, 20), 0)
	nearMode(t, "off full", leverDrive(1, "Off", 5, 20), 1)
	nearMode(t, "digital idle", leverDrive(.02, "Digital at Idle", 5, 20), -1)
	nearMode(t, "digital forward", leverDrive(.3, "Digital at Idle", 5, 20), .3)
	nearMode(t, "analog idle", leverDrive(0, "Analog Brake Zone", 5, 20), -1)
	nearMode(t, "analog midpoint of brake", leverDrive(.1, "Analog Brake Zone", 5, 20), -.5)
	nearMode(t, "analog neutral", leverDrive(.2, "Analog Brake Zone", 5, 20), 0)
	nearMode(t, "analog full forward", leverDrive(1, "Analog Brake Zone", 5, 20), 1)
}
func TestAxisTuningModeAvailabilityV218(t *testing.T) {
	for _, tc := range []struct{ mode, key string }{
		{"HOSAS", profileHOSASThrust},
		{"HOTAT", profileHOSASThrust},
		{"HOTAS", profileHOTASThrottle},
		{"SINGLE", profileHOTASX},
		{"SINGLE", profileHOTASY},
		{"WHEEL", profileWheelSteering},
	} {
		found := false
		for _, entry := range axisTuneEntriesForControlMode(tc.mode) {
			if entry.Key == tc.key {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing %s", tc.mode, tc.key)
		}
	}
	for _, entry := range axisTuneEntriesForControlMode("SINGLE") {
		if entry.Key == profileHOTASThrottle {
			t.Fatal("single stick incorrectly exposes separate throttle")
		}
	}
}
