//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMonitorDuringGameplaySettingSurvivesReload(t *testing.T) {
	oldApp := app
	t.Cleanup(func() { app = oldApp })
	t.Setenv("LOCALAPPDATA", t.TempDir())
	app = &appState{cfg: defaultConfig()}
	if app.cfg.MonitorDuringGameplay {
		t.Fatal("monitor should default off")
	}
	app.cfg.MonitorDuringGameplay = true
	saveConfig()
	loaded := defaultConfig()
	loadConfig(&loaded)
	if !loaded.MonitorDuringGameplay {
		t.Fatal("true opt-in not restored")
	}

	app.cfg.MonitorDuringGameplay = false
	saveConfig()
	loaded = defaultConfig()
	loadConfig(&loaded)
	if loaded.MonitorDuringGameplay {
		t.Fatal("opted-out setting not restored")
	}

	raw, err := os.ReadFile(filepath.Join(os.Getenv("LOCALAPPDATA"), "GalacticRacerHOSAS", "hosas_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err = json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if string(m["monitor_during_gameplay"]) != "false" {
		t.Fatalf("unexpected persisted setting: %s", m["monitor_during_gameplay"])
	}
}
