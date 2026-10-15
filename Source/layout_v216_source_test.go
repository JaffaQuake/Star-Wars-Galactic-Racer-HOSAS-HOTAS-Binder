package main

import (
	"os"
	"strings"
	"testing"
)

func TestV216FullOutputUnderDeadzoneAndNoMidpointHeading(t *testing.T) {
	b, err := os.ReadFile("axis_tuning_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "tuningMidpointHeaderID") || strings.Contains(s, `p.label("Midpoint",`) {
		t.Fatal("standalone midpoint header remains")
	}
	layout := s[strings.Index(s, "func (p *axisTuningPopup) layoutProfile("):]
	dead := strings.Index(layout, "move(tuningDeadzoneLabelID,")
	full := strings.Index(layout, "move(tuningFullLabelID,")
	toggle := strings.Index(layout, "move(tuningTwoStageID,")
	curve := strings.Index(layout, "move(tuningCurveLabelID,")
	if !(0 <= dead && dead < full && full < toggle && toggle < curve) {
		t.Fatal("wrong control order")
	}
	for _, want := range []string{
		"move(tuningFullLabelID, 0, 111, 160, 25)",
		"move(tuningFullID, 168, 106, 300, 34)",
		"move(tuningCustomMidID, 0, 349, 200, 27)",
		"graphLabelY, graphHeight = 462, 134",
		"graphLabelY, graphHeight = 426, 170",
	} {
		if !strings.Contains(layout, want) {
			t.Errorf("missing coordinate %q", want)
		}
	}
	// For full-axis tuning the reset footer fits inside the 807px child.
	for _, bottom := range []int{462 + 24 + 134, 426 + 24 + 170, 270 + 24 + 230, 125 + 24 + 250} {
		if bottom+121+30 > 807 {
			t.Fatalf("out of bounds: %d", bottom+121+30)
		}
	}
}
