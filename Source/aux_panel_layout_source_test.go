package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// The Additional Inputs section must stay clear of the right-hand tab panel,
// and its widest row must fit beside the section's own scrollbar.
func TestAdditionalInputsSectionFits(t *testing.T) {
	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	atoi := func(v string) int {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	panel := regexp.MustCompile(`a\.createAuxInputPanel\((\d+), (\d+), (\d+), `).FindStringSubmatch(s)
	binding := regexp.MustCompile(`const bindingX = (\d+)`).FindStringSubmatch(s)
	tab := regexp.MustCompile(`a\.createControl\("SysTabControl32", "", [^,]+,\s*bindingX-(\d+),`).FindStringSubmatch(s)
	mode := regexp.MustCompile(`a\.createCombo\((\d+), y, (\d+), \d+, ids\[5\]\)`).FindStringSubmatch(s)
	if panel == nil || binding == nil || tab == nil || mode == nil {
		t.Fatal("cannot locate the section, tab panel or Mode dropdown in main.go")
	}
	left, width := atoi(panel[1]), atoi(panel[3])
	tabLeft := atoi(binding[1]) - atoi(tab[1])
	if left+width > tabLeft {
		t.Errorf("section ends at %d, overlapping the tab panel at %d", left+width, tabLeft)
	}
	const border, scrollbar = 1, 17 // logical pixels at 100% scaling
	usableRight := left + width - border - scrollbar
	if rowRight := atoi(mode[1]) + atoi(mode[2]); rowRight > usableRight {
		t.Errorf("Mode dropdown ends at %d, under the section's scrollbar (from %d)", rowRight, usableRight)
	}
}
