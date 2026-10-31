package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

// keepUnknownSettings returns fresh (this build's JSON for a value of type t)
// plus every setting in old (the file being replaced) that type t does not
// know. Settings written by a newer build, or a build from another branch,
// then survive a save by this one instead of being silently dropped.
//
// It recurses into struct-valued settings (such as an Additional Input) and
// into arrays of them (profile slots), matched by position. Maps such as
// bindings and axis_tuning belong entirely to this build: merging them would
// bring back entries the user deliberately removed. Known settings that this
// build left out (omitempty) are never restored from old.
func keepUnknownSettings(fresh, old []byte, t reflect.Type) []byte {
	merged, changed := mergeUnknownJSON(fresh, old, t)
	if !changed {
		return fresh
	}
	var out bytes.Buffer
	if json.Indent(&out, merged, "", "  ") != nil {
		return fresh
	}
	return out.Bytes()
}

func mergeUnknownJSON(fresh, old json.RawMessage, t reflect.Type) (json.RawMessage, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		var f, o map[string]json.RawMessage
		if json.Unmarshal(fresh, &f) != nil || json.Unmarshal(old, &o) != nil || f == nil || o == nil {
			return fresh, false
		}
		known := jsonFieldTypes(t)
		changed := false
		for key, oldValue := range o {
			fieldType, isKnown := known[key]
			freshValue, written := f[key]
			switch {
			case !isKnown && !written:
				f[key] = oldValue
				changed = true
			case isKnown && written:
				if merged, ok := mergeUnknownJSON(freshValue, oldValue, fieldType); ok {
					f[key] = merged
					changed = true
				}
			}
		}
		if !changed {
			return fresh, false
		}
		out, err := json.Marshal(f)
		if err != nil {
			return fresh, false
		}
		return out, true
	case reflect.Array, reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return fresh, false // []byte is a JSON string
		}
		var f, o []json.RawMessage
		if json.Unmarshal(fresh, &f) != nil || json.Unmarshal(old, &o) != nil {
			return fresh, false
		}
		changed := false
		for i := range f {
			if i >= len(o) {
				break
			}
			if merged, ok := mergeUnknownJSON(f[i], o[i], t.Elem()); ok {
				f[i] = merged
				changed = true
			}
		}
		if !changed {
			return fresh, false
		}
		out, err := json.Marshal(f)
		if err != nil {
			return fresh, false
		}
		return out, true
	}
	return fresh, false
}

// jsonFieldTypes maps each JSON setting name of struct type t to its Go type.
func jsonFieldTypes(t reflect.Type) map[string]reflect.Type {
	fields := make(map[string]reflect.Type, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		switch name {
		case "-":
			continue
		case "":
			name = field.Name
		}
		fields[name] = field.Type
	}
	return fields
}
