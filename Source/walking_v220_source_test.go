package main

import (
	"os"
	"strings"
	"testing"
)

// Source contracts for UI paths we cannot exercise from the Linux cross-builder.
func TestWalkingModeUIAndLifecycleSource(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, required := range []string{
		`Bridge v2.2.3`,
		`a.createCheck("Walking Mode: OFF"`,
		`a.createLabel("Walking Toggle Button"`,
		`idWalkingLearn && notify == BN_CLICKED`,
		`a.startLearn("WALKMODE", "Walking Mode toggle")`,
		`a.clearBinding("WALKMODE", "Walking Mode toggle")`,
		`samePhysicalBinding(b, walkingBind)`,
		`walkingButtonWasPressed = buttonPressed`,
		`a.walkingActive.Store(false)`,
		`walkOutput = calculateWalkingOutput(controlMode, in)`,
		`rep.ThumbLY = floatToStick(walkOutput.LY)`,
		`rep.ThumbRX = floatToStick(walkOutput.RX)`,
		`a.monitorOutputHeading = a.createLabel("Drive / Steer"`,
	} {
		if !strings.Contains(s, required) {
			t.Errorf("walking behavior source guard missing: %s", required)
		}
	}
	for _, removed := range []string{`a.createLabel("On-Foot Controls"`, `a.createCombo(190, 258, 190, 180, idOnFootMode)`, `strings.EqualFold(onFootMode, "Buttons -> WASD")`} {
		if strings.Contains(s, removed) {
			t.Errorf("legacy foot UI or keyboard output unexpectedly present: %s", removed)
		}
	}
}
