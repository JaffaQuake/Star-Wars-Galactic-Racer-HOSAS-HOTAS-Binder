package main

import "math"

// digitalAxisAction converts an already deadzoned analog input to a button action.
// Centered negative is Drift and positive Boost; Boost/Boost and Drift/Drift
// use the same action on both sides. Lever Boost/Drift options ignore
// polarity (Output Negative), and a digital pressed state is positive 1.
func digitalAxisAction(mode string, normalized float64) string {
	if normalized == 0 || math.IsNaN(normalized) {
		return ""
	}
	switch mode {
	case "Boost", "Boost/Boost":
		return "Boost"
	case "Drift", "Drift/Drift":
		return "Drift"
	case "Boost/Drift":
		if normalized < 0 {
			return "Drift"
		}
		return "Boost"
	}
	return ""
}

func pedalTriggerByte(normalized float64) byte {
	if math.IsNaN(normalized) {
		return 0
	}
	return byte(math.Round(math.Max(0, math.Min(1, normalized)) * 255))
}

// Boost/Drift-style Additional Input modes press only once the input passes
// its Threshold %, measured like the Keyboard keys threshold (before Axis
// Tuning). tuned is the deadzoned value, so the deadzone still applies too.
func thresholdedButtonValue(normalized, tuned float64, threshold int) float64 {
	if math.IsNaN(normalized) || math.Abs(normalized) < float64(threshold)/100 {
		return 0
	}
	return tuned
}

func isAuxiliaryButtonMode(mode string) bool {
	switch mode {
	case "Boost", "Drift", "Boost/Drift", "Boost/Boost", "Drift/Drift":
		return true
	}
	return false
}
