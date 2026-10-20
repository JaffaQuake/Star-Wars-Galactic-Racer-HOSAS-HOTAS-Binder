//go:build windows

package main

import (
	"encoding/json"
	"testing"
)

func TestProfilesCloneIndependentAndPreserveMode(t *testing.T) {
	base := defaultConfig()
	base.ControlMode = "HOTAS"
	base.Bindings["WALKMODE"] = binding{Kind: "button", DeviceID: 4, Button: 7}
	base.Tuning = map[string]axisTuneProfile{"hotas_x": {Deadzone: 10, Weighting: axisInt(30)}}
	slots := defaultProfileCollection(base)
	if slots.Active != 1 || slots.Slots[0].ControlMode != "HOTAS" || slots.Slots[1] != nil {
		t.Fatal("legacy migration did not initialize only profile 1")
	}
	second := cloneProfileConfig(*slots.Slots[0])
	second.ControlMode = "WHEEL"
	second.Bindings["WALKMODE"] = binding{Kind: "button", DeviceID: 5, Button: 9}
	secondTune := second.Tuning["hotas_x"]
	*secondTune.Weighting = 90
	second.Tuning["hotas_x"] = secondTune
	slots.Active = 2
	slots.Slots[1] = &second
	if slots.Slots[0].ControlMode != "HOTAS" || slots.Slots[0].Bindings["WALKMODE"].Button != 7 || *slots.Slots[0].Tuning["hotas_x"].Weighting != 30 {
		t.Fatal("second profile changed the original profile")
	}
	bytes, err := json.Marshal(slots)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := readProfileCollection(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Active != 2 || recovered.Slots[1].ControlMode != "WHEEL" || recovered.Slots[0].ControlMode != "HOTAS" {
		t.Fatal("profile round-trip lost active slot or control mode")
	}
}

func TestRejectMalformedProfilesWithoutChangingLegacy(t *testing.T) {
	for _, payload := range []string{
		`invalid`,
		`{"version":1,"active":0,"slots":[]}`,
		`{"version":1,"active":5,"slots":[null,null,null,null,null]}`,
		`{"version":99,"active":1,"slots":[{}]}`,
	} {
		if _, err := readProfileCollection([]byte(payload)); err == nil {
			t.Fatalf("should reject invalid profiles: %s", payload)
		}
	}
}

func TestProfileCycleMigrationAndRoundTrip(t *testing.T) {
	p := defaultProfileCollection(defaultConfig())
	if len(p.CycleEnabled) != 5 {
		t.Fatal("default includes five")
	}
	for i, enabled := range p.CycleEnabled {
		if !enabled {
			t.Fatalf("profile %d not included", i+1)
		}
	}
	saved, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]interface{}
	if err := json.Unmarshal(saved, &legacy); err != nil {
		t.Fatal(err)
	}
	delete(legacy, "cycle_enabled")
	delete(legacy, "cycle_binding")
	old, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := readProfileCollection(old)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.CycleEnabled) != 5 || !loaded.CycleEnabled[4] {
		t.Fatal("v2.2.1 migration skipped slots")
	}
	loaded.CycleEnabled = []bool{true, false, false, true, false}
	loaded.CycleBinding = &binding{Kind: "button", Backend: "DirectInput", DeviceID: -1, DIGUID: "example", Button: 4}
	saved, err = json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readProfileCollection(saved)
	if err != nil {
		t.Fatal(err)
	}
	if got.CycleBinding == nil || got.CycleBinding.Button != 4 || got.CycleEnabled[1] || !got.CycleEnabled[3] {
		t.Fatal("cycle global settings lost")
	}
}
func TestProfileCycleNextEligibleAndNoSlots(t *testing.T) {
	a := &appState{profiles: defaultProfileCollection(defaultConfig())}
	a.profiles.Active = 1
	a.profiles.CycleEnabled = []bool{true, false, false, true, false}
	a.profiles.CycleBinding = &binding{Kind: "button", DeviceID: 0, Button: 2}
	_, bound, next := a.cycleSettings()
	if !bound || next != 4 {
		t.Fatalf("1->4 next=%d", next)
	}
	a.profiles.Active = 4
	_, _, next = a.cycleSettings()
	if next != 1 {
		t.Fatalf("4->1 next=%d", next)
	}
	a.profiles.Active = 2
	_, _, next = a.cycleSettings()
	if next != 4 {
		t.Fatalf("unchecked active 2->4 next=%d", next)
	}
	a.profiles.CycleEnabled = []bool{false, false, false, false, false}
	_, _, next = a.cycleSettings()
	if next != 0 {
		t.Fatalf("all excluded should not switch: %d", next)
	}
}
