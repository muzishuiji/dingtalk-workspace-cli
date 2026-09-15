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
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

type node struct {
	ID, Kind, Text, Variant                       string
	Label, Placeholder, Href, Alt                 string
	BindingPath, Value, EventName, EventSurfaceID string
	MarkdownHTML                                  template.HTML
	Children                                      []node
	Options                                       []option
}

type option struct {
	Label, Value string
	Selected     bool
}

const document = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>A2UI reference preview</title><style>
:root{color-scheme:light dark;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f4f6f8;color:#171a1f}body{margin:0;padding:36px}.notice,.interaction-result{max-width:620px;margin:0 auto 12px;color:#687078;font-size:12px}.surface{max-width:620px;margin:auto}.Card{background:#fff;border:1px solid #e7e9ed;border-radius:14px;padding:20px;box-shadow:0 8px 28px #18203312}.Column{display:flex;flex-direction:column;gap:12px}.Row{display:flex;align-items:center;gap:8px}.Row>*{min-width:0}.Text.title{font-size:18px;font-weight:650}.Text{line-height:1.5}.Markdown{line-height:1.65}.Markdown>:first-child{margin-top:0}.Markdown>:last-child{margin-bottom:0}.Markdown h1,.Markdown h2,.Markdown h3,.Markdown h4,.Markdown h5,.Markdown h6{line-height:1.35;margin:0 0 10px;font-weight:650}.Markdown h1{font-size:22px}.Markdown h2{font-size:20px}.Markdown h3{font-size:17px}.Markdown h4,.Markdown h5,.Markdown h6{font-size:15px}.Markdown p{margin:0 0 10px}.Markdown ul,.Markdown ol{margin:0 0 10px;padding-left:22px}.Markdown li+li{margin-top:4px}.Markdown a{color:#126ee3;text-decoration:none}.Markdown code{padding:2px 5px;border-radius:4px;background:#f1f3f5;font-size:.9em}.Markdown blockquote{margin:0 0 10px;padding-left:12px;border-left:3px solid #d9dde3;color:#687078}.Tag{display:inline-flex;width:max-content;border-radius:999px;padding:3px 9px;background:#e8f3ff;color:#126ee3;font-size:12px}.Divider{height:1px;background:#e7e9ed}.Image{width:100%;height:180px;object-fit:cover;border-radius:10px}.File,.CollapsiblePanel{padding:12px;background:#f6f7f9;border-radius:9px}.CollapsiblePanel summary{cursor:pointer;font-weight:600}.CollapsiblePanel[open] summary{margin-bottom:10px}.Button{flex:1}.ButtonControl{box-sizing:border-box;width:100%;padding:10px 14px;border:0;border-radius:9px;background:#1677ff;color:#fff;font:inherit;cursor:pointer}.ButtonControl.default{background:#eef1f5;color:#24292f}.ButtonControl:hover{filter:brightness(.97)}.ButtonControl:active{transform:translateY(1px)}.Link a{color:#126ee3;text-decoration:none}.FieldLabel{display:block;margin-bottom:6px;color:#4c5561;font-size:13px;font-weight:600}.TextField{box-sizing:border-box;width:100%;padding:11px;border:1px solid #d9dde3;border-radius:8px;background:#fff;color:#171a1f;font:inherit;resize:vertical}.TextField:focus{outline:2px solid #1677ff33;border-color:#1677ff}.ChoicePicker{padding:8px 0}.option{display:inline-flex;align-items:center;gap:5px;margin-right:14px;cursor:pointer}.interaction-result{box-sizing:border-box;margin-top:14px;padding:12px;border:1px solid #d9dde3;border-radius:9px;background:#fff;color:#171a1f}.interaction-result pre{margin:8px 0 0;white-space:pre-wrap;word-break:break-word}.interaction-result[hidden]{display:none}@media(prefers-color-scheme:dark){:root{background:#121416;color:#eef1f5}.Card,.interaction-result{background:#1f2328;border-color:#353b42;color:#eef1f5}.Divider{background:#353b42}.File,.CollapsiblePanel{background:#292e35}.Markdown code{background:#292e35}.FieldLabel{color:#c4cad1}.TextField{background:#171a1f;border-color:#4c5561;color:#eef1f5}.ButtonControl.default{background:#353b42;color:#eef1f5}}
</style></head><body><div class="notice">DWS A2UI reference_preview · 可本地编辑表单并模拟 Action，不发送真实请求；不代表钉钉客户端像素级效果</div><main class="surface">{{template "node" .}}</main><aside id="interaction-result" class="interaction-result" hidden><strong>本地 Action 模拟结果</strong><pre id="interaction-json"></pre></aside>{{define "node"}}<section class="{{.Kind}} {{.Variant}}" data-component-id="{{.ID}}" data-component="{{.Kind}}">{{if eq .Kind "Image"}}<img class="Image" src="{{.Text}}" alt="{{.Alt}}">{{else if eq .Kind "TextField"}}<label>{{if .Label}}<span class="FieldLabel">{{.Label}}</span>{{end}}{{if eq .Variant "longText"}}<textarea class="TextField" placeholder="{{.Placeholder}}" data-binding-path="{{.BindingPath}}">{{.Value}}</textarea>{{else}}<input class="TextField" placeholder="{{.Placeholder}}" value="{{.Value}}" data-binding-path="{{.BindingPath}}">{{end}}</label>{{else if eq .Kind "ChoicePicker"}}<div class="ChoicePicker">{{if .Label}}<div class="FieldLabel">{{.Label}}</div>{{end}}{{range .Options}}<label class="option"><input type="radio" name="{{$.ID}}" value="{{.Value}}" data-binding-path="{{$.BindingPath}}" {{if .Selected}}checked{{end}}> {{.Label}}</label>{{end}}</div>{{else if eq .Kind "Markdown"}}{{.MarkdownHTML}}{{else if eq .Kind "Link"}}{{if .Href}}<a href="{{.Href}}" target="_blank" rel="noopener noreferrer">{{.Text}}</a>{{else}}{{.Text}}{{end}}{{else if eq .Kind "CollapsiblePanel"}}<details open><summary>{{.Text}}</summary>{{range .Children}}{{template "node" .}}{{end}}</details>{{else if eq .Kind "Button"}}<button type="button" class="ButtonControl {{.Variant}}" data-event-name="{{.EventName}}" data-surface-id="{{.EventSurfaceID}}">{{if .Text}}{{.Text}}{{end}}{{range .Children}}{{template "node" .}}{{end}}</button>{{else}}{{if .Text}}{{.Text}}{{end}}{{range .Children}}{{template "node" .}}{{end}}{{end}}</section>{{end}}<script>
(()=>{const assign=(target,path,value)=>{const parts=path.replace(/^\//,"").split("/").filter(Boolean);let current=target;parts.forEach((part,index)=>{if(index===parts.length-1){current[part]=value;return}current[part]??={};current=current[part]})};const collect=()=>{const model={};document.querySelectorAll("[data-binding-path]").forEach(control=>{if((control.type==="radio"||control.type==="checkbox")&&!control.checked)return;assign(model,control.dataset.bindingPath,control.value)});return model};document.querySelectorAll("button[data-event-name]").forEach(button=>button.addEventListener("click",()=>{const model=collect();const payload={event:{name:button.dataset.eventName,context:{surfaceId:button.dataset.surfaceId,form:model.form??{}}},localPreview:true};document.getElementById("interaction-json").textContent=JSON.stringify(payload,null,2);document.getElementById("interaction-result").hidden=false;document.dispatchEvent(new CustomEvent("dws-a2ui-preview-action",{detail:payload}))}))})();
</script></body></html>`

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
	if kind == "Markdown" {
		rendered, err := renderMarkdown(view.Text)
		if err != nil {
			return node{}, fmt.Errorf("render Markdown component %s: %w", id, err)
		}
		view.MarkdownHTML = rendered
		view.Text = ""
	}
	if kind == "TextField" {
		view.Label = resolve(component["label"], surface.Data)
		view.Placeholder = resolve(component["placeholder"], surface.Data)
		view.BindingPath = dataBindingPath(component["value"])
		view.Value = resolve(component["value"], surface.Data)
		view.Text = ""
	}
	if kind == "ChoicePicker" {
		view.Label = resolve(component["label"], surface.Data)
		view.BindingPath = dataBindingPath(component["value"])
		view.Value = resolve(component["value"], surface.Data)
	}
	if kind == "Link" {
		view.Href = actionURL(component)
	}
	if kind == "Image" {
		if accessibility, ok := component["accessibility"].(map[string]any); ok {
			view.Alt = resolve(accessibility["label"], surface.Data)
		}
	}
	if id == "title" {
		view.Variant = strings.TrimSpace(view.Variant + " title")
	}
	if kind == "Button" && view.Variant == "" {
		view.Variant = "default"
	}
	if kind == "Button" {
		view.EventName, view.EventSurfaceID = eventInfo(component)
	}
	if options, ok := component["options"].([]any); ok {
		for _, item := range options {
			if item, ok := item.(map[string]any); ok {
				value := resolve(item["value"], surface.Data)
				view.Options = append(view.Options, option{
					Label:    resolve(item["label"], surface.Data),
					Value:    value,
					Selected: value == view.Value,
				})
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

func dataBindingPath(value any) string {
	binding, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	path, _ := binding["path"].(string)
	return path
}

func eventInfo(component map[string]any) (string, string) {
	action, ok := component["action"].(map[string]any)
	if !ok {
		return "", ""
	}
	event, ok := action["event"].(map[string]any)
	if !ok {
		return "", ""
	}
	name, _ := event["name"].(string)
	context, _ := event["context"].(map[string]any)
	surfaceID, _ := context["surfaceId"].(string)
	return name, surfaceID
}

func actionURL(component map[string]any) string {
	action, ok := component["action"].(map[string]any)
	if !ok {
		return ""
	}
	call, ok := action["functionCall"].(map[string]any)
	if !ok {
		return ""
	}
	args, ok := call["args"].(map[string]any)
	if !ok {
		return ""
	}
	url, _ := args["url"].(string)
	return url
}

func renderMarkdown(source string) (template.HTML, error) {
	var out bytes.Buffer
	// Goldmark's default HTML renderer escapes raw HTML and rejects dangerous
	// link schemes. Marking that renderer output as template.HTML prevents a
	// second escaping pass without trusting the original A2UI Markdown string.
	if err := goldmark.New(goldmark.WithExtensions(extension.GFM)).Convert([]byte(source), &out); err != nil {
		return "", err
	}
	return template.HTML(out.String()), nil // #nosec G203 -- output comes from the safe Goldmark renderer.
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
