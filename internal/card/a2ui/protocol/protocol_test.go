// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package protocol

import (
	"strings"
	"testing"
)

func TestLoadAndValidateCreate(t *testing.T) {
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	result := registry.Validate([]map[string]any{{
		"version": "v1.0",
		"createSurface": map[string]any{
			"surfaceId": "card",
			"catalogId": registry.Catalog().CatalogID,
			"components": []any{
				map[string]any{"id": "root", "component": "Card", "child": "body"},
				map[string]any{"id": "body", "component": "Text", "text": "Hello"},
			},
		},
	}}, "create")
	if !result.Valid {
		t.Fatalf("expected valid create: %+v", result.Diagnostics)
	}
	if result.Components != 2 || result.SurfaceID != "card" {
		t.Fatalf("unexpected summary: %+v", result)
	}
}

func TestValidateRejectsRadioButtonAndMissingCatalog(t *testing.T) {
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	result := registry.Validate([]map[string]any{{
		"version": "v1.0",
		"createSurface": map[string]any{
			"surfaceId":  "card",
			"components": []any{map[string]any{"id": "choice", "component": "RadioButton"}},
		},
	}}, "create")
	if result.Valid {
		t.Fatal("expected validation failure")
	}
	codes := map[string]bool{}
	for _, diagnostic := range result.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes["A2UI_CATALOG_REQUIRED"] || !codes["A2UI_COMPONENT_UNKNOWN"] {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
}

func TestParseMessagesSupportsObjectArrayWireAndJSONL(t *testing.T) {
	inputs := []string{
		`[{"version":"v1.0","deleteSurface":{"surfaceId":"s"}}]`,
		`["{\"version\":\"v1.0\",\"deleteSurface\":{\"surfaceId\":\"s\"}}"]`,
		"{\"version\":\"v1.0\",\"deleteSurface\":{\"surfaceId\":\"s\"}}\n",
	}
	for _, input := range inputs {
		messages, err := ParseMessages(strings.NewReader(input))
		if err != nil || len(messages) != 1 {
			t.Fatalf("ParseMessages(%q) = %v, %v", input, messages, err)
		}
	}
}

func TestValidateUpdateAndAppendContracts(t *testing.T) {
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	valid := registry.Validate([]map[string]any{
		{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": "s", "path": "/stream/body", "value": "checkpoint"}},
		{"version": "v0.9.1", "appendDataModel": map[string]any{"surfaceId": "s", "path": "/stream/body", "value": " delta"}},
	}, "update")
	if !valid.Valid {
		t.Fatalf("expected valid update: %+v", valid.Diagnostics)
	}
	invalid := registry.Validate([]map[string]any{{
		"version":         "v1.0",
		"appendDataModel": map[string]any{"surfaceId": "s", "path": "/card/flowStatus", "value": "x"},
	}}, "update")
	if invalid.Valid {
		t.Fatal("expected invalid append")
	}
}

func TestValidateRejectsMissingInitialBinding(t *testing.T) {
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	result := registry.Validate([]map[string]any{
		{"version": "v1.0", "createSurface": map[string]any{"surfaceId": "s", "catalogId": registry.Catalog().CatalogID}},
		{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": "s", "path": "/", "value": map[string]any{}}},
		{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{map[string]any{"id": "root", "component": "Text", "text": map[string]any{"path": "/missing"}}}}},
	}, "create")
	if result.Valid {
		t.Fatal("expected missing binding failure")
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "A2UI_BINDING_PATH_MISSING" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}
