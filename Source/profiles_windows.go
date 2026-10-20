//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Profile slots are independent complete configuration snapshots. The active
// profile also remains in hosas_config.json for older releases and safe updates.
const profileCount = 5

type profileCollection struct {
	Version      int                   `json:"version"`
	Active       int                   `json:"active"` // 1..5, unlike zero-based UI offsets
	Slots        [profileCount]*config `json:"slots"`
	CycleEnabled []bool                `json:"cycle_enabled,omitempty"` // global, not per-profile settings
	CycleBinding *binding              `json:"cycle_binding,omitempty"` // global hardware assignment
}

func cloneProfileConfig(src config) config {
	dst := src
	if src.Bindings != nil {
		dst.Bindings = make(map[string]binding, len(src.Bindings))
		for k, b := range src.Bindings {
			dst.Bindings[k] = b
		}
	}
	if src.Tuning != nil {
		dst.Tuning = cloneAxisProfiles(src.Tuning)
		// The tuning map contains optional pointer-valued weighting fields.
		// Copy those too so profile edits can never affect another slot.
		for key, value := range dst.Tuning {
			for _, field := range []**int{&value.Weighting, &value.Stage1Weight, &value.Stage2Weight} {
				if *field != nil {
					copyValue := **field
					*field = &copyValue
				}
			}
			dst.Tuning[key] = value
		}
	}
	return dst
}

func profilesFilePath() string {
	return filepath.Join(filepath.Dir(configPath()), "hosas_profiles.json")
}

// Only profile 1 is initialized from legacy settings. Untouched slots are
// initialized from the active setup on first use, then become independent.
func defaultProfileCollection(current config) profileCollection {
	p := profileCollection{Version: 1, Active: 1, CycleEnabled: []bool{true, true, true, true, true}}
	first := cloneProfileConfig(current)
	p.Slots[0] = &first
	return p
}

func readProfileCollection(data []byte) (profileCollection, error) {
	var p profileCollection
	if err := json.Unmarshal(data, &p); err != nil {
		return p, err
	}
	if p.Version != 1 || p.Active < 1 || p.Active > profileCount || p.Slots[p.Active-1] == nil {
		return profileCollection{}, fmt.Errorf("invalid profile version, active slot or contents")
	}
	// Existing v2.2.1 JSON has neither cycling field. All five profiles
	// remain included on upgrade until a user changes the checkboxes.
	if p.CycleEnabled == nil {
		p.CycleEnabled = []bool{true, true, true, true, true}
	}
	if len(p.CycleEnabled) != profileCount {
		return profileCollection{}, fmt.Errorf("invalid cycle-enabled profile count")
	}
	if p.CycleBinding != nil {
		if p.CycleBinding.Kind != "button" && p.CycleBinding.Kind != "pov" {
			return profileCollection{}, fmt.Errorf("invalid cycle binding kind")
		}
	}
	for i, slot := range p.Slots {
		if slot != nil {
			copied := cloneProfileConfig(*slot)
			if copied.Bindings == nil {
				copied.Bindings = make(map[string]binding)
			}
			initAxisProfiles(&copied)
			p.Slots[i] = &copied
		}
	}
	return p, nil
}

func (a *appState) loadProfiles() error {
	p := defaultProfileCollection(a.cfg)
	data, err := os.ReadFile(profilesFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			a.profiles = p
			return nil
		}
		a.profiles = p
		a.profileSaveDisabled = true // An unreadable file must not be silently overwritten.
		return err
	}
	loaded, err := readProfileCollection(data)
	if err != nil {
		// Preserve the corrupt original instead of silently overwriting it.
		a.profiles = p
		a.profileSaveDisabled = true
		return fmt.Errorf("profile data invalid; using previous settings, profiles file left untouched: %w", err)
	}
	a.profiles = loaded
	a.cfg = cloneProfileConfig(*loaded.Slots[loaded.Active-1])
	return nil
}

// Called after every normal Save Settings/autosave. The active JSON remains
// backward-compatible; the separate file stores the other four profiles.
func (a *appState) persistProfiles(active config) error {
	a.profilesMu.Lock()
	defer a.profilesMu.Unlock()
	if a.profileSaveDisabled {
		return fmt.Errorf("profile file could not be read; not overwriting it")
	}
	p := &a.profiles
	if p.Active < 1 || p.Active > profileCount {
		return fmt.Errorf("invalid active profile")
	}
	copyOfCurrent := cloneProfileConfig(active)
	p.Slots[p.Active-1] = &copyOfCurrent
	p.Version = 1
	contents, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	path := profilesFilePath()
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, contents, 0644); err != nil {
		return err
	}
	// Windows cannot replace an existing destination with os.Rename; match
	// the established configuration writer's replacement strategy.
	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, path)
}

func (a *appState) activeProfileNumber() int {
	a.profilesMu.Lock()
	defer a.profilesMu.Unlock()
	return a.profiles.Active
}

func (a *appState) refreshProfileButtons() {
	active := a.activeProfileNumber()
	for i := 0; i < profileCount; i++ {
		if h := a.controls[idProfileFirst+i]; h != 0 {
			setChecked(h, active == i+1)
		}
		if h := a.controls[idProfileIncludeFirst+i]; h != 0 {
			setChecked(h, a.profileIncluded(i))
		}
	}
}

// Both the UI and global learned cycle button switch safely in place: stop
// publishing input during the transition, neutralize the existing ViGEm
// target, and restart mapping with the newly selected profile. The virtual
// Xbox target is NOT removed, so the game keeps the same XInput connection.
func (a *appState) switchProfile(number int) {
	if number < 1 || number > profileCount || number == a.activeProfileNumber() {
		a.refreshProfileButtons()
		return
	}
	if a.mappingStarting.Load() {
		a.refreshProfileButtons()
		a.postStatus("Wait until Xbox controller initialization completes before changing profiles.")
		return
	}
	if a.profileSaveDisabled {
		a.refreshProfileButtons()
		a.postStatus("Profile switching is unavailable until the invalid profiles file is recovered.")
		return
	}
	wasActive := a.mappingActive.Load() && a.xboxReady.Load()
	// Prevent another tick from outputting a stale report after we send neutral.
	a.mappingActive.Store(false)
	a.profileSwitchMu.Lock()
	defer a.profileSwitchMu.Unlock()
	if wasActive {
		_ = a.xbox.Update(xusbReport{})
	}
	a.releaseAllKeyboardOutputs()
	a.walkingActive.Store(false)
	a.flushTuningWhenDue(true)
	a.syncConfigFromUI()
	saveConfig() // Snapshot old slot and its current bindings/calibration.

	a.cfgMu.RLock()
	current := cloneProfileConfig(a.cfg)
	a.cfgMu.RUnlock()
	a.profilesMu.Lock()
	target := a.profiles.Slots[number-1]
	var next config
	firstUse := target == nil
	if firstUse {
		next = cloneProfileConfig(current)
	} else {
		next = cloneProfileConfig(*target)
	}
	a.profiles.Active = number
	newSlot := cloneProfileConfig(next)
	a.profiles.Slots[number-1] = &newSlot
	a.profilesMu.Unlock()

	a.cfgMu.Lock()
	a.cfg = next
	a.cfgMu.Unlock()
	a.applyingProfile = true
	if normalizeInputBackend(next.InputBackend) == "WinMM" {
		a.devicesMu.Lock()
		a.devices = enumerateDevices()
		a.devicesMu.Unlock()
	} else {
		_ = a.di.Restart(a.hwnd)
	}
	a.populateCombos()
	a.applyConfigToUI()
	a.updateBindingLabels()
	if a.tuning != nil {
		a.tuning.draft = cloneAxisProfiles(next.Tuning)
		a.tuning.entries = nil // Force reloading slider values even for the same mode.
	}
	a.applyingProfile = false
	a.updateContextVisibility()
	a.refreshProfileButtons()
	saveConfig() // Persist chosen active profile for the next application launch.
	if wasActive {
		a.mappingActive.Store(true)
		a.refreshXboxUI()
	}
	if firstUse {
		a.postStatus(fmt.Sprintf("Profile %d created from the previous profile. Changes now save to this slot independently.", number))
	} else {
		if wasActive {
			a.postStatus(fmt.Sprintf("Profile %d loaded seamlessly. Control mode: %s. Walking Mode reset OFF.", number, next.ControlMode))
		} else {
			a.postStatus(fmt.Sprintf("Profile %d loaded. Control mode: %s. Start Xbox Controller when ready.", number, next.ControlMode))
		}
	}
}

// Global cycling settings intentionally live outside config and are never
// copied from one profile to another.
func (a *appState) cycleSettings() (binding, bool, int) {
	a.profilesMu.Lock()
	defer a.profilesMu.Unlock()
	p := &a.profiles
	if p.CycleBinding == nil || a.profileSaveDisabled {
		return binding{}, false, 0
	}
	// Next enabled slot after current; wrap to the first enabled slot.
	for step := 1; step <= profileCount; step++ {
		n := (p.Active - 1 + step) % profileCount
		if n < len(p.CycleEnabled) && p.CycleEnabled[n] {
			return *p.CycleBinding, true, n + 1
		}
	}
	return *p.CycleBinding, true, 0
}

func (a *appState) profileIncluded(i int) bool {
	a.profilesMu.Lock()
	defer a.profilesMu.Unlock()
	return i >= 0 && i < len(a.profiles.CycleEnabled) && a.profiles.CycleEnabled[i]
}

func (a *appState) updateProfileIncluded(i int, included bool) {
	if a.profileSaveDisabled {
		a.refreshProfileButtons()
		return
	}
	a.profilesMu.Lock()
	if i >= 0 && i < len(a.profiles.CycleEnabled) {
		a.profiles.CycleEnabled[i] = included
	}
	a.profilesMu.Unlock()
	saveConfig()
	a.refreshProfileButtons()
}

func (a *appState) updateCycleBinding(b *binding) {
	if a.profileSaveDisabled {
		return
	}
	a.profilesMu.Lock()
	if b == nil {
		a.profiles.CycleBinding = nil
	} else {
		v := *b
		a.profiles.CycleBinding = &v
	}
	a.profilesMu.Unlock()
	saveConfig()
	a.updateBindingLabels()
}

func (a *appState) refreshCycleBindingLabel() {
	a.profilesMu.Lock()
	b := a.profiles.CycleBinding
	var assigned binding
	if b != nil {
		assigned = *b
	}
	a.profilesMu.Unlock()
	label := "Not assigned"
	if b != nil {
		label = bindingDisplay(assigned)
	}
	if h := a.controls[idCycleLabel]; h != 0 {
		setText(h, label)
	}
}
