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
