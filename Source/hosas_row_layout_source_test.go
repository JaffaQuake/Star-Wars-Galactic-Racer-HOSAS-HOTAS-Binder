package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// Avoid Drive shares the Left/Right Input rows with Invert, Throttle axis and
// Detect. Guard the logical coordinates so no control overlaps its neighbour
// or the Control mode column; native Windows rendering remains to test.
func TestHOSASInputRowsFitAvoidDrive(t *testing.T) {
	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	controlMode := regexp.MustCompile(`a\.createLabel\("Control mode", (\d+), `).FindStringSubmatch(s)
	if controlMode == nil {
		t.Fatal("cannot locate Control mode label")
	}
	columnLeft, _ := strconv.Atoi(controlMode[1])
	for _, side := range []string{"Left", "Right"} {
		ids := []string{"id" + side + "Device", "id" + side + "Axis", "id" + side + "Invert", "id" + side + "ThrottleMode", "id" + side + "AvoidDrive", "idDetect" + side}
		prevRight, prevID := 0, ""
		for _, id := range ids {
			m := regexp.MustCompile(`a\.create(?:Combo|Check|Button)\((?:"[^"]*", )?(\d+), \d+, (\d+), \d+, ` + id + `\)`).FindStringSubmatch(s)
			if m == nil {
				t.Fatalf("cannot locate %s in the %s Input row", id, side)
			}
			x, _ := strconv.Atoi(m[1])
			w, _ := strconv.Atoi(m[2])
			if x < prevRight {
				t.Errorf("%s overlaps %s by %d pixels", id, prevID, prevRight-x)
			}
			prevRight, prevID = x+w, id
		}
		if gap := columnLeft - prevRight; gap < 20 {
			t.Errorf("%s Input row needs >=20 logical pixels before Control mode; got %d", side, gap)
		}
	}
}
