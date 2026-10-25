package main

import (
	"math"
	"testing"
)

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-8 {
		t.Fatalf("got %.10f want %.10f", got, want)
	}
}
func TestTuningDefaults(t *testing.T) {
	p := defaultAxisTune(0)
	for _, x := range []float64{-1, -0.7, -0.25, 0, 0.25, 0.7, 1} {
		near(t, tunedCentered(x, p), x)
	}
	for _, x := range []float64{0, 0.25, 0.7, 1} {
		near(t, tunedOneWay(x, p), x)
	}
}
func TestSaturationAndCurves(t *testing.T) {
	p := axisTuneProfile{Deadzone: 0, Curve: 2, FullOutput: 80}
	near(t, tunedCentered(0.4, p), 0.25)
	near(t, tunedCentered(0.8, p), 1)
	near(t, tunedCentered(1, p), 1)
	near(t, tunedCentered(-0.8, p), -1)
	near(t, tunedOneWay(0.8, p), 1)
	p.Curve = 0.5
	near(t, tunedOneWay(0.2, p), 0.5)
}
func TestDeadzonePreservesMax(t *testing.T) {
	p := axisTuneProfile{Deadzone: 10, Curve: 1, FullOutput: 100}
	near(t, tunedCentered(0.05, p), 0)
	near(t, tunedCentered(0.55, p), 0.5)
	near(t, tunedCentered(-1, p), -1)
	near(t, tunedOneWay(0.1, p), 0)
	near(t, tunedOneWay(1, p), 1)
}
func TestCurveValidation(t *testing.T) {
	p := checkedAxisTune(axisTuneProfile{Deadzone: -1, Curve: math.NaN(), FullOutput: 0})
	if p.Deadzone != 6 || p.Curve != 1 || p.FullOutput != 100 {
		t.Fatalf("invalid fallback %+v", p)
	}
}

func TestAxisTuningEntriesByMode(t *testing.T) {
	cases := []struct {
		name  string
		hotas bool
		want  []string
	}{
		{"Podracer Style", false, []string{profileHOSASLeft, profileHOSASRight, profileHOSASThrust, profileHOSASSteer, profilePedal, profileTwistLeft, profileTwistRight, profileTwist3, profileTwist4, profileTwist5, profileTwist6}},
		{"Traditional HOTAS", true, []string{profileHOTASX, profileHOTASY, profileHOTASThrottle, profilePedal, profileTwistLeft, profileTwistRight, profileTwist3, profileTwist4, profileTwist5, profileTwist6}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := axisTuneEntriesForMode(test.hotas)
			if len(got) != len(test.want) {
				t.Fatalf("profiles got %d expected %d", len(got), len(test.want))
			}
			for i, e := range got {
				if e.Key != test.want[i] {
					t.Fatalf("at %d got %s expected %s", i, e.Key, test.want[i])
				}
				if test.hotas && (e.Key == profileHOSASLeft || e.Key == profileHOSASRight || e.Key == profileHOSASThrust || e.Key == profileHOSASSteer) {
					t.Fatal("Podracer profile in HOTAS")
				}
				if !test.hotas && (e.Key == profileHOTASX || e.Key == profileHOTASY || e.Key == profileHOTASThrottle) {
					t.Fatal("HOTAS profile in Podracer")
				}
			}
		})
	}
}

func TestIndividualInputsAreDeadzoneOnly(t *testing.T) {
	entries := axisTuneEntriesForMode(false)
	if !entries[0].DeadzoneOnly || !entries[1].DeadzoneOnly {
		t.Fatal("Podracer inputs must not have individual curves")
	}
	for _, entry := range entries[2:] {
		if entry.DeadzoneOnly {
			t.Fatalf("unexpected deadzone-only %s", entry.Key)
		}
	}
}

func TestCameraButtonsVisibility(t *testing.T) {
	tests := []struct {
		name  string
		modes []string
		want  bool
	}{
		{"all off", []string{"Off", "Off", "Off"}, false},
		{"pedals", []string{"Move Camera", "Off", "Off"}, true},
		{"left twist", []string{"Fine steering", "Move Camera", "Pitch"}, true},
		{"right twist", []string{"Pitch", "Keyboard keys", "Move Camera"}, true},
		{"case insensitive", []string{"OFF", "MOVE CAMERA", "Off"}, true},
		{"other modes", []string{"Pitch", "Keyboard keys", "Fine steering"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cameraTestButtonsVisible(tc.modes...); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// Regression: initiating/deactivating virtual Xbox should reveal/hide the
// diagnostic actions without changing users' auxiliary camera assignments.
func TestCameraTestRequiresActiveXbox(t *testing.T) {
	cases := []struct {
		active bool
		modes  []string
		want   bool
	}{
		{false, []string{"Move Camera", "Off", "Off"}, false},
		{true, []string{"Move Camera", "Off", "Off"}, true},
		{true, []string{"Off", "Off", "Off"}, false},
		{true, []string{"Fine steering", "Move Camera", "Off"}, true},
		{false, []string{"Off", "Off", "Move Camera"}, false},
	}
	for _, c := range cases {
		if got := cameraTestControlsVisible(c.active, c.modes...); got != c.want {
			t.Fatalf("active=%v modes=%v -> %v, want %v", c.active, c.modes, got, c.want)
		}
	}
}
