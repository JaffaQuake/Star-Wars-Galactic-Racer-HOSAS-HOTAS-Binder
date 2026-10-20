package main

import (
	"os"
	"strings"
	"testing"
)

func TestV218ModeWiringSource(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`"HOSAS", "HOTAT", "HOTAS", "SINGLE", "WHEEL"`,
		`"Podracer Style (HOSAS)"`, `"Podracer Style (HOTAT)"`, `"Single Stick"`,
		`idHOTATBrakeBehavior`, `idRightThrottleBar`, `RawWheelBrake`,
		`hotat_brake_behavior`, `hotat_brake_zone`, `hotas_brake_behavior`,
		`--check-vigem`, `a.configureInputMonitorBars(leftBar, rightBar, isWheel)`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing v2.1.8 requirement: %s", want)
		}
	}
	if strings.Contains(s, `modes := []string{"HOSAS", "HOTAS", "WHEEL"}`) {
		t.Fatal("stale 3-mode UI switch handler")
	}
}
