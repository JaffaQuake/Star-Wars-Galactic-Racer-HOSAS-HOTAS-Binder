//go:build windows

package main

import (
	"testing"
	"unsafe"
)

func TestDIPropGUIDAndPathLayout(t *testing.T) {
	// DIPROPGUIDANDPATH: 16-byte DIPROPHEADER, 16-byte GUID, WCHAR[MAX_PATH].
	if got := unsafe.Sizeof(diPropGUIDAndPath{}); got != 552 {
		t.Fatalf("size %d, want 552", got)
	}
}

// With the real settings type: a swap moves every assignment, in all the
// places a controller ID is stored, and records the new paths.
func TestReconcileConfigIDsFollowsSwappedSticks(t *testing.T) {
	c := defaultConfig()
	c.LeftDIGUID, c.RightDIGUID, c.PedalDIGUID = idA, idB, idPedal
	c.TwistLeft.DIGUID = idA
	c.Bindings["A"] = binding{Kind: "button", Backend: "DirectInput", DIGUID: idA, Button: 3}
	c.Bindings["B"] = binding{Kind: "button", Backend: "DirectInput", DIGUID: idB, Button: 1}
	c.DevicePaths = map[string]string{idA: portX, idB: portY, idPedal: pathPed}

	got, changes := reconcileConfigIDs(c, map[string]string{idA: portY, idB: portX, idPedal: pathPed})
	if len(changes) != 2 {
		t.Fatalf("changes %v", changes)
	}
	if got.LeftDIGUID != idB || got.RightDIGUID != idA || got.PedalDIGUID != idPedal || got.TwistLeft.DIGUID != idB {
		t.Errorf("assignments did not follow the sticks: left %s right %s pedal %s twist %s", got.LeftDIGUID, got.RightDIGUID, got.PedalDIGUID, got.TwistLeft.DIGUID)
	}
	if got.Bindings["A"].DIGUID != idB || got.Bindings["B"].DIGUID != idA || got.Bindings["A"].Button != 3 {
		t.Errorf("learned buttons did not follow the sticks: %+v", got.Bindings)
	}
	if got.DevicePaths[idA] != portY || got.DevicePaths[idB] != portX {
		t.Errorf("paths not updated: %v", got.DevicePaths)
	}

	again, changes := reconcileConfigIDs(got, map[string]string{idA: portY, idB: portX, idPedal: pathPed})
	if len(changes) != 0 || again.LeftDIGUID != idB {
		t.Errorf("a second check must change nothing: %v", changes)
	}
}
