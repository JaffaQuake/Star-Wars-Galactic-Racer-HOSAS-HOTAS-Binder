package main

import (
	"os"
	"strings"
	"testing"
)

// The Win32 interface cannot be rendered by portable Go tests. Guard the
// exact layout coordinates and mode-based visibility until verified on Windows.
func TestV219HOTATBrakeRowAndVisibility(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	requirements := []string{
		`a.createLabel("Brake behavior", 20, 145, 115, 20)`,
		`a.createCombo(150, 140, 240, 160, idHOTATBrakeBehavior)`,
		`a.createLabel("Idle %", 405, 145, 62, 20)`,
		`a.createEdit("5", 470, 140, 52, 25, idHOTATBrakeThreshold)`,
		`a.createLabel("Zone %", 405, 145, 62, 20)`,
		`a.createEdit("20", 470, 140, 52, 25, idHOTATBrakeZone)`,
		`a.setGroupVisible(a.hotatBrakeGroup, isHOTAT)`,
		`a.setGroupVisible(a.hotatBrakeDigitalGroup, isHOTAT && hotatMode == "Digital at Idle")`,
		`a.setGroupVisible(a.hotatBrakeAnalogGroup, isHOTAT && hotatMode == "Analog Brake Zone")`,
		`a.createLabel("Additional Analogue Inputs", 20, 178, 220, 22)`,
		`a.createCombo(925, 105, 185, 160, idControlMode)`,
		`a.createCombo(925, 137, 185, 140, idInputBackend)`,
	}
	for _, want := range requirements {
		if !strings.Contains(s, want) {
			t.Errorf("missing UI regression guard: %s", want)
		}
	}
	for _, forbidden := range []string{
		`a.createLabel("Brake behavior", 820, 48, 110, 20)`,
		`a.createCombo(815, 72, 170, 160, idHOTATBrakeBehavior)`,
		`a.createLabel("Idle %", 995, 48, 70, 20)`,
		`a.createLabel("Zone %", 995, 48, 70, 20)`,
	} {
		if strings.Contains(s, forbidden) {
			t.Errorf("old overlapping HOTAT layout remains: %s", forbidden)
		}
	}
}
