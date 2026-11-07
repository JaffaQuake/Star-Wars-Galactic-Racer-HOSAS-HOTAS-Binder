package main

import (
	"regexp"
	"strings"
)

// Windows gives each DirectInput controller an instance ID (a GUID). When two
// controllers are the same model, a restart can hand those IDs out the other
// way round, so every assignment points at the wrong controller. A
// controller's device path does not change: it holds the serial number, or for
// a controller without one, the USB port it is plugged into. The app saves
// each ID's path and, when a path turns up under a different ID, moves the
// assignments that used the old ID across to the new one.

var guidPattern = regexp.MustCompile(`(?i)[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}`)

// normalizeDevicePaths returns the map with upper-case IDs and lower-case
// paths, leaving out entries that are missing either.
func normalizeDevicePaths(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for id, path := range m {
		id, path = strings.ToUpper(strings.TrimSpace(id)), strings.ToLower(strings.TrimSpace(path))
		if id != "" && path != "" {
			out[id] = path
		}
	}
	return out
}

// uniquePathOwners maps each path to its ID, leaving out any path shared by
// more than one ID: those cannot tell controllers apart.
func uniquePathOwners(paths map[string]string) map[string]string {
	owners := make(map[string]string, len(paths))
	shared := map[string]bool{}
	for id, path := range paths {
		if _, seen := owners[path]; seen {
			shared[path] = true
		}
		owners[path] = id
	}
	for path := range shared {
		delete(owners, path)
	}
	return owners
}

// joystickIDChanges returns old ID → new ID for each saved controller whose
// path is now under a different ID. saved and current map ID → path; inUse
// holds the IDs this configuration refers to. A change is left out when it
// could put two roles on one controller: when its new ID already meant
// another controller (saved, or used by this configuration) that has not
// itself moved. When unsure, nothing changes.
func joystickIDChanges(saved, current map[string]string, inUse map[string]bool) map[string]string {
	saved, current = normalizeDevicePaths(saved), normalizeDevicePaths(current)
	savedOwners, currentOwners := uniquePathOwners(saved), uniquePathOwners(current)
	changes := map[string]string{}
	for path, oldID := range savedOwners {
		if newID, ok := currentOwners[path]; ok && newID != oldID {
			changes[oldID] = newID
		}
	}
	for {
		dropped := false
		for oldID, newID := range changes {
			_, newWasSaved := saved[newID]
			_, newMoved := changes[newID]
			if (newWasSaved || inUse[newID]) && !newMoved {
				delete(changes, oldID)
				dropped = true
			}
		}
		if !dropped {
			return changes
		}
	}
}

// mergeDevicePaths is the device-path list to save: every connected
// controller, plus disconnected ones whose ID and path nothing has taken over.
func mergeDevicePaths(saved, current map[string]string, changes map[string]string) map[string]string {
	saved, current = normalizeDevicePaths(saved), normalizeDevicePaths(current)
	out := make(map[string]string, len(saved)+len(current))
	connectedPaths := map[string]bool{}
	for id, path := range current {
		out[id] = path
		connectedPaths[path] = true
	}
	for id, path := range saved {
		if _, connected := current[id]; connected {
			continue
		}
		if _, moved := changes[id]; moved || connectedPaths[path] {
			continue
		}
		out[id] = path
	}
	return out
}

// guidsIn returns every DirectInput ID that appears in text, upper-case.
func guidsIn(text string) map[string]bool {
	ids := map[string]bool{}
	for _, id := range guidPattern.FindAllString(text, -1) {
		ids[strings.ToUpper(id)] = true
	}
	return ids
}

// remapGUIDs replaces every ID in text that changes maps (ignoring letter
// case) in one pass, so swapped IDs cannot undo each other.
func remapGUIDs(text string, changes map[string]string) (string, int) {
	if len(changes) == 0 {
		return text, 0
	}
	count := 0
	out := guidPattern.ReplaceAllStringFunc(text, func(id string) string {
		if newID, ok := changes[strings.ToUpper(id)]; ok {
			count++
			return newID
		}
		return id
	})
	return out, count
}
