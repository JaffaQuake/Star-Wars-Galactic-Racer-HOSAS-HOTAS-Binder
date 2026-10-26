//go:build windows

package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

// With the real settings types: what another build wrote survives, and this
// build's values win for everything it knows.
func TestRealSettingsKeepUnknownEntries(t *testing.T) {
	c := defaultConfig()
	c.PollingHz = 250
	fresh, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	old := []byte(`{"polling_hz":1000,"theme":"Dark","twist_9":{"device_id":2,"mode":"Pitch"},"twist_left":{"future_field":7},"bindings":{"A":{"kind":"button"}}}`)
	var out map[string]json.RawMessage
	if err := json.Unmarshal(keepUnknownSettings(fresh, old, reflect.TypeOf(c)), &out); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"theme", "twist_9"} {
		if out[key] == nil {
			t.Errorf("%s was dropped", key)
		}
	}
	var twist map[string]any
	_ = json.Unmarshal(out["twist_left"], &twist)
	if twist["future_field"] != float64(7) || twist["device_id"] != float64(-1) {
		t.Errorf("twist_left should keep future_field and this build's device_id: %v", twist)
	}
	if string(out["polling_hz"]) != "250" || string(out["bindings"]) != "{}" {
		t.Errorf("this build's values must win: polling %s, bindings %s", out["polling_hz"], out["bindings"])
	}
	var back config
	if err := json.Unmarshal(keepUnknownSettings(fresh, old, reflect.TypeOf(c)), &back); err != nil || back.PollingHz != 250 {
		t.Fatalf("merged settings must still load: %v %+v", err, back.PollingHz)
	}

	p := defaultProfileCollection(c)
	freshProfiles, _ := json.MarshalIndent(p, "", "  ")
	oldProfiles := []byte(`{"version":1,"active":1,"slots":[{"twist_9":{"mode":"Drive"}},null,null,null,null]}`)
	var profiles struct {
		Slots []map[string]json.RawMessage `json:"slots"`
	}
	if err := json.Unmarshal(keepUnknownSettings(freshProfiles, oldProfiles, reflect.TypeOf(p)), &profiles); err != nil {
		t.Fatal(err)
	}
	if profiles.Slots[0]["twist_9"] == nil {
		t.Error("profile 1 lost a setting this build does not know")
	}
}
