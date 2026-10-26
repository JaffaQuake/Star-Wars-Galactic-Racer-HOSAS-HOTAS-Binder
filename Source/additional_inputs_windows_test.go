//go:build windows

package main

import (
	"encoding/json"
	"testing"
)

// Additional Inputs 3-8 need their own controls; a shared ID would make
// two rows edit the same setting.
func TestAdditionalInputIDsAreUnique(t *testing.T) {
	seen := map[int]string{}
	for slot := 0; slot < auxiliaryInputCount; slot++ {
		for field, id := range auxiliaryIDs(slot) {
			if prev, ok := seen[id]; ok {
				t.Fatalf("%s field %d reuses ID %d from %s", auxiliaryInputName(slot), field, id, prev)
			}
			seen[id] = auxiliaryInputName(slot)
			if got := auxiliarySlotForID(id, field); got != slot {
				t.Fatalf("auxiliarySlotForID(%d, %d) = %d, want %d", id, field, got, slot)
			}
		}
		if id := auxiliaryIDs(slot)[len(auxiliaryIDs(slot))-1]; id >= idLearnBase {
			t.Fatalf("%s ID %d overlaps the learn-button range", auxiliaryInputName(slot), id)
		}
	}
}

func TestAdditionalInputsDefaultAndRoundTrip(t *testing.T) {
	c := defaultConfig()
	for slot, v := range c.auxiliaryAxes() {
		if v.DeviceID != -1 || v.Mode != "Off" {
			t.Fatalf("%s default %+v", auxiliaryInputName(slot), *v)
		}
	}
	c.Twist8.Mode = "Move Camera"
	c.Twist8.DIGUID = "guid-8"
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var back config
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Twist8.Mode != "Move Camera" || back.Twist8.DIGUID != "guid-8" || back.Twist7.DIGUID != "" {
		t.Fatalf("twist_8 round-trip lost settings: %+v", back.Twist8)
	}
}

// Profiles written before v2.2.4 have no twist_3 to twist_8. They must load as
// unassigned inputs rather than zero values that select WinMM device 0.
func TestOlderProfilesGetUnassignedAdditionalInputs(t *testing.T) {
	payload := `{"version":1,"active":1,"slots":[{"control_mode":"HOSAS","twist_left":{"device_id":2,"mode":"Pitch"}},null,null,null,null]}`
	p, err := readProfileCollection([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	slot := p.Slots[0]
	if slot.TwistLeft.DeviceID != 2 || slot.TwistLeft.Mode != "Pitch" {
		t.Fatalf("existing Additional Input 1 changed: %+v", slot.TwistLeft)
	}
	for _, v := range []auxiliaryAxisConfig{slot.Twist3, slot.Twist4, slot.Twist5, slot.Twist6, slot.Twist7, slot.Twist8} {
		if v != defaultAuxiliaryAxis() {
			t.Fatalf("missing Additional Input did not get defaults: %+v", v)
		}
	}
	if _, ok := slot.Tuning[profileTwist8]; !ok {
		t.Fatal("Additional Input 8 has no tuning profile")
	}
}

func TestAdditionalInputButtonThresholdSavedAndMigrated(t *testing.T) {
	if got := auxiliaryButtonThreshold(defaultAuxiliaryAxis()); got != 20 {
		t.Fatalf("new inputs should default to 20%%, got %d", got)
	}
	c := defaultConfig()
	c.Twist5.ButtonThreshold = 35
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var back config
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Twist5.ButtonThreshold != 35 {
		t.Fatalf("button_threshold not saved: %+v", back.Twist5)
	}
	// Settings written before Threshold % existed: keep pressing at the deadzone.
	var old auxiliaryAxisConfig
	if err := json.Unmarshal([]byte(`{"device_id":1,"mode":"Boost/Drift","deadzone":12}`), &old); err != nil {
		t.Fatal(err)
	}
	validateAuxiliaryAxis(&old, true, 7)
	if got := auxiliaryButtonThreshold(old); got != 12 {
		t.Fatalf("older settings should use their 12%% deadzone, got %d", got)
	}
	old.ButtonThreshold = 400
	validateAuxiliaryAxis(&old, true, 7)
	if got := auxiliaryButtonThreshold(old); got != 12 {
		t.Fatalf("an invalid saved threshold should fall back to the deadzone, got %d", got)
	}
}
