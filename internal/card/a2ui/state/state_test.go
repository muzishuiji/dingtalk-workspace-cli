// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package state

import "testing"

func TestReduceAppendAndDiff(t *testing.T) {
	create := []map[string]any{
		{"version": "v1.0", "createSurface": map[string]any{"surfaceId": "s", "catalogId": "catalog"}},
		{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": "s", "path": "/", "value": map[string]any{"stream": map[string]any{"body": "hello"}}}},
		{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{map[string]any{"id": "root", "component": "Text", "text": map[string]any{"path": "/stream/body"}}}}},
	}
	base, err := Reduce(Surface{}, create)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := Reduce(base, []map[string]any{{"version": "v0.9.1", "appendDataModel": map[string]any{"surfaceId": "s", "path": "/stream/body", "value": " world"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := getPointer(updated.Data, "/stream/body"); got != "hello world" {
		t.Fatalf("got %#v", got)
	}
	messages, err := Diff(base, updated)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("diff messages = %d", len(messages))
	}
	replayed, err := Reduce(base, messages)
	if err != nil {
		t.Fatal(err)
	}
	if !equalJSON(replayed, updated) {
		t.Fatalf("replay mismatch: %#v != %#v", replayed, updated)
	}
}

func TestReduceRejectsCrossSurfaceAndNonStringAppend(t *testing.T) {
	base := Surface{SurfaceID: "a", Data: map[string]any{"value": 1.0}}
	if _, err := Reduce(base, []map[string]any{{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": "b", "path": "/", "value": map[string]any{}}}}); err == nil {
		t.Fatal("expected surface mismatch")
	}
	if _, err := Reduce(base, []map[string]any{{"version": "v0.9.1", "appendDataModel": map[string]any{"surfaceId": "a", "path": "/value", "value": "x"}}}); err == nil {
		t.Fatal("expected append type error")
	}
}
