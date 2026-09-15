// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package preview

import (
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/authoring"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/state"
)

func TestRenderReferencePreview(t *testing.T) {
	messages, err := authoring.Compile(authoring.Spec{Recipe: "approval", SurfaceID: "preview", Title: "方案审批", Status: "待确认", Body: "请检查发布范围。", PrimaryCTA: "同意", Secondary: "拒绝"})
	if err != nil {
		t.Fatal(err)
	}
	surface, err := state.Reduce(state.Surface{}, messages)
	if err != nil {
		t.Fatal(err)
	}
	html, err := Render(surface)
	if err != nil {
		t.Fatal(err)
	}
	text := string(html)
	for _, want := range []string{"reference_preview", "方案审批", "请检查发布范围", `data-component="Button"`, `type="radio"`} {
		if !strings.Contains(text, want) {
			t.Errorf("preview missing %q", want)
		}
	}
}
