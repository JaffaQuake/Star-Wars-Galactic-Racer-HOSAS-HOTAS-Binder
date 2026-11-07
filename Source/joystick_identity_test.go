package main

import (
	"reflect"
	"testing"
)

const (
	idA     = "6F1D2B60-D5A0-11CF-BFC7-444553540000"
	idB     = "6F1D2B61-D5A0-11CF-BFC7-444553540000"
	idC     = "6F1D2B62-D5A0-11CF-BFC7-444553540000"
	idPedal = "0F2A9B10-1111-11EF-8001-444553540000"
	portX   = `\\?\hid#vid_231d&pid_0200#7&1a2b&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`
	portY   = `\\?\hid#vid_231d&pid_0200#7&3c4d&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`
	pathPed = `\\?\hid#vid_06a3&pid_0763#serial123#{4d1e55b2-f16f-11cf-88cb-001111000030}`
)

func ids(list ...string) map[string]bool {
	out := map[string]bool{}
	for _, id := range list {
		out[id] = true
	}
	return out
}

func TestJoystickIDChanges(t *testing.T) {
	for _, tc := range []struct {
		name           string
		saved, current map[string]string
		inUse          map[string]bool
		want           map[string]string
	}{
		{
			name:    "nothing changed",
			saved:   map[string]string{idA: portX, idB: portY},
			current: map[string]string{idA: portX, idB: portY},
			inUse:   ids(idA, idB),
			want:    map[string]string{},
		},
		{
			name:    "Windows swapped the two sticks' IDs",
			saved:   map[string]string{idA: portX, idB: portY, idPedal: pathPed},
			current: map[string]string{idA: portY, idB: portX, idPedal: pathPed},
			inUse:   ids(idA, idB, idPedal),
			want:    map[string]string{idA: idB, idB: idA},
		},
		{
			name:    "three controllers rotated",
			saved:   map[string]string{idA: portX, idB: portY, idC: pathPed},
			current: map[string]string{idA: pathPed, idB: portX, idC: portY},
			inUse:   ids(idA, idB, idC),
			want:    map[string]string{idA: idB, idB: idC, idC: idA},
		},
		{
			name:    "a stick got a brand-new ID",
			saved:   map[string]string{idA: portX},
			current: map[string]string{idC: portX},
			inUse:   ids(idA),
			want:    map[string]string{idA: idC},
		},
		{
			name: "the other stick is unplugged: its ID now means the first stick, so leave both",
			// Moving A to B would leave Right Input (B) on the left stick too.
			saved:   map[string]string{idA: portX, idB: portY},
			current: map[string]string{idB: portX},
			inUse:   ids(idA, idB),
			want:    map[string]string{},
		},
		{
			name:    "the new ID is used by this configuration but was never saved",
			saved:   map[string]string{idA: portX},
			current: map[string]string{idB: portX},
			inUse:   ids(idA, idB),
			want:    map[string]string{},
		},
		{
			name:    "two controllers report the same path: cannot tell them apart",
			saved:   map[string]string{idA: portX, idB: portY},
			current: map[string]string{idA: portY, idB: portY},
			inUse:   ids(idA, idB),
			want:    map[string]string{},
		},
		{
			name:    "no saved paths yet (first run of this version)",
			saved:   nil,
			current: map[string]string{idA: portY, idB: portX},
			inUse:   ids(idA, idB),
			want:    map[string]string{},
		},
		{
			name:    "letter case and spacing do not matter",
			saved:   map[string]string{" 6f1d2b60-d5a0-11cf-bfc7-444553540000": `\\?\HID#VID_231D&PID_0200#7&1A2B&0&0000#{4D1E55B2-F16F-11CF-88CB-001111000030}`, idB: portY},
			current: map[string]string{idA: portY, idB: portX},
			inUse:   ids(idA, idB),
			want:    map[string]string{idA: idB, idB: idA},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := joystickIDChanges(tc.saved, tc.current, tc.inUse); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMergeDevicePaths(t *testing.T) {
	saved := map[string]string{idA: portX, idB: portY, idC: "\\\\?\\hid#old-unplugged"}
	current := map[string]string{idA: portY, idB: portX}
	changes := map[string]string{idA: idB, idB: idA}
	want := map[string]string{idA: portY, idB: portX, idC: "\\\\?\\hid#old-unplugged"}
	if got := mergeDevicePaths(saved, current, changes); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// A disconnected ID whose path now belongs to a connected controller is stale.
	got := mergeDevicePaths(map[string]string{idA: portX}, map[string]string{idB: portX}, nil)
	if !reflect.DeepEqual(got, map[string]string{idB: portX}) {
		t.Errorf("stale entry kept: %v", got)
	}
}

func TestRemapGUIDs(t *testing.T) {
	text := `{"left_di_guid":"` + idA + `","right_di_guid":"` + idB + `","pedal_di_guid":"` + idPedal +
		`","bindings":{"A":{"di_guid":"6f1d2b60-d5a0-11cf-bfc7-444553540000"}}}`
	got, n := remapGUIDs(text, map[string]string{idA: idB, idB: idA})
	want := `{"left_di_guid":"` + idB + `","right_di_guid":"` + idA + `","pedal_di_guid":"` + idPedal +
		`","bindings":{"A":{"di_guid":"` + idB + `"}}}`
	if got != want || n != 3 {
		t.Errorf("got %d replacements:\n%s\nwant 3:\n%s", n, got, want)
	}
	if ids := guidsIn(text); !ids[idA] || !ids[idB] || !ids[idPedal] || len(ids) != 3 {
		t.Errorf("guidsIn: %v", ids)
	}
}
