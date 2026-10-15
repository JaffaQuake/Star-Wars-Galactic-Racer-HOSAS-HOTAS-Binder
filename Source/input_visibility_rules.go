package main

import "strings"

// inputDeviceSelected treats a physical device as configured only if it has
// a real device identifier for the selected input backend. It prevents the
// placeholder "None" entry from exposing irrelevant axis/mode controls.
func inputDeviceSelected(backend string, winMMID int, directInputGUID string) bool {
	if strings.EqualFold(strings.TrimSpace(backend), "WinMM") {
		return winMMID >= 0
	}
	return strings.TrimSpace(directInputGUID) != ""
}
