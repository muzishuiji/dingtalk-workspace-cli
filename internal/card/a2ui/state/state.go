// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package state

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Surface struct {
	SurfaceID  string                    `json:"surfaceId"`
	CatalogID  string                    `json:"catalogId"`
	Components map[string]map[string]any `json:"components"`
	Data       any                       `json:"dataModel"`
}

func Reduce(base Surface, messages []map[string]any) (Surface, error) {
	base = cloneSurface(base)
	if base.Components == nil {
		base.Components = map[string]map[string]any{}
	}
	for i, message := range messages {
		operation, body, err := operation(message)
		if err != nil {
			return Surface{}, fmt.Errorf("message %d: %w", i, err)
		}
		surfaceID, _ := body["surfaceId"].(string)
		if base.SurfaceID != "" && surfaceID != base.SurfaceID {
			return Surface{}, fmt.Errorf("message %d targets surface %q, expected %q", i, surfaceID, base.SurfaceID)
		}
		if surfaceID != "" {
			base.SurfaceID = surfaceID
		}
		switch operation {
		case "createSurface":
			base.CatalogID, _ = body["catalogId"].(string)
		case "updateComponents":
			components, _ := body["components"].([]any)
			for _, item := range components {
				component, ok := item.(map[string]any)
				if !ok {
					continue
				}
				id, _ := component["id"].(string)
				if id != "" {
					base.Components[id] = cloneMap(component)
				}
			}
		case "updateDataModel":
			path, _ := body["path"].(string)
			base.Data, err = setPointer(base.Data, path, body["value"])
			if err != nil {
				return Surface{}, fmt.Errorf("message %d: %w", i, err)
			}
		case "appendDataModel":
			path, _ := body["path"].(string)
			value, _ := body["value"].(string)
			current, ok := getPointer(base.Data, path).(string)
			if !ok {
				return Surface{}, fmt.Errorf("message %d: append target %s is not a string", i, path)
			}
			base.Data, err = setPointer(base.Data, path, current+value)
			if err != nil {
				return Surface{}, fmt.Errorf("message %d: %w", i, err)
			}
		case "deleteSurface":
			return Surface{}, nil
		}
	}
	return base, nil
}

func cloneSurface(value Surface) Surface {
	raw, _ := json.Marshal(value)
	var out Surface
	_ = json.Unmarshal(raw, &out)
	return out
}

func Diff(previous, desired Surface) ([]map[string]any, error) {
	if previous.SurfaceID == "" || desired.SurfaceID != previous.SurfaceID {
		return nil, fmt.Errorf("snapshot diff requires the same non-empty surfaceId")
	}
	var messages []map[string]any
	changed := make([]any, 0)
	for id, component := range desired.Components {
		if !equalJSON(component, previous.Components[id]) {
			changed = append(changed, component)
		}
	}
	if len(changed) > 0 {
		messages = append(messages, map[string]any{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": desired.SurfaceID, "components": changed}})
	}
	if !equalJSON(previous.Data, desired.Data) {
		messages = append(messages, map[string]any{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": desired.SurfaceID, "path": "/", "value": desired.Data}})
	}
	return messages, nil
}

func operation(message map[string]any) (string, map[string]any, error) {
	for _, name := range []string{"createSurface", "updateComponents", "updateDataModel", "appendDataModel", "deleteSurface"} {
		if raw, ok := message[name]; ok {
			body, ok := raw.(map[string]any)
			if !ok {
				return "", nil, fmt.Errorf("%s body must be an object", name)
			}
			return name, body, nil
		}
	}
	return "", nil, fmt.Errorf("unsupported operation")
}

func cloneMap(value map[string]any) map[string]any {
	raw, _ := json.Marshal(value)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}
func equalJSON(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

func tokens(pointer string) ([]string, error) {
	if pointer == "/" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("invalid JSON Pointer %q", pointer)
	}
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for i := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(parts[i], "~1", "/"), "~0", "~")
	}
	return parts, nil
}

func getPointer(root any, pointer string) any {
	parts, err := tokens(pointer)
	if err != nil {
		return nil
	}
	current := root
	for _, part := range parts {
		switch typed := current.(type) {
		case map[string]any:
			current = typed[part]
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil
			}
			current = typed[index]
		default:
			return nil
		}
	}
	return current
}

func setPointer(root any, pointer string, value any) (any, error) {
	parts, err := tokens(pointer)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return value, nil
	}
	if root == nil {
		root = map[string]any{}
	}
	current, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("pointer parent is not an object")
	}
	for _, part := range parts[:len(parts)-1] {
		next, exists := current[part]
		if !exists {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("pointer parent %s is not an object", part)
		}
		current = child
	}
	current[parts[len(parts)-1]] = value
	return root, nil
}
