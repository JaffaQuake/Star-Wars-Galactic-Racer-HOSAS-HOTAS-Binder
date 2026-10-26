package main

import "testing"

func TestThemeSettingValues(t *testing.T) {
	if len(themeValues) != len(themeLabels) {
		t.Fatal("every theme value needs a dropdown label")
	}
	for _, tc := range []struct {
		saved string
		want  string
		index int
	}{
		{"", themeLight, 0}, // settings saved before the Theme option
		{"Light", themeLight, 0},
		{"Dark", themeDark, 1},
		{"System", themeSystem, 2},
		{"purple", themeLight, 0},
	} {
		if got := normalizeTheme(tc.saved); got != tc.want {
			t.Errorf("normalizeTheme(%q) = %q, want %q", tc.saved, got, tc.want)
		}
		if got := themeIndex(tc.saved); got != tc.index {
			t.Errorf("themeIndex(%q) = %d, want %d", tc.saved, got, tc.index)
		}
	}
}

func TestUseDarkTheme(t *testing.T) {
	for _, tc := range []struct {
		theme      string
		systemDark bool
		want       bool
	}{
		{themeLight, true, false},
		{themeDark, false, true},
		{themeSystem, true, true},
		{themeSystem, false, false},
		{"", true, false},
	} {
		if got := useDarkTheme(tc.theme, tc.systemDark); got != tc.want {
			t.Errorf("useDarkTheme(%q, system dark %v) = %v, want %v", tc.theme, tc.systemDark, got, tc.want)
		}
	}
}
