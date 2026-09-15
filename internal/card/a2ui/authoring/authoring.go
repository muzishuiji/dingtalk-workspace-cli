// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

// Package authoring converts a small semantic intent into stable A2UI messages.
// It deliberately does not mirror the A2UI component tree: Agents choose an
// information structure here and inspect the generated protocol separately.
package authoring

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/protocol"
)

type Recipe struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	UseWhen     []string `json:"useWhen"`
	Blocks      []string `json:"blocks"`
}

type ComponentGuide struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	UseWhen     string `json:"useWhen"`
	AvoidWhen   string `json:"avoidWhen,omitempty"`
	Reliability string `json:"reliability,omitempty"`
}

type Block struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Components  []string `json:"components"`
}

type Metric struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Highlight struct {
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Status string `json:"status,omitempty"`
	Theme  string `json:"theme,omitempty"`
}

type Spec struct {
	Recipe          string      `json:"recipe"`
	SurfaceID       string      `json:"surfaceId"`
	Title           string      `json:"title"`
	Subtitle        string      `json:"subtitle,omitempty"`
	Status          string      `json:"status,omitempty"`
	Body            string      `json:"body"`
	ImageURL        string      `json:"imageUrl,omitempty"`
	ImageAlt        string      `json:"imageAlt,omitempty"`
	ImageCaption    string      `json:"imageCaption,omitempty"`
	Metrics         []Metric    `json:"metrics,omitempty"`
	Highlights      []Highlight `json:"highlights,omitempty"`
	FileName        string      `json:"fileName,omitempty"`
	FileURL         string      `json:"fileUrl,omitempty"`
	FilePreviewURL  string      `json:"filePreviewUrl,omitempty"`
	FileDescription string      `json:"fileDescription,omitempty"`
	FileMIMEType    string      `json:"fileMimeType,omitempty"`
	FileSize        float64     `json:"fileSize,omitempty"`
	Details         string      `json:"details,omitempty"`
	DetailURL       string      `json:"detailUrl,omitempty"`
	PrimaryCTA      string      `json:"primaryCta,omitempty"`
	Secondary       string      `json:"secondaryCta,omitempty"`
	Options         []string    `json:"options,omitempty"`
}

var recipes = []Recipe{
	{Name: "notification", Description: "一件事、一个状态、一个主要动作的通知", UseWhen: []string{"结果通知", "风险告警", "轻量提醒"}, Blocks: []string{"header", "summary", "actions"}},
	{Name: "information", Description: "有摘要、媒体、附件和详情入口的信息卡", UseWhen: []string{"信息摘要", "内容推荐", "结果报告"}, Blocks: []string{"header", "hero", "summary", "attachments", "details", "actions"}},
	{Name: "approval", Description: "带意见、单选和双动作的可靠审批卡", UseWhen: []string{"审批确认", "需要收集结构化输入"}, Blocks: []string{"header", "facts", "form", "actions", "terminal-status"}},
	{Name: "task", Description: "任务状态、说明和处理动作", UseWhen: []string{"任务分派", "待办跟进"}, Blocks: []string{"header", "facts", "summary", "actions"}},
	{Name: "schedule", Description: "日程冲突或会议提醒", UseWhen: []string{"日程提醒", "冲突处理"}, Blocks: []string{"header", "facts", "summary", "actions"}},
	{Name: "report", Description: "层次化结论、明细和附件", UseWhen: []string{"分析报告", "执行结果"}, Blocks: []string{"header", "summary", "details", "attachments", "actions"}},
	{Name: "form", Description: "收集文本和互斥选择后提交", UseWhen: []string{"澄清信息", "用户配置"}, Blocks: []string{"header", "form", "actions"}},
}

var guides = map[string]ComponentGuide{
	"Button":           {Name: "Button", Role: "明确触发一次业务动作", UseWhen: "主要或次要操作", Reliability: "event.name 必须稳定；提交表单时 context 带回绑定值"},
	"File":             {Name: "File", Role: "展示可预览或下载的附件", UseWhen: "结果含文件实体"},
	"Markdown":         {Name: "Markdown", Role: "表达多段、有层次的正文", UseWhen: "摘要、列表、说明", AvoidWhen: "短标签或按钮文案"},
	"TextField":        {Name: "TextField", Role: "收集单行或多行文本", UseWhen: "审批意见、补充说明", Reliability: "绑定 DataModel；由提交按钮统一校验和回传"},
	"Link":             {Name: "Link", Role: "低强调导航或事件入口", UseWhen: "查看详情、帮助"},
	"Card":             {Name: "Card", Role: "提供视觉表面与内边距", UseWhen: "需要独立信息面"},
	"Row":              {Name: "Row", Role: "横向并列少量同级元素", UseWhen: "标签行、双按钮", AvoidWhen: "超过两列或长正文"},
	"Column":           {Name: "Column", Role: "组织主要纵向阅读顺序", UseWhen: "绝大多数卡片根布局"},
	"Tag":              {Name: "Tag", Role: "短状态或分类", UseWhen: "风险、进度、来源", AvoidWhen: "句子或主要动作"},
	"Divider":          {Name: "Divider", Role: "分隔语义区块", UseWhen: "正文与操作区边界"},
	"CollapsiblePanel": {Name: "CollapsiblePanel", Role: "收纳低频明细", UseWhen: "日志、详情、补充说明"},
	"Image":            {Name: "Image", Role: "强化人物、封面或关键视觉", UseWhen: "图片本身携带信息"},
	"Text":             {Name: "Text", Role: "标题、标签和短正文", UseWhen: "一到两行文本"},
	"ChoicePicker":     {Name: "ChoicePicker", Role: "单选或多选结构化输入", UseWhen: "固定选项", Reliability: "RadioButton 语义编译为 mutuallyExclusive + checkbox"},
}

var blocks = []Block{
	{Name: "header", Description: "标题、来源与状态入口", Components: []string{"Row", "Text", "Tag"}},
	{Name: "hero", Description: "关键封面或人物视觉", Components: []string{"Image"}},
	{Name: "status", Description: "当前阶段或风险状态", Components: []string{"Tag", "Text"}},
	{Name: "facts", Description: "短标签与值的事实表", Components: []string{"Column", "Row", "Text"}},
	{Name: "summary", Description: "结论优先的正文摘要", Components: []string{"Markdown"}},
	{Name: "metrics", Description: "少量关键指标", Components: []string{"Row", "Column", "Text"}},
	{Name: "list", Description: "同构条目列表", Components: []string{"Column", "Card", "Text"}},
	{Name: "timeline", Description: "按时间或阶段组织的进展", Components: []string{"Column", "Row", "Text", "Tag"}},
	{Name: "media", Description: "携带业务信息的图片", Components: []string{"Image", "Text"}},
	{Name: "attachments", Description: "文件结果和附件", Components: []string{"File"}},
	{Name: "form", Description: "文本与选择输入", Components: []string{"TextField", "ChoicePicker"}},
	{Name: "actions", Description: "一个主动作及可选次动作", Components: []string{"Row", "Button", "Text"}},
	{Name: "details", Description: "低频补充信息", Components: []string{"CollapsiblePanel", "Markdown", "Link"}},
	{Name: "progress", Description: "生成或执行中的阶段提示", Components: []string{"Tag", "Text"}},
	{Name: "terminal-status", Description: "完成、失败或取消后的稳定终态", Components: []string{"Tag", "Text", "Link"}},
}

func Recipes() []Recipe { return append([]Recipe(nil), recipes...) }

func Blocks() []Block { return append([]Block(nil), blocks...) }

func Guides() []ComponentGuide {
	out := make([]ComponentGuide, 0, len(guides))
	for _, guide := range guides {
		out = append(out, guide)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func Guide(name string) (ComponentGuide, bool) { guide, ok := guides[name]; return guide, ok }

func Recommend(intent string) Recipe {
	lower := strings.ToLower(intent)
	for _, candidate := range []string{"approval", "schedule", "task", "form", "report", "information"} {
		if strings.Contains(lower, candidate) || strings.Contains(intent, map[string]string{"approval": "审批", "schedule": "日程", "task": "任务", "form": "收集", "report": "报告", "information": "信息"}[candidate]) {
			for _, recipe := range recipes {
				if recipe.Name == candidate {
					return recipe
				}
			}
		}
	}
	return recipes[0]
}

func Compile(spec Spec) ([]map[string]any, error) {
	if strings.TrimSpace(spec.SurfaceID) == "" || strings.TrimSpace(spec.Title) == "" || strings.TrimSpace(spec.Body) == "" {
		return nil, fmt.Errorf("surfaceId, title and body are required")
	}
	if spec.Recipe == "" {
		spec.Recipe = Recommend(spec.Title + " " + spec.Body).Name
	}
	known := false
	for _, recipe := range recipes {
		if recipe.Name == spec.Recipe {
			known = true
			break
		}
	}
	if !known {
		return nil, fmt.Errorf("unknown recipe %q", spec.Recipe)
	}
	registry, err := protocol.Load()
	if err != nil {
		return nil, err
	}
	status := defaultString(spec.Status, "进行中")
	children := []string{"header"}
	components := []any{
		map[string]any{"id": "root", "component": "Card", "child": "content", "padding": 16.0, "cornerRadius": 12.0},
		map[string]any{"id": "content", "component": "Column", "children": children, "align": "stretch", "gap": 12.0},
		map[string]any{"id": "header", "component": "Row", "children": []string{"title", "status"}, "align": "center", "gap": 8.0},
		map[string]any{"id": "title", "component": "Text", "text": map[string]any{"path": "/content/title"}, "variant": "body", "bold": true, "maxLine": 2.0, "weight": 1.0},
		map[string]any{"id": "status", "component": "Tag", "text": map[string]any{"path": "/content/status"}, "theme": statusTagTheme(status)},
	}
	data := map[string]any{"content": map[string]any{"title": spec.Title, "subtitle": spec.Subtitle, "status": status, "body": spec.Body}, "form": map[string]any{}}
	if spec.Subtitle != "" {
		children = append(children, "subtitle")
		components = append(components, map[string]any{"id": "subtitle", "component": "Text", "text": map[string]any{"path": "/content/subtitle"}, "variant": "caption", "color": "gray", "maxLine": 3.0})
	}
	children = append(children, "divider_top")
	components = append(components, map[string]any{"id": "divider_top", "component": "Divider", "axis": "horizontal", "borderColorToken": "common_line_light_color"})
	if spec.ImageURL != "" {
		children = append(children, "hero")
		components = append(components, map[string]any{"id": "hero", "component": "Image", "url": spec.ImageURL, "fit": "cover", "variant": "header", "cornerRadius": 10.0, "previewEnabled": true, "accessibility": map[string]any{"label": defaultString(spec.ImageAlt, spec.Title)}})
		if spec.ImageCaption != "" {
			children = append(children, "image_caption")
			components = append(components, map[string]any{"id": "image_caption", "component": "Text", "text": spec.ImageCaption, "variant": "caption", "color": "gray", "maxLine": 3.0})
		}
	}
	if len(spec.Metrics) > 0 {
		children = append(children, "metrics_title", "metrics")
		metricChildren := make([]string, 0, len(spec.Metrics))
		components = append(components, map[string]any{"id": "metrics_title", "component": "Text", "text": "关键指标", "variant": "body", "bold": true})
		for i, metric := range spec.Metrics {
			metricID := fmt.Sprintf("metric_%d", i+1)
			valueID := fmt.Sprintf("metric_value_%d", i+1)
			labelID := fmt.Sprintf("metric_label_%d", i+1)
			metricChildren = append(metricChildren, metricID)
			components = append(components,
				map[string]any{"id": metricID, "component": "Column", "children": []string{valueID, labelID}, "gap": 2.0, "padding": 10.0, "cornerRadius": 8.0, "backgroundColorToken": "common_fg_z1_color", "weight": 1.0},
				map[string]any{"id": valueID, "component": "Text", "text": metric.Value, "variant": "body", "bold": true, "maxLine": 2.0},
				map[string]any{"id": labelID, "component": "Text", "text": metric.Label, "variant": "caption", "color": "gray", "maxLine": 2.0},
			)
		}
		components = append(components, map[string]any{"id": "metrics", "component": "Row", "children": metricChildren, "align": "start", "gap": 8.0})
	}
	children = append(children, "body")
	components = append(components, map[string]any{"id": "body", "component": "Markdown", "content": map[string]any{"path": "/content/body"}})
	if len(spec.Highlights) > 0 {
		children = append(children, "highlights_title", "highlights")
		highlightChildren := make([]string, 0, len(spec.Highlights))
		components = append(components, map[string]any{"id": "highlights_title", "component": "Text", "text": "重点进展", "variant": "body", "bold": true})
		for i, highlight := range spec.Highlights {
			rowID := fmt.Sprintf("highlight_%d", i+1)
			copyID := fmt.Sprintf("highlight_copy_%d", i+1)
			titleID := fmt.Sprintf("highlight_title_%d", i+1)
			detailID := fmt.Sprintf("highlight_detail_%d", i+1)
			tagID := fmt.Sprintf("highlight_status_%d", i+1)
			copyChildren := []string{titleID}
			highlightChildren = append(highlightChildren, rowID)
			components = append(components, map[string]any{"id": titleID, "component": "Text", "text": highlight.Title, "variant": "body", "bold": true, "maxLine": 2.0})
			if highlight.Detail != "" {
				copyChildren = append(copyChildren, detailID)
				components = append(components, map[string]any{"id": detailID, "component": "Text", "text": highlight.Detail, "variant": "caption", "color": "gray", "maxLine": 3.0})
			}
			components = append(components, map[string]any{"id": copyID, "component": "Column", "children": copyChildren, "gap": 2.0, "weight": 1.0})
			rowChildren := []string{copyID}
			if highlight.Status != "" {
				rowChildren = append(rowChildren, tagID)
				components = append(components, map[string]any{"id": tagID, "component": "Tag", "text": highlight.Status, "theme": validTagTheme(highlight.Theme), "variant": "filled"})
			}
			components = append(components, map[string]any{"id": rowID, "component": "Row", "children": rowChildren, "align": "center", "gap": 10.0, "padding": 10.0, "cornerRadius": 8.0, "backgroundColorToken": "common_fg_z1_color"})
		}
		components = append(components, map[string]any{"id": "highlights", "component": "Column", "children": highlightChildren, "gap": 8.0})
	}
	if spec.FileURL != "" {
		children = append(children, "divider_attachment", "attachment_title", "file")
		file := map[string]any{"id": "file", "component": "File", "fileName": defaultString(spec.FileName, "附件"), "url": spec.FileURL, "description": defaultString(spec.FileDescription, "相关附件"), "onPreview": eventAction("file_preview", spec.SurfaceID)}
		if spec.FilePreviewURL != "" {
			file["previewUrl"] = spec.FilePreviewURL
		}
		if spec.FileMIMEType != "" {
			file["mimeType"] = spec.FileMIMEType
		}
		if spec.FileSize > 0 {
			file["size"] = spec.FileSize
		}
		components = append(components,
			map[string]any{"id": "divider_attachment", "component": "Divider", "axis": "horizontal", "borderColorToken": "common_line_light_color"},
			map[string]any{"id": "attachment_title", "component": "Text", "text": "交付产物", "variant": "body", "bold": true},
			file,
		)
	}
	if spec.DetailURL != "" || spec.Details != "" {
		children = append(children, "details")
		detailChildren := []string{}
		if spec.Details != "" {
			detailChildren = append(detailChildren, "details_body")
			components = append(components, map[string]any{"id": "details_body", "component": "Markdown", "content": spec.Details})
		}
		if spec.DetailURL != "" {
			detailChildren = append(detailChildren, "detail_link")
			components = append(components, map[string]any{"id": "detail_link", "component": "Link", "text": "打开完整详情", "action": map[string]any{"functionCall": map[string]any{"call": "openUrl", "args": map[string]any{"url": spec.DetailURL}}}})
		}
		components = append(components,
			map[string]any{"id": "details", "component": "CollapsiblePanel", "title": "数据口径与补充说明", "variant": "indented", "defaultExpanded": false, "maxHeight": 240.0, "children": detailChildren, "fallbackMarkdown": "### 数据口径与补充说明\n\n请打开完整详情查看。"},
		)
	}
	if spec.Recipe == "approval" || spec.Recipe == "form" {
		children = append(children, "comment", "choice")
		options := spec.Options
		if len(options) == 0 {
			options = []string{"同意", "拒绝"}
		}
		optionValues := make([]any, 0, len(options))
		for i, option := range options {
			optionValues = append(optionValues, map[string]any{"label": option, "value": fmt.Sprintf("option_%d", i+1)})
		}
		components = append(components,
			map[string]any{"id": "comment", "component": "TextField", "label": "补充说明", "value": map[string]any{"path": "/form/comment"}, "placeholder": "请输入说明", "variant": "longText"},
			map[string]any{"id": "choice", "component": "ChoicePicker", "label": "处理结果", "variant": "mutuallyExclusive", "displayStyle": "checkbox", "options": optionValues, "value": map[string]any{"path": "/form/choice"}},
		)
		data["form"] = map[string]any{"comment": "", "choice": ""}
	}
	if spec.PrimaryCTA != "" || spec.Secondary != "" {
		children = append(children, "divider_actions", "actions")
		actionChildren := []string{}
		components = append(components, map[string]any{"id": "divider_actions", "component": "Divider", "axis": "horizontal"})
		if spec.Secondary != "" {
			actionChildren = append(actionChildren, "secondary_button")
			components = append(components,
				map[string]any{"id": "secondary_label", "component": "Text", "text": spec.Secondary, "variant": "body"},
				map[string]any{"id": "secondary_button", "component": "Button", "child": "secondary_label", "variant": "default", "weight": 1.0, "action": eventAction("secondary", spec.SurfaceID)},
			)
		}
		if spec.PrimaryCTA != "" {
			actionChildren = append(actionChildren, "primary_button")
			components = append(components,
				map[string]any{"id": "primary_label", "component": "Text", "text": spec.PrimaryCTA, "variant": "body"},
				map[string]any{"id": "primary_button", "component": "Button", "child": "primary_label", "variant": "primary", "weight": 1.0, "action": eventAction("primary", spec.SurfaceID)},
			)
		}
		components = append(components, map[string]any{"id": "actions", "component": "Row", "children": actionChildren, "align": "center", "gap": 8.0})
	}
	components[1].(map[string]any)["children"] = children
	messages := []map[string]any{
		{"version": "v1.0", "createSurface": map[string]any{"surfaceId": spec.SurfaceID, "catalogId": registry.Catalog().CatalogID, "sendDataModel": true}},
		{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": spec.SurfaceID, "path": "/", "value": data}},
		{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": spec.SurfaceID, "components": components}},
	}
	validation := registry.Validate(messages, "create")
	if !validation.Valid {
		return nil, fmt.Errorf("generated recipe failed validation: %+v", validation.Diagnostics)
	}
	return messages, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func validTagTheme(value string) string {
	switch value {
	case "black", "gray", "red", "orange", "green", "blue":
		return value
	default:
		return "blue"
	}
}

func statusTagTheme(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch {
	case containsAny(lower, "incomplete", "unsuccessful", "not complete", "not completed", "not successful", "failed", "failure", "error"),
		containsAny(value, "未完成", "未通过", "不通过", "失败", "异常", "风险"):
		return "red"
	case containsAny(lower, "complete", "success", "passed"), containsAny(value, "已完成", "成功", "已通过"):
		return "green"
	case containsAny(lower, "pending", "waiting", "in progress"), containsAny(value, "待", "进行"):
		return "orange"
	default:
		return "blue"
	}
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func eventAction(name, surfaceID string) map[string]any {
	return map[string]any{"event": map[string]any{"name": name, "context": map[string]any{"surfaceId": surfaceID, "form": map[string]any{"path": "/form"}}}}
}
