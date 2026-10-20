package main

import (
	"os"
	"strings"
	"testing"
)

func TestProfileCycleSourceWiring(t *testing.T) {
	m, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	p, err := os.ReadFile("profiles_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	main, profiles := string(m), string(p)
	for _, needle := range []string{
		`idProfileIncludeFirst+i`, `idCycleLearn`, `idCycleClear`,
		`a.startLearn("PROFILECYCLE"`, `a.updateCycleBinding(&r.Binding)`,
		`case WM_APP_PROFILE:`, `app.profileCyclePending.Store(false)`,
		`cycleButtonWasPressed = cyclePressed`, `cycleBind, cycleBound, nextProfile := a.cycleSettings()`,
		`ids = append(ids, cycleBind.DeviceID)`, `bindingsToSample = append(bindingsToSample, cycleBind)`,
		`a.profileSwitchMu.RLock()`, `a.profileSwitchMu.RUnlock()`,
		`a.profileSwitchMu.Lock()`, // other file below
		`walkingThrottleNeutral(in.Throttle)`, `walkingThrottleNeutral(in.LeftY)`,
		`walkingThrottleNeutral(in.RightY)`, `if c.LeftThrottleMode`,
		`if c.RightThrottleMode`, `samePhysicalBinding(b, cycleBind)`,
	} {
		if !strings.Contains(main, needle) && !strings.Contains(profiles, needle) {
			t.Errorf("missing profile-cycle integration: %s", needle)
		}
	}
	for _, needle := range []string{`CycleEnabled []bool`, `CycleBinding *binding`, `json:"cycle_enabled,omitempty"`, `json:"cycle_binding,omitempty"`, `a.xbox.Update(xusbReport{})`, `a.mappingActive.Store(true)`, `a.releaseAllKeyboardOutputs()`} {
		if !strings.Contains(profiles, needle) {
			t.Errorf("missing global profile persistence/safe switch: %s", needle)
		}
	}
}
