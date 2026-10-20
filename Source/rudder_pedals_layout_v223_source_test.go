package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// Guard the overlapping controls in the Rudder Pedals row against regressions.
// This checks their logical Win32 coordinates; native Windows DPI remains to test.
func TestV223RudderPedalModeClearsRefreshButton(t *testing.T) {
	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	label := regexp.MustCompile(`pedalUseLabel := a\.createLabel\("Mode", (\d+), 182, (\d+), 20\)\s+a\.createCombo\((\d+), 178, (\d+), 180, idPedalMode\)`).FindStringSubmatch(s)
	refresh := regexp.MustCompile(`a\.createButton\("Refresh devices", (\d+), 168, (\d+), 27, idRefresh\)`).FindStringSubmatch(s)
	if label == nil || refresh == nil {
		t.Fatal("cannot locate Rudder Pedals Mode dropdown or Refresh devices button")
	}
	val := func(str string) int {
		n, err := strconv.Atoi(str)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	labelLeft, labelWidth := val(label[1]), val(label[2])
	comboLeft, comboWidth := val(label[3]), val(label[4])
	refreshLeft := val(refresh[1])
	if labelLeft+labelWidth > comboLeft {
		t.Errorf("Mode label overlaps dropdown by %d pixels", labelLeft+labelWidth-comboLeft)
	}
	if gap := refreshLeft - (comboLeft + comboWidth); gap < 20 {
		t.Errorf("Mode dropdown needs >=20 logical-pixel clearance; got %d", gap)
	}
}
