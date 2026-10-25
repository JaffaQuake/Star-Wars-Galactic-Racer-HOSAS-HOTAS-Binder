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

func TestAuxiliaryDriveContribution(t *testing.T) {
	cases := []struct {
		mode        string
		value, want float64
	}{
		{"Drive", 0.4, 0.4},
		{"Drive", -0.6, -0.6},
		{"Drive", 1.5, 1},
		{"Forward", 0.7, 0.7},
		{"Forward", -0.7, 0.7}, // Output Negative lever still drives forward
		{"Reverse", 0.7, -0.7},
		{"Reverse", -0.7, -0.7},
		{"Fine steering", 0.9, 0},
		{"Off", 0.9, 0},
	}
	for _, c := range cases {
		if got := auxiliaryDriveContribution(c.mode, c.value); got != c.want {
			t.Errorf("%s %.2f = %.2f, want %.2f", c.mode, c.value, got, c.want)
		}
	}
}

func TestAuxiliaryModeForAxisType(t *testing.T) {
	cases := []struct{ mode, axisType, want string }{
		{"Drive", "Lever", "Forward"},
		{"Forward", "Centered", "Drive"},
		{"Reverse", "Centered", "Drive"},
		{"Forward", "Lever", "Forward"},
		{"Drive", "Centered", "Drive"},
		{"Pitch", "Lever", "Pitch"},
		{"Boost/Boost", "Lever", "Boost"},
		{"Drift/Drift", "Lever", "Drift"},
		{"Boost", "Centered", "Boost/Boost"},
		{"Drift", "Centered", "Drift/Drift"},
		{"Boost/Drift", "Lever", "Boost/Drift"}, // not offered on a Lever, so the UI falls back to Off
	}
	for _, c := range cases {
		if got := auxiliaryModeForAxisType(c.mode, c.axisType); got != c.want {
			t.Errorf("%s -> %s = %s, want %s", c.mode, c.axisType, got, c.want)
		}
	}
}

func TestHOSASAvoidDrive(t *testing.T) {
	cases := []struct {
		name                  string
		left, right, combined float64
		leftAvoid, rightAvoid bool
		want                  float64
	}{
		{"neither keeps the normal drive", 0.4, 1, 1, false, false, 1},
		{"left avoided: right drives alone", 0.4, 1, 1, true, false, 1},
		{"left avoided: right reverses fully", 1, -1, 0, true, false, -1},
		{"right avoided: left drives alone", 0.6, -0.8, 0.12, false, true, 0.6},
		{"both avoided: no stick drive", 1, 1, 1, true, true, 0},
	}
	for _, c := range cases {
		if got := hosasDrive(c.left, c.right, c.combined, c.leftAvoid, c.rightAvoid); got != c.want {
			t.Errorf("%s: got %.2f, want %.2f", c.name, got, c.want)
		}
	}
}

func TestRacingPitch(t *testing.T) {
	cases := []struct {
		name          string
		isHOTAS       bool
		hotasY        float64
		pitchAssigned bool
		pitchValue    float64
		up, down      bool
		want          float64
	}{
		{"Podracer with no pitch source", false, 0.8, false, 0, false, false, 0},
		{"HOTAS stick Y", true, 0.8, false, 0, false, false, 0.8},
		{"Pitch input replaces HOTAS stick Y", true, 0.8, true, -0.3, false, false, -0.3},
		{"Pitch Up button", false, 0, true, -0.3, true, false, 1},
		{"Pitch Down button", true, 0.8, false, 0, false, true, -1},
		{"both buttons cancel", true, 0.8, true, 0.5, true, true, 0},
	}
	for _, c := range cases {
		if got := racingPitch(c.isHOTAS, c.hotasY, c.pitchAssigned, c.pitchValue, c.up, c.down); got != c.want {
			t.Errorf("%s: got %.2f, want %.2f", c.name, got, c.want)
		}
	}
}
