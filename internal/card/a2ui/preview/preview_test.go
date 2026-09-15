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
	for _, want := range []string{"reference_preview", "方案审批", "请检查发布范围", `data-component="Button"`, `type="radio"`, `class="FieldLabel">补充说明`, `placeholder="请输入说明"`, `class="FieldLabel">处理结果`} {
		if !strings.Contains(text, want) {
			t.Errorf("preview missing %q", want)
		}
	}
}

func TestReferencePreviewInteractionContract(t *testing.T) {
	messages, err := authoring.Compile(authoring.Spec{
		Recipe:     "approval",
		SurfaceID:  "interactive-preview",
		Title:      "交互验收",
		Body:       "填写意见并选择结果。",
		PrimaryCTA: "批准",
		Secondary:  "退回",
	})
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
	rendered := string(html)
	for _, want := range []string{
		`<textarea class="TextField" placeholder="请输入说明" data-binding-path="/form/comment">`,
		`type="radio" name="choice" value="option_1" data-binding-path="/form/choice"`,
		`<button type="button" class="ButtonControl default" data-event-name="secondary" data-surface-id="interactive-preview">`,
		`<button type="button" class="ButtonControl primary" data-event-name="primary" data-surface-id="interactive-preview">`,
		`id="interaction-result"`,
		`new CustomEvent("dws-a2ui-preview-action"`,
		`form:model.form??{}`,
		`不发送真实请求`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("interactive preview missing %q", want)
		}
	}
	for _, forbidden := range []string{" disabled", "fetch(", "XMLHttpRequest", "navigator.sendBeacon"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("interactive preview contains forbidden behavior %q", forbidden)
		}
	}
}

func TestRenderRichInformationComponents(t *testing.T) {
	messages, err := authoring.Compile(authoring.Spec{
		Recipe:     "information",
		SurfaceID:  "rich-preview",
		Title:      "季度报告",
		Body:       "## 核心结论\n\n指标符合预期。",
		ImageURL:   "https://example.com/cover.png",
		FileName:   "report.pdf",
		FileURL:    "https://example.com/report.pdf",
		DetailURL:  "https://example.com/details",
		PrimaryCTA: "查看报告",
	})
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
	rendered := string(html)
	for _, want := range []string{
		`data-component="Image"`, `alt="季度报告"`, `data-component="File"`, "report.pdf",
		`data-component="CollapsiblePanel"`, "<details", "<summary>更多信息</summary>",
		`data-component="Link"`, `href="https://example.com/details"`, `data-component="Button"`, "<h2>核心结论</h2>",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rich preview missing %q", want)
		}
	}
}

func TestRenderMarkdownStructureAndEscaping(t *testing.T) {
	messages, err := authoring.Compile(authoring.Spec{
		Recipe:    "notification",
		SurfaceID: "markdown-preview",
		Title:     "Markdown 预览",
		Body:      "### 变更摘要\n\n- 第一项\n- **第二项**\n\n<script>alert('xss')</script>\n\n[危险链接](javascript:alert(1))",
	})
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
	rendered := string(html)
	for _, want := range []string{"<h3>变更摘要</h3>", "<ul>", "<li>第一项</li>", "<strong>第二项</strong>"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered Markdown missing %q: %s", want, rendered)
		}
	}
	for _, unsafe := range []string{"<script>alert('xss')</script>", `href="javascript:`} {
		if strings.Contains(rendered, unsafe) {
			t.Fatalf("rendered Markdown contains unsafe output %q: %s", unsafe, rendered)
		}
	}
}
