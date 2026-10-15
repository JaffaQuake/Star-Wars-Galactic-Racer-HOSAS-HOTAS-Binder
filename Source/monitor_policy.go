package main

// shouldRenderMonitor preserves the historical paused display unless the
// user explicitly enables live UI updates during controller gameplay.
func shouldRenderMonitor(mappingActive, monitorDuringGameplay bool) bool {
	return !mappingActive || monitorDuringGameplay
}
