package main

// walkingAxes are physical controller positions normalized to [-1,+1], except
// pedals (0..1). Walking mode deliberately ignores racing brake zones,
// response curves, steering/thrust mixing, and extra axis assignments.
type walkingAxes struct {
	LeftX, LeftY, RightX, RightY float64
	Throttle, StickX, StickY     float64
	WheelX, Accelerator, Brake   float64
}

type walkingOutput struct {
	LX, LY, RX, RY float64
}

// walkingThrottleNeutral gives non-spring-centered throttle users a stationary
// 45%-55% physical-travel band, remapping both outside ranges to full output.
// The caller normalizes inverted physical axes to [-1,1] first.
func walkingThrottleNeutral(signed float64) float64 {
	if signed < -0.1 {
		return clampWalk((signed+0.1)/0.9, -1, 0)
	}
	if signed > 0.1 {
		return clampWalk((signed-0.1)/0.9, 0, 1)
	}
	return 0
}

func calculateWalkingOutput(mode string, in walkingAxes) walkingOutput {
	var out walkingOutput
	switch mode {
	case "HOSAS":
		out = walkingOutput{LX: in.LeftX, LY: in.LeftY, RX: in.RightX, RY: in.RightY}
	case "HOTAT":
		// Physical midpoint is always neutral; no racing Analog Brake Zone.
		out = walkingOutput{LY: (in.LeftY + in.RightY) / 2, RX: in.LeftY - in.RightY}
	case "HOTAS":
		out = walkingOutput{LY: in.Throttle, RX: in.StickX, RY: in.StickY}
	case "SINGLE":
		out = walkingOutput{LY: in.StickY, RX: in.StickX}
	case "WHEEL":
		// Released pedals both read zero and cancel to neutral.
		out = walkingOutput{LY: in.Accelerator - in.Brake, RX: in.WheelX}
	}
	out.LX = clampWalk(out.LX, -1, 1)
	out.LY = clampWalk(out.LY, -1, 1)
	out.RX = clampWalk(out.RX, -1, 1)
	out.RY = clampWalk(out.RY, -1, 1)
	return out
}

func clampWalk(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
