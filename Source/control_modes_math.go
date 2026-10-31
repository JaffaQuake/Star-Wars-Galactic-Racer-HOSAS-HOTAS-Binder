package main

// clamp with no Windows API dependencies so lever-brake mapping can be tested
// on Linux/CI before the real Windows hardware test.
func limitFraction(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Physical lever travel is 0..1. This is the existing HOTAS brake law,
// shared by HOTAT without modifying either legacy HOSAS mixing path.
func leverDrive(travel float64, behavior string, threshold, zone int) float64 {
	travel = limitFraction(travel, 0, 1)
	switch behavior {
	case "Off":
		return travel
	case "Analog Brake Zone", "Centered Stick":
		fraction := limitFraction(float64(zone)/100, 0.01, 0.50)
		if behavior == "Centered Stick" {
			fraction = 0.50
		}
		if travel <= fraction {
			return limitFraction(travel/fraction-1, -1, 0)
		}
		return limitFraction((travel-fraction)/(1-fraction), 0, 1)
	default:
		if travel <= limitFraction(float64(threshold)/100, 0, 0.50) {
			return -1
		}
		return travel
	}
}

// Additional Inputs in Drive (Centered), Forward or Reverse (Lever) mode add
// to the control mode's own drive value: positive accelerates (RT), negative
// brakes/reverses (LT). Lever modes use travel magnitude, so the mode name,
// not Output Negative, sets the direction.
func auxiliaryDriveContribution(mode string, value float64) float64 {
	if value < 0 && mode != "Drive" {
		value = -value
	}
	switch mode {
	case "Drive":
		return limitFraction(value, -1, 1)
	case "Forward":
		return limitFraction(value, 0, 1)
	case "Reverse":
		return -limitFraction(value, 0, 1)
	}
	return 0
}

// Changing Axis Type keeps a mode in the form the new type offers: Drive
// becomes Forward on a Lever and Forward/Reverse become Drive; Boost/Boost
// and Drift/Drift become a Lever's Boost and Drift, and back again.
func auxiliaryModeForAxisType(mode, axisType string) string {
	if axisType == "Lever" {
		switch mode {
		case "Drive":
			return "Forward"
		case "Boost/Boost":
			return "Boost"
		case "Drift/Drift":
			return "Drift"
		}
		return mode
	}
	switch mode {
	case "Forward", "Reverse":
		return "Drive"
	case "Boost":
		return "Boost/Boost"
	case "Drift":
		return "Drift/Drift"
	}
	return mode
}

// Podracer (HOSAS) Avoid Drive keeps a stick in steering (Left - Right) only.
// combined is the normal two-stick drive. With one stick avoided the other
// sets drive by itself over its full range; with both avoided the sticks add
// no drive, leaving it to Additional Inputs in Drive/Forward/Reverse mode.
func hosasDrive(left, right, combined float64, leftAvoid, rightAvoid bool) float64 {
	switch {
	case leftAvoid && rightAvoid:
		return 0
	case leftAvoid:
		return limitFraction(right, -1, 1)
	case rightAvoid:
		return limitFraction(left, -1, 1)
	}
	return combined
}

// racingPitch is the Xbox Left Stick Y sent while racing, which the Pitch
// meter also shows: a Traditional HOTAS stick's Y axis, replaced by any
// Pitch-mode inputs, then by learned Pitch Up/Down buttons (both held cancel).
func racingPitch(isHOTAS bool, hotasY float64, pitchAssigned bool, pitchValue float64, up, down bool) float64 {
	pitch := 0.0
	if isHOTAS {
		pitch = hotasY
	}
	if pitchAssigned {
		pitch = pitchValue
	}
	switch {
	case up && down:
		pitch = 0
	case up:
		pitch = 1
	case down:
		pitch = -1
	}
	return limitFraction(pitch, -1, 1)
}
