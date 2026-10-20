package main

import (
	"os"
	"strings"
	"testing"
)

func TestFiveProfileSlotsUIAndPersistenceSource(t *testing.T) {
	ui, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	core, err := os.ReadFile("profiles_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(ui)
	p := string(core)
	for _, required := range []string{
		`a.createLabel("Profiles", 830, 52, 90, 22)`,
		`925+i*38, 45, 33, 27, idProfileFirst+i`,
		`a.createCombo(925, 105, 185, 160, idControlMode)`,
		`a.createCombo(925, 137, 185, 140, idInputBackend)`,
		`a.switchProfile(id - idProfileFirst + 1)`,
		`a.refreshProfileButtons()`,
		`app.persistProfiles(c)`,
	} {
		if !strings.Contains(s, required) {
			t.Errorf("profile UI/source integration missing: %s", required)
		}
	}
	for _, required := range []string{
		`hosas_profiles.json`, `const profileCount = 5`,
		`a.mappingActive.Store(false)`, `a.xbox.Update(xusbReport{})`, `a.syncConfigFromUI()`, `a.populateCombos()`,
		`a.tuning.draft = cloneAxisProfiles(next.Tuning)`,
		`a.updateBindingLabels()`,
	} {
		if !strings.Contains(p, required) {
			t.Errorf("profile management regression guard missing: %s", required)
		}
	}
}
