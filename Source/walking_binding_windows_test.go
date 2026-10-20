package main

import "testing"

func TestReservedWalkingBinding(t *testing.T) {
	b := binding{Kind: "button", DeviceID: 3, Button: 7}
	if !samePhysicalBinding(b, b) || samePhysicalBinding(b, binding{Kind: "button", DeviceID: 3, Button: 8}) {
		t.Fatal("button identity")
	}
	di := binding{Kind: "pov", DeviceID: -1, DIGUID: "device", POVIndex: 1, POV: 9000}
	if !samePhysicalBinding(di, di) || samePhysicalBinding(di, binding{Kind: "pov", DeviceID: -1, DIGUID: "device", POVIndex: 1, POV: 0}) {
		t.Fatal("POV identity")
	}
}
