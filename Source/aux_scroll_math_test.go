package main

import "testing"

// Eight 78px rows, two visible.
const testContent, testView, testRow = 8 * 78, 2 * 78, 78

func TestAuxScrollAfter(t *testing.T) {
	for _, tc := range []struct {
		name             string
		code, thumb, pos int
		want             int
	}{
		{"line down", scrollLineDown, 0, 0, 78},
		{"line up at top stays", scrollLineUp, 0, 0, 0},
		{"page down moves two rows", scrollPageDown, 0, 78, 234},
		{"page down stops at the last rows", scrollPageDown, 0, 390, 468},
		{"page up", scrollPageUp, 0, 468, 312},
		{"thumb drag", scrollThumbTrack, 200, 0, 200},
		{"thumb past the end", scrollThumbPosition, 900, 0, 468},
		{"top", scrollTop, 0, 300, 0},
		{"bottom", scrollBottom, 0, 0, 468},
	} {
		if got := auxScrollAfter(tc.code, tc.thumb, tc.pos, testContent, testView, testRow); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestAuxWheelScroll(t *testing.T) {
	for _, tc := range []struct {
		name         string
		delta, pos   int
		want         int
		wantMovement bool
	}{
		{"wheel down one notch", -120, 0, 78, true},
		{"wheel up one notch", 120, 156, 78, true},
		{"two notches", -240, 0, 156, true},
		{"precision touchpad nudge", -30, 0, 78, true},
		{"already at top", 120, 0, 0, false},
		{"already at bottom", -120, 468, 468, false},
	} {
		got, moved := auxWheelScroll(tc.delta, tc.pos, testContent, testView, testRow)
		if got != tc.want || moved != tc.wantMovement {
			t.Errorf("%s: got %d moved=%v, want %d moved=%v", tc.name, got, moved, tc.want, tc.wantMovement)
		}
	}
}

func TestClampScrollWhenEverythingFits(t *testing.T) {
	if got := clampScroll(50, 100, 200); got != 0 {
		t.Fatalf("content shorter than the view should not scroll, got %d", got)
	}
}
