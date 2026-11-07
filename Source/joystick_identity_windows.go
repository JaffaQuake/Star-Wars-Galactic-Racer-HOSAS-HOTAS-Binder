//go:build windows

package main

import (
	"encoding/json"
	"strings"
	"syscall"
	"unsafe"
)

// diPropGUIDAndPath is DIPROPGUIDANDPATH: a controller's HID device path.
type diPropGUIDAndPath struct {
	Header    diPropHeader
	ClassGUID diGUID
	Path      [260]uint16
}

// diDevicePath reads a DirectInput controller's device path, which survives
// restarts (see joystick_identity.go), or "" if Windows does not report one.
func diDevicePath(dev uintptr) string {
	p := diPropGUIDAndPath{Header: diPropHeader{
		DwSize:       uint32(unsafe.Sizeof(diPropGUIDAndPath{})),
		DwHeaderSize: uint32(unsafe.Sizeof(diPropHeader{})),
	}}
	// IDirectInputDevice8::GetProperty is slot 5; DIPROP_GUIDANDPATH is the
	// property token 12, read for the whole device (DIPH_DEVICE, 0).
	if uint32(comCall(dev, 5, 12, uintptr(unsafe.Pointer(&p)))) != 0 {
		return ""
	}
	return strings.ToLower(syscall.UTF16ToString(p.Path[:]))
}

// DevicePaths maps each connected controller's ID to its device path.
func (m *directInputManager) DevicePaths() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.devices))
	for _, d := range m.devices {
		if d.Path != "" {
			out[d.GUIDString] = d.Path
		}
	}
	return out
}

// reconcileConfigIDs moves c's assignments to follow any controller whose ID
// Windows changed, and records the current paths. It returns the changes.
func reconcileConfigIDs(c config, current map[string]string) (config, map[string]string) {
	saved := c.DevicePaths
	data, err := json.Marshal(c)
	if err != nil {
		return c, nil
	}
	changes := joystickIDChanges(saved, current, guidsIn(string(data)))
	if len(changes) > 0 {
		text, _ := remapGUIDs(string(data), changes)
		var moved config
		if json.Unmarshal([]byte(text), &moved) == nil {
			c = moved
		} else {
			changes = nil
		}
	}
	c.DevicePaths = mergeDevicePaths(saved, current, changes)
	return c, changes
}

// reconcileJoystickIDs runs after DirectInput lists the connected
// controllers, before the UI shows any assignment. Each profile is checked
// against its own saved paths.
func (a *appState) reconcileJoystickIDs() {
	current := a.di.DevicePaths()
	if len(current) == 0 {
		return
	}
	a.cfgMu.Lock()
	cfg, changes := reconcileConfigIDs(a.cfg, current)
	a.cfg = cfg
	a.cfgMu.Unlock()
	moved := len(changes)
	a.profilesMu.Lock()
	for i, slot := range a.profiles.Slots {
		if slot == nil {
			continue
		}
		next, slotChanges := reconcileConfigIDs(*slot, current)
		a.profiles.Slots[i] = &next
		moved += len(slotChanges)
	}
	// The Cycle Profile button is global; it follows the active profile's changes.
	if b := a.profiles.CycleBinding; b != nil {
		if newID, ok := changes[strings.ToUpper(b.DIGUID)]; ok {
			b.DIGUID = newID
		}
	}
	a.profilesMu.Unlock()
	if moved > 0 {
		a.postStatus("Windows changed which ID each controller has. Your assignments now follow the same physical controllers.")
	}
	saveConfig()
}
