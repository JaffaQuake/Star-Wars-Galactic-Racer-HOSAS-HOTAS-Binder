//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPollingRateSurvivesConfigRoundTrip(t *testing.T) {
	originalApp := app
	t.Cleanup(func() { app = originalApp })
	t.Setenv("LOCALAPPDATA", t.TempDir())
	for _, rate := range []int{100, 200, 250, 500, 1000} {
		app = &appState{cfg: defaultConfig()}
		app.cfg.PollingHz = rate
		saveConfig()
		cfgBytes, err := os.ReadFile(filepath.Join(os.Getenv("LOCALAPPDATA"), "GalacticRacerHOSAS", "hosas_config.json"))
		if err != nil {
			t.Fatal(err)
		}
		if len(cfgBytes) == 0 {
			t.Fatal("empty settings")
		}
		loaded := defaultConfig()
		loadConfig(&loaded)
		if loaded.PollingHz != rate {
			t.Fatalf("polling %d reset to %d after reloading settings", rate, loaded.PollingHz)
		}
	}
}
