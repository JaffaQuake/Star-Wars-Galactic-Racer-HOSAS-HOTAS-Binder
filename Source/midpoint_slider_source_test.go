package main

import (
	"os"
	"strings"
	"testing"
)

// Portable regression guards for Win32-only layout code. They inspect the
// source because native windows cannot be executed on the build host.
func TestMidpointSliderReplacesNumericEditAndHidesInCustomMode(t *testing.T) {
	data, err := os.ReadFile("axis_tuning_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, want := range []string{
		`p.slider(168, 366, 300, 34, tuningMidpointSliderID, 10, 90)`,
		`show(id, stage && !custom)`,
		`show(id, custom)`,
		`show(id, stage)`,
		`old.Midpoint = p.sliderValue(tuningMidpointSliderID)`,
		`p.setSlider(tuningMidpointSliderID, tune.Midpoint)`,
		`setText(p.controls[tuningMidpointPercentID], fmt.Sprintf("%d%%", profile.Midpoint))`,
		`move(tuningMidpointSliderID, 168, 383, 300, 34)`,
		`case WM_HSCROLL:`,
		`p.applyDraft()`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("missing slider requirement %q", want)
		}
	}
	for _, forbidden := range []string{`tuningMidpointEditID`, `case tuningMidpointEditID`, `p.control("EDIT", "50"`, `tuningPresetID`, `tuningPresetLabelID`, `"Midpoint Presets"`} {
		if strings.Contains(src, forbidden) {
			t.Errorf("old midpoint edit survives: %q", forbidden)
		}
	}
	if !strings.Contains(src, `tuningMidpointLabelID, tuningMidpointSliderID, tuningMidpointPercentID`) {
		t.Error("ordinary midpoint slider and its readout must share the custom-mode visibility condition")
	}
}

func TestDiagonalMidpointRangeAndCustomPositionsRemainValid(t *testing.T) {
	p := defaultAxisTune(0)
	p.TwoStage = true
	for _, midpoint := range []int{10, 11, 35, 50, 65, 89, 90} {
		p.Midpoint = midpoint
		p.CustomMidpoint = false
		p = checkedAxisTune(p)
		if p.Midpoint != midpoint {
			t.Fatalf("lost diagonal midpoint %d: %+v", midpoint, p)
		}
		x := float64(midpoint) / 100
		if got := twoStageArc(x, p); got < x-0.00001 || got > x+0.00001 {
			t.Fatalf("midpoint %d should be on X=Y diagonal, got %v", midpoint, got)
		}
	}
	p.CustomMidpoint = true
	p.MidpointX, p.MidpointY = 35, 65
	p = checkedAxisTune(p)
	if p.MidpointX != 35 || p.MidpointY != 65 {
		t.Fatalf("custom midpoint altered: %+v", p)
	}
	if got := twoStageArc(.35, p); got < .64999 || got > .65001 {
		t.Fatalf("custom X/Y join incorrect: %v", got)
	}
}

// Windows GUI cannot be executed in this Linux build environment. Guard the
// measured control coordinates and custom/normal mode reflow at source level.
func TestTwoStageSectionsOrderAndMidpointReflow(t *testing.T) {
	b, err := os.ReadFile("axis_tuning_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	layoutStart := strings.Index(src, "func (p *axisTuningPopup) layoutProfile(")
	layoutEnd := strings.Index(src, "func (p *axisTuningPopup) refreshEntriesFromConfig(")
	if layoutStart < 0 || layoutEnd <= layoutStart {
		t.Fatal("cannot locate Win32 layout")
	}
	layout := src[layoutStart:layoutEnd]
	stage1 := strings.Index(layout, "move(tuningCurveLabelID,")
	stage2 := strings.Index(layout, "move(tuningStage2LabelID,")
	midpoint := strings.Index(layout, "move(tuningCustomMidID,")
	full := strings.Index(layout, "move(tuningFullLabelID,")
	ordinary := strings.Index(layout, "move(tuningMidpointSliderID,")
	if !(stage1 >= 0 && stage1 < stage2 && stage2 < midpoint && midpoint < ordinary && full >= 0 && full < stage1) {
		t.Fatalf("expected Full Output -> curve 1 -> curve 2 -> checkbox -> slider order; positions: %d %d %d %d %d", full, stage1, stage2, midpoint, ordinary)
	}
	for _, want := range []string{
		`move(tuningMidXID, 168, 383, 300, 34)`,
		`move(tuningMidYID, 168, 419, 300, 34)`,
		`move(tuningMidpointSliderID, 168, 383, 300, 34)`,
		`graphLabelY, graphHeight = 462, 134`,
		`graphLabelY, graphHeight = 426, 170`,
		`move(tuningFullID, 168, 106, 300, 34)`,
		`show(id, stage && !custom)`, `show(id, custom)`,
	} {
		if !strings.Contains(layout, want) {
			t.Errorf("missing two-stage geometry or visibility %q", want)
		}
	}
	// Graph top = graphLabelY + 24. Both modes finish at the same
	// vertical coordinate; Reset remains inside the 807px tuning child.
	standardBottom, customBottom := 426+24+170, 462+24+134
	if standardBottom != customBottom || standardBottom+121+30 > 807 {
		t.Fatalf("two-stage graph/reset outside page or uneven footer: default=%d custom=%d", standardBottom, customBottom)
	}
	if strings.Contains(src, "tuningPreset") || strings.Contains(src, "Midpoint Presets") || strings.Contains(src, "tuningMidpointHeaderID") || strings.Contains(src, `p.label("Midpoint",`) {
		t.Fatal("removed midpoint presets must not appear in Windows control code")
	}
}
