// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package authoring

import (
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/protocol"
)

func TestRecipesCompileToValidA2UI(t *testing.T) {
	registry, err := protocol.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, recipe := range Recipes() {
		t.Run(recipe.Name, func(t *testing.T) {
			messages, err := Compile(Spec{Recipe: recipe.Name, SurfaceID: "test-" + recipe.Name, Title: "发布检查", Status: "待处理", Body: "**结论**：所有检查通过。", ImageURL: "https://example.com/cover.png", FileName: "report.pdf", FileURL: "https://example.com/report.pdf", DetailURL: "https://example.com/details", PrimaryCTA: "确认", Secondary: "稍后处理"})
			if err != nil {
				t.Fatal(err)
			}
			validation := registry.Validate(messages, "create")
			if !validation.Valid {
				t.Fatalf("invalid recipe: %+v", validation.Diagnostics)
			}
		})
	}
}

func TestRequestedComponentGuidesAreCovered(t *testing.T) {
	for _, name := range []string{"Button", "File", "Markdown", "TextField", "Link", "Card", "Row", "Column", "Tag", "Divider", "CollapsiblePanel", "Image", "Text", "ChoicePicker"} {
		if _, ok := Guide(name); !ok {
			t.Errorf("missing guide for %s", name)
		}
	}
}

func TestFifteenSemanticBlocksAreRegistered(t *testing.T) {
	if got := len(Blocks()); got != 15 {
		t.Fatalf("blocks=%d, want 15", got)
	}
}

func TestRecommend(t *testing.T) {
	if got := Recommend("请生成一个审批确认卡片").Name; got != "approval" {
		t.Fatalf("got %s", got)
	}
	if got := Recommend("一般提醒").Name; got != "notification" {
		t.Fatalf("got %s", got)
	}
}
