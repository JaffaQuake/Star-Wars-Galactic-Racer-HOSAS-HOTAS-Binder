//go:build windows

package main

import "testing"

// Light Mode must keep the original colours the app paints itself.
func TestLightPaletteKeepsOriginalColours(t *testing.T) {
	p := lightPalette()
	for _, tc := range []struct {
		name      string
		got, want uintptr
	}{
		{"window", p.Window, mainBackgroundColor},
		{"help shadow outer", p.HelpShadowOuter, colorRGB(207, 207, 207)},
		{"help shadow inner", p.HelpShadowInner, colorRGB(220, 220, 220)},
		{"help card", p.HelpCard, colorRGB(255, 255, 255)},
		{"help border", p.HelpBorder, colorRGB(199, 199, 199)},
		{"help text", p.HelpText, colorRGB(30, 30, 30)},
		{"graph background", p.GraphBack, colorRGB(255, 255, 255)},
		{"graph grid", p.GraphGrid, colorRGB(224, 228, 233)},
		{"graph diagonal", p.GraphDiagonal, colorRGB(158, 164, 173)},
		{"graph marker", p.GraphMarker, colorRGB(255, 255, 255)},
		{"monitor dot border", p.MonitorDotBorder, colorRGB(0, 0, 0)},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %#x, want %#x", tc.name, tc.got, tc.want)
		}
	}
}

func luminance(c uintptr) int { return int(c&0xFF)*299 + int(c>>8&0xFF)*587 + int(c>>16&0xFF)*114 }

func TestDarkPaletteIsDarkAndReadable(t *testing.T) {
	light, dark := lightPalette(), darkPalette()
	if luminance(dark.Window) >= luminance(light.Window) {
		t.Fatal("dark window background should be darker than the light one")
	}
	for _, tc := range []struct {
		name             string
		text, background uintptr
	}{
		{"labels", dark.Text, dark.Window},
		{"tab pages", dark.Text, dark.Panel},
		{"text boxes", dark.Text, dark.EditBack},
		{"buttons", dark.ButtonText, dark.ButtonFace},
		{"pressed buttons", dark.ButtonText, dark.ButtonPressed},
		{"help panel", dark.HelpText, dark.HelpCard},
	} {
		if luminance(tc.text)-luminance(tc.background) < 120000 {
			t.Errorf("%s: text is too close to its background", tc.name)
		}
	}
}

// Settings saved before the Theme option open in Light Mode.
func TestThemeDefaultsToLight(t *testing.T) {
	if got := normalizeTheme(defaultConfig().Theme); got != themeLight {
		t.Fatalf("default theme %q, want Light", got)
	}
}
