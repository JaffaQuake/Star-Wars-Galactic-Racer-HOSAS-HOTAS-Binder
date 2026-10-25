//go:build windows

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDrivingWheelProfileIsolationAndDefaults(t *testing.T) {
	original := defaultConfig()
	if original.ControlMode != "HOSAS" {
		t.Fatal("Existing default control mode changed")
	}
	if original.DrivingWheel.Pedal1.DeviceID != -1 || original.DrivingWheel.Pedal2.DeviceID != -1 || original.DrivingWheel.Handbrake.DeviceID != -1 {
		t.Fatalf("new wheel devices should start unassigned: %+v", original.DrivingWheel)
	}
	if original.DrivingWheel.HandbrakeMode != "Drift" || original.DrivingWheel.Pedal3Mode != "Off" {
		t.Fatal("wrong default extra pedal/handbrake modes")
	}
	original.PedalMode = "Move Camera"
	original.DrivingWheel.Handbrake.DeviceID = 4
	original.DrivingWheel.HandbrakeMode = "Boost"
	original.DrivingWheel.Pedal1.DIGUID = "GUID-ACCEL"
	original.DrivingWheel.Pedal2.Axis = 5
	original.DrivingWheel.Pedal3Mode = "Drift"
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored config
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.PedalMode != "Move Camera" || restored.DrivingWheel.Handbrake.DeviceID != 4 || restored.DrivingWheel.HandbrakeMode != "Boost" || restored.DrivingWheel.Pedal1.DIGUID != "GUID-ACCEL" || restored.DrivingWheel.Pedal2.Axis != 5 || restored.DrivingWheel.Pedal3Mode != "Drift" {
		t.Fatalf("settings lost or mixed: %+v", restored)
	}
}

func TestDigitalActionsAfterDeadzone(t *testing.T) {
	for _, tc := range []struct {
		mode string
		raw  float64
		want uint16
	}{
		{"Boost", 0, 0}, {"Boost", .0001, XUSB_A}, {"Boost", -.5, XUSB_A},
		{"Drift", 1, XUSB_X}, {"Drift", 0, 0},
		{"Boost/Drift", -.5, XUSB_X}, {"Boost/Drift", .5, XUSB_A}, {"Boost/Drift", 0, 0},
		{"Off", 1, 0},
	} {
		if got := axisDigitalButtons(tc.mode, tc.raw); got != tc.want {
			t.Errorf("%s %v got %x want %x", tc.mode, tc.raw, got, tc.want)
		}
	}
	// Deadzone is applied in tuning before converting nonzero analog input to buttons.
	profile := defaultAxisTune(6)
	if axisDigitalButtons("Boost", tunedOneWay(.03, profile)) != 0 {
		t.Fatal("noise below deadzone triggered boost")
	}
	if axisDigitalButtons("Boost", tunedOneWay(.07, profile)) != XUSB_A {
		t.Fatal("real pedal movement did not trigger boost")
	}
}

func TestWheelBrakeAndAccelerationIndependent(t *testing.T) {
	rt, lt := drivingPedals(1, .5)
	if rt != 255 || lt != 128 {
		t.Fatalf("triggers RT=%d LT=%d", rt, lt)
	}
	rt, lt = drivingPedals(0, 1)
	if rt != 0 || lt != 255 {
		t.Fatalf("brake should not introduce acceleration RT=%d LT=%d", rt, lt)
	}
}

func TestWheelAxisMenusAreIsolated(t *testing.T) {
	entries := axisTuneEntriesForControlMode("WHEEL")
	found := map[string]bool{}
	for _, entry := range entries {
		found[entry.Key] = true
		if entry.Key == profileHOSASLeft || entry.Key == profileHOTASX || entry.Key == profilePedal {
			t.Fatalf("wrong-mode axis visible %s", entry.Key)
		}
	}
	for _, key := range []string{profileWheelSteering, profileWheelPedal1, profileWheelPedal2, profileWheelPedal3, profileWheelHandbrake, profileTwistLeft, profileTwistRight, profileTwist3, profileTwist4, profileTwist5, profileTwist6} {
		if !found[key] {
			t.Fatalf("missing tuning profile %s", key)
		}
	}
	normal := axisTuneEntriesForControlMode("HOSAS")
	for _, entry := range normal {
		if entry.Key == profileWheelPedal1 {
			t.Fatal("wheel tuning leaked into Podracer")
		}
	}
}

func TestAuxiliaryModeChoicesRespectAxisType(t *testing.T) {
	centered := auxiliaryModeOptions("Centered")
	lever := auxiliaryModeOptions("Lever")
	if strings.Join(centered[len(centered)-3:], ",") != "Boost/Drift,Boost/Boost,Drift/Drift" || lever[len(lever)-2] != "Boost" || lever[len(lever)-1] != "Drift" {
		t.Fatalf("wrong options: centered %v lever %v", centered, lever)
	}
}
