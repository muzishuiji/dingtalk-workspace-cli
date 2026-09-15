// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package authoring

import (
	"fmt"
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

func TestInformationRecipeBuildsAReadableHierarchy(t *testing.T) {
	messages, err := Compile(Spec{
		Recipe:          "information",
		SurfaceID:       "information-hierarchy",
		Title:           "季度执行报告",
		Subtitle:        "2026 Q3 联合复盘",
		Status:          "已完成",
		Body:            "## 核心结论\n\n目标按计划完成。",
		ImageURL:        "https://example.com/cover.jpg",
		ImageCaption:    "报告封面",
		Metrics:         []Metric{{Label: "完成率", Value: "96%"}, {Label: "风险", Value: "0"}},
		Highlights:      []Highlight{{Title: "核心能力", Detail: "完成验收", Status: "已完成", Theme: "green"}},
		FileName:        "report.pdf",
		FileURL:         "https://example.com/report.pdf",
		FilePreviewURL:  "https://example.com/report/preview",
		FileDescription: "正式版",
		FileMIMEType:    "application/pdf",
		FileSize:        1024,
		Details:         "### 数据口径\n\n按验收结果计算。",
		DetailURL:       "https://example.com/details",
		PrimaryCTA:      "查看完整报告",
	})
	if err != nil {
		t.Fatal(err)
	}
	updates := messages[2]["updateComponents"].(map[string]any)["components"].([]any)
	byID := map[string]map[string]any{}
	for _, raw := range updates {
		component := raw.(map[string]any)
		byID[fmt.Sprint(component["id"])] = component
	}
	for _, id := range []string{"subtitle", "hero", "metrics", "metric_1", "body", "highlights", "file", "details", "detail_link", "actions"} {
		if byID[id] == nil {
			t.Errorf("compiled information card is missing hierarchy component %q", id)
		}
	}
	if got := byID["status"]["theme"]; got != "green" {
		t.Errorf("completed status theme = %v, want green", got)
	}
	if got := byID["details"]["defaultExpanded"]; got != false {
		t.Errorf("details defaultExpanded = %v, want false", got)
	}
	if got := byID["file"]["previewUrl"]; got != "https://example.com/report/preview" {
		t.Errorf("file previewUrl = %v", got)
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

func TestStatusTagThemeHandlesNegativeFormsBeforeSuccessTokens(t *testing.T) {
	for _, value := range []string{"未完成", "未通过", "incomplete", "unsuccessful", "not completed", "failure"} {
		if got := statusTagTheme(value); got != "red" {
			t.Errorf("statusTagTheme(%q) = %q, want red", value, got)
		}
	}
	for _, value := range []string{"已完成", "已通过", "complete", "success"} {
		if got := statusTagTheme(value); got != "green" {
			t.Errorf("statusTagTheme(%q) = %q, want green", value, got)
		}
	}
	messages, err := Compile(Spec{Recipe: "notification", SurfaceID: "default-status", Title: "默认状态", Body: "验证默认状态。"})
	if err != nil {
		t.Fatal(err)
	}
	components := messages[2]["updateComponents"].(map[string]any)["components"].([]any)
	if got := components[4].(map[string]any)["theme"]; got != "orange" {
		t.Errorf("default status theme = %v, want orange", got)
	}
}
