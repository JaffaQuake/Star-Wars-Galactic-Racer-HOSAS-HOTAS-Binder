package main

import "testing"

func TestWalkingOutputsAllModes(t *testing.T) {
	cases := []struct {
		mode string
		in   walkingAxes
		want walkingOutput
	}{
		{"HOSAS", walkingAxes{LeftX: -.4, LeftY: 1, RightX: .3, RightY: -.7}, walkingOutput{LX: -.4, LY: 1, RX: .3, RY: -.7}},
		{"HOTAT", walkingAxes{LeftY: 0, RightY: 0}, walkingOutput{}},
		{"HOTAT", walkingAxes{LeftY: 1, RightY: 1}, walkingOutput{LY: 1}},
		{"HOTAT", walkingAxes{LeftY: -1, RightY: -1}, walkingOutput{LY: -1}},
		{"HOTAT", walkingAxes{LeftY: 1, RightY: -1}, walkingOutput{RX: 1}},
		{"HOTAS", walkingAxes{Throttle: -.8, StickX: .4, StickY: -.3}, walkingOutput{LY: -.8, RX: .4, RY: -.3}},
		{"SINGLE", walkingAxes{StickX: -.4, StickY: .5}, walkingOutput{LY: .5, RX: -.4}},
		{"WHEEL", walkingAxes{WheelX: .25, Accelerator: .8, Brake: .2}, walkingOutput{LY: .6, RX: .25}},
		{"WHEEL", walkingAxes{Accelerator: 0, Brake: 0}, walkingOutput{}},
	}
	for _, tt := range cases {
		got := calculateWalkingOutput(tt.mode, tt.in)
		if !closeWalk(got.LX, tt.want.LX) || !closeWalk(got.LY, tt.want.LY) || !closeWalk(got.RX, tt.want.RX) || !closeWalk(got.RY, tt.want.RY) {
			t.Errorf("%s %+v: got %+v; want %+v", tt.mode, tt.in, got, tt.want)
		}
	}
}
func closeWalk(a, b float64) bool { return a-b < 0.000001 && b-a < 0.000001 }

// Regression guard: Walking Mode uses a fixed symmetric physical throttle
// deadband, independent of each profile's racing brake zones and curve joins.
func TestWalkingThrottleNeutralBandAndScaling(t *testing.T) {
	tests := []struct{ travel, want float64 }{
		{0, -1}, {.225, -.5}, {.449, (-.102 + .1) / .9},
		{.45, 0}, {.50, 0}, {.55, 0},
		{.775, .5}, {1, 1},
	}
	for _, tt := range tests {
		got := walkingThrottleNeutral(tt.travel*2 - 1)
		if !closeWalk(got, tt.want) {
			t.Errorf("travel %g got %.9f want %.9f", tt.travel, got, tt.want)
		}
	}
	prev := -1.0
	for i := 0; i <= 1000; i++ {
		v := walkingThrottleNeutral(float64(i)/500 - 1)
		if v < prev-1e-9 || v < -1 || v > 1 {
			t.Fatalf("non-monotonic or invalid output at %d: %g", i, v)
		}
		prev = v
	}
}
func TestHOTATWalkingNeutralDifferential(t *testing.T) {
	in := walkingAxes{LeftY: walkingThrottleNeutral(.05), RightY: walkingThrottleNeutral(-.05)}
	if out := calculateWalkingOutput("HOTAT", in); out != (walkingOutput{}) {
		t.Errorf("both levers in 45-55%% should cause no walk or camera drift: %+v", out)
	}
}
