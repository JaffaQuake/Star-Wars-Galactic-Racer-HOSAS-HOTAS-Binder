package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Stand-ins shaped like config and profileCollection: a nested input group,
// a map this build owns, omitempty settings and an array of profile slots.
type testInput struct {
	DeviceID int    `json:"device_id"`
	Mode     string `json:"mode"`
}

type testSettings struct {
	Twist    testInput         `json:"twist_left"`
	Bindings map[string]int    `json:"bindings"`
	Tuning   map[string]int    `json:"axis_tuning,omitempty"`
	Polling  int               `json:"polling_hz"`
	Notes    string            `json:"notes,omitempty"`
	Internal string            `json:"-"`
	Extra    map[string]string `json:"extra,omitempty"`
}

type testProfiles struct {
	Active int              `json:"active"`
	Slots  [3]*testSettings `json:"slots"`
}

func savedJSON(t *testing.T, fresh any, old string) map[string]any {
	t.Helper()
	data, err := json.MarshalIndent(fresh, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(keepUnknownSettings(data, []byte(old), reflect.TypeOf(fresh)), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestKeepsSettingsThisBuildDoesNotKnow(t *testing.T) {
	old := `{"twist_left":{"device_id":3,"mode":"Pitch","button_threshold":35},
		"twist_7":{"device_id":5,"mode":"Drive"},"theme":"Dark","polling_hz":250,
		"bindings":{"A":1,"B":2}}`
	fresh := testSettings{Twist: testInput{DeviceID: 4, Mode: "Off"}, Bindings: map[string]int{"A": 9}, Polling: 500}
	out := savedJSON(t, fresh, old)

	if out["theme"] != "Dark" {
		t.Errorf("unknown top-level setting lost: %v", out["theme"])
	}
	if twist7, ok := out["twist_7"].(map[string]any); !ok || twist7["mode"] != "Drive" {
		t.Errorf("unknown Additional Input lost: %v", out["twist_7"])
	}
	twist := out["twist_left"].(map[string]any)
	if twist["button_threshold"] != float64(35) {
		t.Errorf("unknown field inside a known input lost: %v", twist)
	}
	if twist["device_id"] != float64(4) || twist["mode"] != "Off" || out["polling_hz"] != float64(500) {
		t.Errorf("this build's own values must win: %v", out)
	}
	if bindings := out["bindings"].(map[string]any); len(bindings) != 1 || bindings["A"] != float64(9) {
		t.Errorf("a cleared binding came back: %v", bindings)
	}
}

func TestDoesNotRestoreKnownSettingsThisBuildLeftOut(t *testing.T) {
	old := `{"axis_tuning":{"twist_3":7},"notes":"old","polling_hz":250,"bindings":{}}`
	out := savedJSON(t, testSettings{Polling: 500}, old)
	if _, ok := out["axis_tuning"]; ok {
		t.Errorf("an omitted known setting was restored: %v", out["axis_tuning"])
	}
	if _, ok := out["notes"]; ok {
		t.Errorf("an omitted known setting was restored: %v", out["notes"])
	}
}

func TestKeepsUnknownSettingsInEveryProfileSlot(t *testing.T) {
	old := `{"active":2,"cycle_binding":{"kind":"button"},"slots":[
		{"twist_8":{"mode":"Boost"},"polling_hz":100},
		null,
		{"theme":"System","polling_hz":200}]}`
	fresh := testProfiles{Active: 1, Slots: [3]*testSettings{{Polling: 500}, {Polling: 300}, nil}}
	out := savedJSON(t, fresh, old)
	if out["cycle_binding"] == nil {
		t.Error("unknown collection-level setting lost")
	}
	slots := out["slots"].([]any)
	first := slots[0].(map[string]any)
	if first["twist_8"] == nil || first["polling_hz"] != float64(500) {
		t.Errorf("slot 1: %v", first)
	}
	if second := slots[1].(map[string]any); len(second) != 3 || second["polling_hz"] != float64(300) {
		t.Errorf("slot 2 was empty on disk, so it should be exactly this build's: %v", second)
	}
	if slots[2] != nil {
		t.Errorf("a slot this build has not created must stay empty: %v", slots[2])
	}
}

func TestUnreadableOldFileLeavesFreshUnchanged(t *testing.T) {
	fresh := []byte("{\n  \"polling_hz\": 500\n}")
	for _, old := range []string{"", "not json", "[]", "null"} {
		if got := keepUnknownSettings(fresh, []byte(old), reflect.TypeOf(testSettings{})); string(got) != string(fresh) {
			t.Errorf("old %q changed the saved file: %s", old, got)
		}
	}
}
