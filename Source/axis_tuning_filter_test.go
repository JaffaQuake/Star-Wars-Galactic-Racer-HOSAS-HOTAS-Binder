package main

import "testing"

func TestFilterAxisTuningForAssignedActiveDigital(t *testing.T) {
	entries := axisTuneEntriesForControlMode("WHEEL")
	statuses := map[string]tuneInputInfo{
		profileWheelSteering:  {Enabled: true},
		profileWheelPedal1:    {Enabled: true},
		profileWheelPedal2:    {Enabled: false},
		profileWheelPedal3:    {Enabled: true, DigitalOnly: true},
		profileWheelHandbrake: {Enabled: true, DigitalOnly: true},
	}
	filtered := filterAxisTuneEntries(entries, statuses)
	var found = make(map[string]axisTuneEntry)
	for _, v := range filtered {
		found[v.Key] = v
	}
	if len(found) != 4 {
		t.Fatalf("selected %d: %v", len(found), filtered)
	}
	if _, ok := found[profileWheelPedal2]; ok {
		t.Fatal("unassigned brake must be filtered")
	}
	if found[profileWheelPedal1].DeadzoneOnly {
		t.Fatal("analog accelerator lost its curves")
	}
	if !found[profileWheelPedal3].DeadzoneOnly || !found[profileWheelHandbrake].DeadzoneOnly {
		t.Fatal("digital outputs must show only deadzone")
	}
}
func TestTuningDigitalModeAndLeverBehavior(t *testing.T) {
	for _, mode := range []string{"Boost", "Drift", "Boost/Drift", "Keyboard keys"} {
		if !isTuningDigitalMode(mode) {
			t.Errorf("%s should be digital", mode)
		}
	}
	for _, mode := range []string{"Fine steering", "Pitch", "Move Camera", "Off"} {
		if isTuningDigitalMode(mode) {
			t.Errorf("%s shouldn't count as digital", mode)
		}
	}
	entries := filterAxisTuneEntries([]axisTuneEntry{{Key: profileTwistLeft, Name: "Additional Input 1"}}, map[string]tuneInputInfo{
		profileTwistLeft: {Enabled: true, OneWay: true},
	})
	if len(entries) != 1 || !entries[0].OneWay {
		t.Fatal("lever type not preserved")
	}
}
func TestNoDeviceOrOffFiltersAll(t *testing.T) {
	entries := axisTuneEntriesForControlMode("HOTAS")
	if len(filterAxisTuneEntries(entries, nil)) != 0 {
		t.Fatal("unassigned axes should not show")
	}
}
