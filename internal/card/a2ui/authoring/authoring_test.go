// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package authoring

import (
	"fmt"
	"strings"
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
	for _, id := range []string{"subtitle", "hero", "metrics", "metric_1", "body", "highlights", "file", "details", "details_content", "detail_link", "actions"} {
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
	if got := fmt.Sprint(byID["details"]["children"]); got != "[details_content]" {
		t.Errorf("details children = %v, want padded content wrapper", got)
	}
	if got := byID["details_content"]["padding"]; got != 12.0 {
		t.Errorf("details content padding = %v, want 12", got)
	}
	if got := byID["details_content"]["gap"]; got != 8.0 {
		t.Errorf("details content gap = %v, want 8", got)
	}
	if got := byID["file"]["previewUrl"]; got != "https://example.com/report/preview" {
		t.Errorf("file previewUrl = %v", got)
	}
	if got := byID["root"]["padding"]; got != 12.0 {
		t.Errorf("root card padding = %v, want 12", got)
	}
	if got := byID["root"]["backgroundColor"]; got != "#00FFFFFF" {
		t.Errorf("root card background = %v, want transparent", got)
	}
	if got := byID["content"]["gap"]; got != 8.0 {
		t.Errorf("top-level block gap = %v, want 8", got)
	}
	if got := byID["metric_1"]["padding"]; got != 12.0 {
		t.Errorf("surfaced block padding = %v, want 12", got)
	}
	if got := byID["highlight_1"]["gap"]; got != 8.0 {
		t.Errorf("highlight row gap = %v, want 8", got)
	}
	for _, id := range []string{"metric_1", "highlight_1"} {
		if _, ok := byID[id]["backgroundColorToken"]; ok {
			t.Errorf("%s must stay transparent unless the caller requests a surface", id)
		}
	}
}

func TestRequestedComponentGuidesAreCovered(t *testing.T) {
	for _, name := range []string{"Button", "File", "Markdown", "TextField", "Link", "Card", "Row", "Column", "Tag", "Divider", "CollapsiblePanel", "Image", "Text", "ChoicePicker"} {
		if _, ok := Guide(name); !ok {
			t.Errorf("missing guide for %s", name)
		}
	}
}

func TestSurfacePoliciesKeepCardsReadableWithoutInventingWireFields(t *testing.T) {
	policies := SurfacePolicies()
	if len(policies) != 3 {
		t.Fatalf("surface policies=%d, want 3", len(policies))
	}
	byName := make(map[string]SurfacePolicy, len(policies))
	for _, policy := range policies {
		byName[policy.Name] = policy
		if policy.MinWidth < 360 {
			t.Errorf("%s minWidth=%d, want at least 360", policy.Name, policy.MinWidth)
		}
		if policy.ObservedHostMinWidth != 320 || policy.ObservedHostMaxWidth != 640 {
			t.Errorf("%s observed host width range=%d..%d, want 320..640", policy.Name, policy.ObservedHostMinWidth, policy.ObservedHostMaxWidth)
		}
		if policy.WidthMode != "content-adaptive" || policy.NarrowViewport != "fit-available-width" {
			t.Errorf("%s has non-responsive policy: %+v", policy.Name, policy)
		}
		if policy.Enforcement != "host-surface-required" {
			t.Errorf("%s enforcement=%q", policy.Name, policy.Enforcement)
		}
	}
	if byName["notification"].MinWidth != 360 {
		t.Errorf("notification minWidth=%d, want 360", byName["notification"].MinWidth)
	}
	if byName["ai"].MinWidth != 360 {
		t.Errorf("ai minWidth=%d, want 360", byName["ai"].MinWidth)
	}
}

func TestBackgroundDefaultsToTransparentAndAllowsExplicitOverride(t *testing.T) {
	compileBackground := func(background string) any {
		t.Helper()
		messages, err := Compile(Spec{Recipe: "notification", SurfaceID: "background", Title: "通知", Body: "内容", BackgroundColor: background})
		if err != nil {
			t.Fatal(err)
		}
		components := messages[2]["updateComponents"].(map[string]any)["components"].([]any)
		return components[0].(map[string]any)["backgroundColor"]
	}
	if got := compileBackground(""); got != "#00FFFFFF" {
		t.Errorf("default background=%v, want transparent", got)
	}
	if got := compileBackground("#FFE8F3FF"); got != "#FFE8F3FF" {
		t.Errorf("explicit background=%v", got)
	}
}

func TestSixteenSemanticBlocksAreRegistered(t *testing.T) {
	if got := len(Blocks()); got != 16 {
		t.Fatalf("blocks=%d, want 16", got)
	}
}

func TestNotificationRecipesUseCompactHeaderBlock(t *testing.T) {
	for _, recipe := range Recipes() {
		if recipe.Name != "schedule" && recipe.Name != "task" {
			continue
		}
		if !containsString(recipe.Blocks, "compact-notification-header") {
			t.Errorf("%s blocks=%v, want compact-notification-header", recipe.Name, recipe.Blocks)
		}
	}
	for _, block := range Blocks() {
		if block.Name == "compact-notification-header" && !strings.Contains(block.Layout, "CardHeader") {
			t.Errorf("compact notification layout=%q, want CardHeader", block.Layout)
		}
	}
	for _, recipe := range []string{"schedule", "task"} {
		messages, err := Compile(Spec{Recipe: recipe, SurfaceID: "compact-" + recipe, Title: "待处理事项", Status: "进行中", Body: "请及时处理。"})
		if err != nil {
			t.Fatal(err)
		}
		updates := messages[2]["updateComponents"].(map[string]any)["components"].([]any)
		byID := map[string]map[string]any{}
		for _, raw := range updates {
			component := raw.(map[string]any)
			byID[fmt.Sprint(component["id"])] = component
		}
		if got := byID["header"]["component"]; got != "CardHeader" {
			t.Errorf("%s header=%v, want CardHeader", recipe, got)
		}
		if got := byID["header"]["trailing"]; got != "status" {
			t.Errorf("%s trailing=%v, want status", recipe, got)
		}
		if got := byID["root"]["padding"]; got != float64(0) && got != 0 {
			t.Errorf("%s root padding=%v, want 0", recipe, got)
		}
		if got := byID["compact_body"]["padding"]; got != 8.0 {
			t.Errorf("%s body padding=%v, want 8", recipe, got)
		}
	}
}

func TestResolveRecipeReportsTheEffectiveDefault(t *testing.T) {
	got, err := ResolveRecipe(Spec{Title: "普通通知", Body: "处理完成"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "notification" {
		t.Fatalf("recipe=%q, want notification", got)
	}
	if _, err := ResolveRecipe(Spec{Recipe: "missing"}); err == nil {
		t.Fatal("expected unknown recipe rejection")
	}
}

func TestImageTextItemBlockDeclaresCompactHorizontalContract(t *testing.T) {
	var found *Block
	for _, block := range Blocks() {
		if block.Name == "image-text-item" {
			blockCopy := block
			found = &blockCopy
			break
		}
	}
	if found == nil {
		t.Fatal("missing image-text-item block")
	}
	if found.Layout != "Row(align=center,gap=12) > Column(weight=1) + Image(variant=smallFeature)" {
		t.Errorf("layout=%q", found.Layout)
	}
	if !strings.Contains(found.Fallback, "纯文本") {
		t.Errorf("fallback=%q, want text-only degradation", found.Fallback)
	}
}

func TestApprovalAndFormRecipesHaveDistinctInformationArchitecture(t *testing.T) {
	compile := func(recipe string) map[string]map[string]any {
		t.Helper()
		messages, err := Compile(Spec{
			Recipe:     recipe,
			SurfaceID:  "distinct-" + recipe,
			Title:      "流程处理",
			Body:       "请检查内容并继续。",
			PrimaryCTA: "提交",
			Secondary:  "取消",
		})
		if err != nil {
			t.Fatal(err)
		}
		updates := messages[2]["updateComponents"].(map[string]any)["components"].([]any)
		result := make(map[string]map[string]any, len(updates))
		for _, raw := range updates {
			component := raw.(map[string]any)
			result[fmt.Sprint(component["id"])] = component
		}
		return result
	}

	approval := compile("approval")
	form := compile("form")
	if approval["decision_title"] == nil || approval["form_fields"] != nil {
		t.Fatal("approval must expose a decision flow without the form field container")
	}
	if got := approval["choice"]["label"]; got != "处理结果" {
		t.Errorf("approval choice label = %v", got)
	}
	if got := approval["comment"]["label"]; got != "审批意见（选填）" {
		t.Errorf("approval comment label = %v", got)
	}
	if got := eventName(approval["primary_button"]); got != "approval_submit" {
		t.Errorf("approval primary event = %q", got)
	}
	if got := eventName(approval["secondary_button"]); got != "approval_return" {
		t.Errorf("approval secondary event = %q", got)
	}

	if form["form_intro"] == nil || form["form_fields"] == nil || form["decision_title"] != nil {
		t.Fatal("form must expose an instruction and grouped fields without approval decision chrome")
	}
	if got := form["form_fields"]["children"]; fmt.Sprint(got) != "[comment choice]" {
		t.Errorf("form field order = %v", got)
	}
	if got := form["form_fields"]["padding"]; got != 12.0 {
		t.Errorf("form field container padding = %v", got)
	}
	if got := form["choice"]["label"]; got != "执行方式" {
		t.Errorf("form choice label = %v", got)
	}
	if got := form["secondary_button"]["variant"]; got != "borderless" {
		t.Errorf("form secondary variant = %v", got)
	}
	if got := eventName(form["primary_button"]); got != "form_submit" {
		t.Errorf("form primary event = %q", got)
	}
	if got := eventName(form["secondary_button"]); got != "form_cancel" {
		t.Errorf("form secondary event = %q", got)
	}
}

func eventName(component map[string]any) string {
	action, _ := component["action"].(map[string]any)
	event, _ := action["event"].(map[string]any)
	name, _ := event["name"].(string)
	return name
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
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
