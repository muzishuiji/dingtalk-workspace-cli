// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

// Package preview provides a deterministic reference renderer for fast local
// authoring checks. It is intentionally not a DingTalk client fidelity claim.
package preview

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/state"
)

type node struct {
	ID, Kind, Text, Variant string
	Children                []node
	Options                 []string
}

const document = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>A2UI reference preview</title><style>
:root{color-scheme:light dark;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f4f6f8;color:#171a1f}body{margin:0;padding:36px}.notice{max-width:620px;margin:0 auto 12px;color:#687078;font-size:12px}.surface{max-width:620px;margin:auto}.Card{background:#fff;border:1px solid #e7e9ed;border-radius:14px;padding:20px;box-shadow:0 8px 28px #18203312}.Column{display:flex;flex-direction:column;gap:12px}.Row{display:flex;align-items:center;gap:8px}.Row>*{min-width:0}.Text.title{font-size:18px;font-weight:650}.Text{line-height:1.5}.Markdown{line-height:1.65;white-space:pre-wrap}.Tag{display:inline-flex;width:max-content;border-radius:999px;padding:3px 9px;background:#e8f3ff;color:#126ee3;font-size:12px}.Divider{height:1px;background:#e7e9ed}.Image{width:100%;height:180px;object-fit:cover;border-radius:10px}.File,.CollapsiblePanel{padding:12px;background:#f6f7f9;border-radius:9px}.Button{flex:1;text-align:center;padding:10px 14px;border-radius:9px;background:#1677ff;color:#fff}.Button.default{background:#eef1f5;color:#24292f}.Link{color:#126ee3}.TextField{box-sizing:border-box;width:100%;padding:11px;border:1px solid #d9dde3;border-radius:8px}.ChoicePicker{padding:8px 0}.option{margin-right:14px}@media(prefers-color-scheme:dark){:root{background:#121416;color:#eef1f5}.Card{background:#1f2328;border-color:#353b42}.Divider{background:#353b42}.File,.CollapsiblePanel{background:#292e35}}
</style></head><body><div class="notice">DWS A2UI reference_preview · 用于结构、层级与交互合同快速验证，不代表钉钉客户端像素级效果</div><main class="surface">{{template "node" .}}</main>{{define "node"}}<section class="{{.Kind}} {{.Variant}}" data-component-id="{{.ID}}" data-component="{{.Kind}}">{{if eq .Kind "Image"}}<img class="Image" src="{{.Text}}" alt="">{{else if eq .Kind "TextField"}}<input class="TextField" placeholder="{{.Text}}" disabled>{{else if eq .Kind "ChoicePicker"}}<div class="ChoicePicker">{{range .Options}}<label class="option"><input type="radio" disabled> {{.}}</label>{{end}}</div>{{else}}{{if .Text}}{{.Text}}{{end}}{{range .Children}}{{template "node" .}}{{end}}{{end}}</section>{{end}}</body></html>`

func Render(surface state.Surface) ([]byte, error) {
	root, ok := surface.Components["root"]
	if !ok {
		return nil, fmt.Errorf("preview requires component id root")
	}
	view, err := buildNode("root", root, surface, map[string]bool{})
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("preview").Parse(document)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, view); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func buildNode(id string, component map[string]any, surface state.Surface, stack map[string]bool) (node, error) {
	if stack[id] {
		return node{}, fmt.Errorf("component cycle at %s", id)
	}
	stack[id] = true
	defer delete(stack, id)
	kind, _ := component["component"].(string)
	view := node{ID: id, Kind: kind}
	if variant, ok := component["variant"].(string); ok {
		view.Variant = variant
	}
	for _, field := range []string{"text", "content", "title", "fileName", "label", "placeholder", "url"} {
		if value, ok := component[field]; ok {
			if rendered := resolve(value, surface.Data); rendered != "" {
				view.Text = rendered
				break
			}
		}
	}
	if id == "title" {
		view.Variant = strings.TrimSpace(view.Variant + " title")
	}
	if kind == "Button" && view.Variant == "" {
		view.Variant = "default"
	}
	if options, ok := component["options"].([]any); ok {
		for _, item := range options {
			if option, ok := item.(map[string]any); ok {
				view.Options = append(view.Options, resolve(option["label"], surface.Data))
			}
		}
	}
	for _, childID := range refs(component) {
		child, ok := surface.Components[childID]
		if !ok {
			return node{}, fmt.Errorf("component %s references missing child %s", id, childID)
		}
		childView, err := buildNode(childID, child, surface, stack)
		if err != nil {
			return node{}, err
		}
		view.Children = append(view.Children, childView)
	}
	return view, nil
}

func refs(component map[string]any) []string {
	if child, ok := component["child"].(string); ok {
		return []string{child}
	}
	var out []string
	if children, ok := component["children"].([]any); ok {
		for _, child := range children {
			if id, ok := child.(string); ok {
				out = append(out, id)
			}
		}
	}
	return out
}

func resolve(value, data any) string {
	if text, ok := value.(string); ok {
		return text
	}
	binding, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	path, ok := binding["path"].(string)
	if !ok {
		return ""
	}
	current := data
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[part]
	}
	return fmt.Sprint(current)
}

func ComponentInventory(surface state.Surface) []string {
	set := map[string]bool{}
	for _, component := range surface.Components {
		if kind, ok := component["component"].(string); ok {
			set[kind] = true
		}
	}
	out := make([]string, 0, len(set))
	for kind := range set {
		out = append(out, kind)
	}
	sort.Strings(out)
	return out
}
