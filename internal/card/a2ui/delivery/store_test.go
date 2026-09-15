// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package delivery

import (
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/state"
)

func TestStoreRoundTripAndExtractBizID(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	want := Record{Handle: "card-123", BizID: "biz-1", Surface: state.Surface{SurfaceID: "surface"}, FlowStatus: "PROCESSING", Revision: 1}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(want.Handle)
	if err != nil {
		t.Fatal(err)
	}
	if got.BizID != want.BizID || got.Surface.SurfaceID != "surface" {
		t.Fatalf("got %+v", got)
	}
	list, err := store.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	if got := ExtractBizID(map[string]any{"result": map[string]any{"bizId": "nested"}}); got != "nested" {
		t.Fatalf("bizId=%q", got)
	}
}
