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
		Recipe:          "information",
		SurfaceID:       "rich-preview",
		Title:           "季度报告",
		Subtitle:        "2026 Q3 · 联合复盘",
		Body:            "## 核心结论\n\n指标符合预期。",
		ImageURL:        "https://example.com/cover.png",
		ImageCaption:    "报告封面",
		Metrics:         []authoring.Metric{{Label: "完成率", Value: "96%"}, {Label: "风险", Value: "0"}},
		Highlights:      []authoring.Highlight{{Title: "核心交付", Detail: "已完成验收", Status: "已完成", Theme: "green"}},
		FileName:        "report.pdf",
		FileURL:         "https://example.com/report.pdf",
		FilePreviewURL:  "https://example.com/report/preview",
		FileDescription: "正式版",
		FileMIMEType:    "application/pdf",
		FileSize:        2488320,
		Details:         "### 数据口径\n\n按验收结果计算。",
		DetailURL:       "https://example.com/details",
		PrimaryCTA:      "查看报告",
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
		`data-component="Image"`, `class="ImagePreview"`, `aria-label="预览：季度报告"`, `alt="季度报告"`, `data-component="File"`, "report.pdf",
		`class="FileMain" href="https://example.com/report/preview"`, `data-preview-event="file_preview"`,
		`class="FileAction" href="https://example.com/report.pdf"`, "正式版 · 2.4 MB · application/pdf", `class="FileIcon">PDF`,
		`data-component-id="metrics"`, "关键指标", "96%", `data-component-id="highlights"`, "核心交付", "已完成验收",
		`data-max-line="2"`, `data-component="CollapsiblePanel"`, "<details>", "<summary>数据口径与补充说明</summary>",
		`class="PanelContent" data-max-height="240"`, "<h3>数据口径</h3>",
		`data-component="Link"`, `href="https://example.com/details"`, `data-component="Button"`, "<h2>核心结论</h2>",
		"图片暂不可用", `image.naturalWidth<=1`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rich preview missing %q", want)
		}
	}
	if strings.Contains(rendered, "<details open>") {
		t.Fatal("CollapsiblePanel must honor the protocol default and start collapsed")
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

func TestRenderRejectsUnsafeImageURL(t *testing.T) {
	messages, err := authoring.Compile(authoring.Spec{
		Recipe:    "information",
		SurfaceID: "unsafe-image-preview",
		Title:     "图片安全验收",
		Body:      "图片地址必须使用受控协议。",
		ImageURL:  "javascript:alert(1)",
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
	if strings.Contains(rendered, `src="javascript:`) || strings.Contains(rendered, `src="#ZgotmplZ`) {
		t.Fatalf("unsafe image URL reached rendered src: %s", rendered)
	}
	if !strings.Contains(rendered, `src=""`) {
		t.Fatalf("unsafe image URL must render an empty source for the fallback state")
	}
}

func TestRenderHonorsUnboundedPanelAndLargeLineLimit(t *testing.T) {
	surface := state.Surface{
		SurfaceID: "numeric-preview",
		Components: map[string]map[string]any{
			"root":        {"component": "Card", "child": "content"},
			"content":     {"component": "Column", "children": []any{"copy", "details"}},
			"copy":        {"component": "Text", "text": "长文本", "maxLine": 1200.0},
			"details":     {"component": "CollapsiblePanel", "title": "不限高详情", "maxHeight": 0.0, "children": []any{"detail_copy"}},
			"detail_copy": {"component": "Text", "text": "完整内容"},
		},
	}
	html, err := Render(surface)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(html)
	for _, want := range []string{`data-max-line="1200" style="-webkit-line-clamp:1200"`, `data-max-height="0"`} {
		if !strings.Contains(rendered, want) {
			t.Errorf("numeric preview missing %q", want)
		}
	}
	if strings.Contains(rendered, "max-height:0px") {
		t.Fatal("maxHeight 0 means unlimited and must not collapse panel content")
	}
}
