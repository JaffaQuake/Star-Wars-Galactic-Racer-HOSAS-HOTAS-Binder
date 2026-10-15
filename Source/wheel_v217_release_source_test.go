package main

import (
	"os"
	"strings"
	"testing"
)

// The Win32 GUI cannot run in a Linux CI host; guard the exact v2.1.7
// presentation regression here and verify the real GUI separately on Windows.
func TestV217DrivingWheelHandBrakeLabelVisibilitySource(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	checks := []string{
		`pedalDeviceLabel         uintptr`,
		`a.pedalDeviceLabel = a.createLabel("Pedal device", 20, 182, 115, 24)`,
		`a.setGroupVisible([]uintptr{a.pedalDeviceLabel}, !isWheel)`,
		`a.handbrakeShiftHandles = append(a.handbrakeShiftHandles, a.handbrakeTitle, a.pedalDeviceLabel, a.controls[idPedalDevice])`,
		`setText(a.handbrakeTitle, "Hand Brake")`,
		`setText(a.handbrakeTitle, "Rudder Pedals")`,
	}
	for _, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("missing Hand Brake/Rudder Pedals UI guard: %s", want)
		}
	}
	if strings.Contains(s, "Driving Wheel (EXPERIMENTAL)") {
		t.Fatal("experimental branding has returned to current UI")
	}
	if !strings.Contains(s, `"Driving Wheel"`) || !strings.Contains(s, "Bridge v2.1.7") {
		t.Fatal("current Driving Wheel name or release version is missing")
	}
}
