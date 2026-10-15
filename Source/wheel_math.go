package main

import "math"

// digitalAxisAction converts an already deadzoned analog input to a button action.
// Centered negative is Drift and positive Boost. Lever Boost/Drift options
// ignore polarity (Output Negative), and a digital pressed state is positive 1.
func digitalAxisAction(mode string, normalized float64) string {
	if normalized == 0 || math.IsNaN(normalized) {
		return ""
	}
	switch mode {
	case "Boost":
		return "Boost"
	case "Drift":
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
