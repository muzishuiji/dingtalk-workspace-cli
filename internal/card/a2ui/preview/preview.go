// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

// Package preview provides a deterministic reference renderer for fast local
// authoring checks. It is intentionally not a DingTalk client fidelity claim.
package preview

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/state"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

type node struct {
	ID, Kind, Text, Variant                       string
	Label, Placeholder, Href, Alt                 string
	BindingPath, Value, EventName, EventSurfaceID string
	Description, MimeType, FileSize, FileType     string
	Theme, Color, Align, Justify, InputType       string
	Gap, Padding, Radius, Weight, BackgroundToken string
	MaxLine, MaxHeight, DownloadHref              string
	PreviewEventName, PreviewEventSurfaceID       string
	DefaultExpanded, Disabled, PreviewEnabled     bool
	ImageSrc                                      template.URL
	LineStyle, PanelStyle                         template.CSS
	MarkdownHTML                                  template.HTML
	Children                                      []node
	Options                                       []option
}

type option struct {
	Label, Value string
	Selected     bool
}

const document = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>A2UI reference preview</title><style>
:root{color-scheme:light dark;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif;background:#f3f5f8;color:#171a1f}*{box-sizing:border-box}body{margin:0;padding:28px 18px 56px}.notice,.interaction-result{max-width:600px;margin:0 auto 12px;color:#687078;font-size:12px;line-height:1.5}.surface{max-width:600px;margin:auto}.Card{background:#fff;border:1px solid #e3e7ec;border-radius:16px;padding:20px;box-shadow:0 10px 32px #18203312}.Column{display:flex;min-width:0;flex-direction:column;gap:12px}.Row{display:flex;min-width:0;align-items:center;gap:8px}.Row>*{min-width:0}.Row[data-align="start"]{align-items:flex-start}.Row[data-align="end"]{align-items:flex-end}.Row[data-align="stretch"]{align-items:stretch}.Row[data-justify="center"]{justify-content:center}.Row[data-justify="end"]{justify-content:flex-end}.Column[data-align="center"]{align-items:center}.Column[data-align="end"]{align-items:flex-end}.Column[data-align="stretch"]{align-items:stretch}[data-gap="2"]{gap:2px}[data-gap="8"]{gap:8px}[data-gap="10"]{gap:10px}[data-gap="12"]{gap:12px}[data-weight="1"]{flex:1 1 0}[data-padding="10"]{padding:10px}[data-radius="8"]{border-radius:8px}.token-common_fg_z1_color{background:#f5f7fa}.Text{line-height:1.45;overflow-wrap:anywhere}.Text[data-max-line]{display:-webkit-box;overflow:hidden;-webkit-box-orient:vertical}.Text.title{font-size:20px;font-weight:680;letter-spacing:-.01em}.Text.bold{font-weight:650}.Text.caption{font-size:13px}.Text.gray{color:#687078}.Tag{display:inline-flex;width:max-content;max-width:100%;align-items:center;border-radius:999px;padding:3px 9px;font-size:12px;font-weight:550;white-space:nowrap}.Tag.blue{background:#e8f3ff;color:#075fc1}.Tag.green{background:#e8f8ef;color:#0b6b38}.Tag.orange{background:#fff3e5;color:#934900}.Tag.red{background:#ffebeb;color:#b22222}.Tag.gray,.Tag.black{background:#eef1f4;color:#4c5561}.Tag.hollow{border:1px solid currentColor;background:transparent}.Divider{height:1px;margin:1px 0;background:#e4e7eb}.ImagePreview{display:block;border-radius:10px;outline-offset:2px}.ImagePreview:focus-visible{outline:3px solid #1677ff55}.ImageFrame{position:relative;width:100%;height:200px;margin:0;overflow:hidden;border-radius:10px;background:#eef2f6}.Image{display:block;width:100%;height:100%;object-fit:cover}.ImageFrame.contain .Image{object-fit:contain}.ImageFrame[data-load="error"]{height:112px}.ImageFrame[data-load="error"] .Image{display:none}.ImageFallback{height:100%;padding:20px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:4px;color:#687078;text-align:center}.ImageFallback strong{color:#343b44;font-size:14px}.ImageFallback[hidden]{display:none}.Markdown{font-size:15px;line-height:1.65;color:#282e36}.Markdown>:first-child{margin-top:0}.Markdown>:last-child{margin-bottom:0}.Markdown h1,.Markdown h2,.Markdown h3,.Markdown h4,.Markdown h5,.Markdown h6{line-height:1.35;margin:0 0 10px;color:#171a1f;font-weight:660}.Markdown h1{font-size:21px}.Markdown h2{font-size:18px}.Markdown h3{font-size:16px}.Markdown h4,.Markdown h5,.Markdown h6{font-size:15px}.Markdown p{margin:0 0 10px}.Markdown ul,.Markdown ol{margin:0 0 10px;padding-left:22px}.Markdown li+li{margin-top:4px}.Markdown a,.Link a{color:#126ee3;text-decoration:none}.Markdown a:hover,.Link a:hover{text-decoration:underline}.Markdown code{padding:2px 5px;border-radius:4px;background:#f0f2f5;font-size:.9em}.Markdown blockquote{margin:0 0 10px;padding-left:12px;border-left:3px solid #d9dde3;color:#687078}[data-component-id="metrics"]{flex-wrap:wrap}[data-component-id="metrics"]>.Column{min-width:108px;min-height:68px;justify-content:center;border:1px solid #edf0f3}[data-component-id^="metric_value_"]{font-size:18px;color:#171a1f}[data-component-id="highlights"]>.Row{border:1px solid #edf0f3}.FileControl{display:flex;align-items:center;gap:8px;padding:8px;border:1px solid #e2e6eb;border-radius:10px;background:#fafbfc}.FileMain{display:flex;min-width:0;flex:1;align-items:center;gap:12px;padding:4px;color:inherit;text-decoration:none}.FileControl:hover{border-color:#b8c8df;background:#f7f9fc}.FileIcon{display:flex;width:40px;height:40px;flex:none;align-items:center;justify-content:center;border-radius:8px;background:#e8f3ff;color:#075fc1;font-size:11px;font-weight:700;letter-spacing:.03em}.FileCopy{min-width:0;flex:1}.FileName{display:block;overflow:hidden;color:#20252b;font-weight:600;text-overflow:ellipsis;white-space:nowrap}.FileMeta{display:block;margin-top:3px;color:#5f6872;font-size:12px}.FileAction{display:flex;min-width:44px;min-height:44px;align-items:center;justify-content:center;padding:8px;color:#075fc1;font-size:13px;font-weight:600;text-decoration:none}.CollapsiblePanel{border-radius:10px;background:#f7f8fa}.CollapsiblePanel details{padding:0 12px}.CollapsiblePanel summary{padding:12px 0;cursor:pointer;color:#343b44;font-weight:600;list-style-position:outside}.CollapsiblePanel details[open]>summary{border-bottom:1px solid #e3e7ec}.PanelContent{margin:12px 0;overflow:auto}.CollapsiblePanel.indented .PanelContent{padding-left:12px;border-left:2px solid #d9dde3}.Button{flex:1}.ButtonControl{box-sizing:border-box;width:100%;min-height:44px;padding:10px 14px;border:0;border-radius:9px;background:#1677ff;color:#fff;font:inherit;font-weight:600;cursor:pointer}.ButtonControl.default{background:#eef1f5;color:#24292f}.ButtonControl.borderless{background:transparent;color:#075fc1}.ButtonControl:hover{filter:brightness(.97)}.ButtonControl:focus-visible{outline:3px solid #1677ff33;outline-offset:2px}.ButtonControl:active{transform:translateY(1px)}.ButtonControl:disabled{cursor:not-allowed;opacity:.45}.FieldLabel{display:block;margin-bottom:6px;color:#4c5561;font-size:13px;font-weight:600}.TextField{box-sizing:border-box;width:100%;padding:11px;border:1px solid #d4d9df;border-radius:8px;background:#fff;color:#171a1f;font:inherit;resize:vertical}.TextField:focus{outline:2px solid #1677ff33;border-color:#1677ff}.ChoicePicker{padding:4px 0}.option{display:inline-flex;align-items:center;gap:6px;margin:5px 14px 5px 0;cursor:pointer}.interaction-result{box-sizing:border-box;margin-top:14px;padding:12px;border:1px solid #d9dde3;border-radius:9px;background:#fff;color:#171a1f}.interaction-result pre{margin:8px 0 0;white-space:pre-wrap;word-break:break-word}.interaction-result[hidden]{display:none}@media(max-width:480px){body{padding:16px 10px 40px}.Card{padding:16px;border-radius:14px}.Text.title{font-size:19px}[data-component-id="metrics"]{gap:6px}[data-component-id="metrics"]>.Column{min-width:88px;padding:8px}}@media(prefers-color-scheme:dark){:root{background:#121416;color:#eef1f5}.Card,.interaction-result{background:#1f2328;border-color:#353b42;color:#eef1f5}.Divider{background:#353b42}.token-common_fg_z1_color,.CollapsiblePanel{background:#292e35}.Markdown,.Markdown h1,.Markdown h2,.Markdown h3,.Markdown h4,.Markdown h5,.Markdown h6,.Text.title,[data-component-id^="metric_value_"],.FileName,.CollapsiblePanel summary{color:#eef1f5}.Markdown code{background:#292e35}.FieldLabel,.Text.gray,.FileMeta,.ImageFallback{color:#c4cad1}.TextField{background:#171a1f;border-color:#4c5561;color:#eef1f5}.ButtonControl.default{background:#353b42;color:#eef1f5}.FileControl,[data-component-id="metrics"]>.Column,[data-component-id="highlights"]>.Row{border-color:#3b424a;background:#252a30}.FileControl:hover{border-color:#586675;background:#292f36}.CollapsiblePanel details[open]>summary{border-color:#3b424a}.CollapsiblePanel.indented .PanelContent{border-color:#4d5661}.ImageFrame{background:#292e35}.ImageFallback strong{color:#eef1f5}.FileAction,.Link a{color:#79b8ff}}
</style></head><body><div class="notice">DWS A2UI reference_preview · 可本地编辑表单并模拟 Action，不发送真实请求；用于结构、层级与字段映射快速验证，不代表钉钉客户端像素级效果</div><main class="surface">{{template "node" .}}</main><aside id="interaction-result" class="interaction-result" hidden><strong>本地 Action 模拟结果</strong><pre id="interaction-json"></pre></aside>{{define "node"}}<section class="{{.Kind}} {{.Variant}} {{.Theme}} {{.Color}} {{.BackgroundToken}}{{if .DefaultExpanded}} is-expanded{{end}}" data-component-id="{{.ID}}" data-component="{{.Kind}}" data-align="{{.Align}}" data-justify="{{.Justify}}" data-gap="{{.Gap}}" data-padding="{{.Padding}}" data-radius="{{.Radius}}" data-weight="{{.Weight}}" data-max-line="{{.MaxLine}}"{{if .LineStyle}} style="{{.LineStyle}}"{{end}}>{{if eq .Kind "Image"}}{{if .PreviewEnabled}}<a class="ImagePreview" href="{{.ImageSrc}}" target="_blank" rel="noopener noreferrer" aria-label="预览：{{.Alt}}">{{end}}<figure class="ImageFrame {{.Variant}} {{.Align}}" data-load="pending"><img class="Image" src="{{.ImageSrc}}" alt="{{.Alt}}"><figcaption class="ImageFallback" hidden><strong>图片暂不可用</strong><span>{{.Alt}}</span></figcaption></figure>{{if .PreviewEnabled}}</a>{{end}}{{else if eq .Kind "File"}}<div class="FileControl"><a class="FileMain" href="{{.Href}}" target="_blank" rel="noopener noreferrer" data-preview-event="{{.PreviewEventName}}" data-surface-id="{{.PreviewEventSurfaceID}}"><span class="FileIcon">{{.FileType}}</span><span class="FileCopy"><span class="FileName">{{.Text}}</span><span class="FileMeta">{{.Description}}{{if and .Description .FileSize}} · {{end}}{{.FileSize}}{{if and .MimeType .FileSize}} · {{end}}{{.MimeType}}</span></span></a><a class="FileAction" href="{{.DownloadHref}}" target="_blank" rel="noopener noreferrer" aria-label="下载 {{.Text}}">下载</a></div>{{else if eq .Kind "TextField"}}<label>{{if .Label}}<span class="FieldLabel">{{.Label}}</span>{{end}}{{if eq .Variant "longText"}}<textarea class="TextField" placeholder="{{.Placeholder}}" data-binding-path="{{.BindingPath}}"{{if .Disabled}} disabled{{end}}>{{.Value}}</textarea>{{else}}<input class="TextField" placeholder="{{.Placeholder}}" value="{{.Value}}" data-binding-path="{{.BindingPath}}"{{if .Disabled}} disabled{{end}}>{{end}}</label>{{else if eq .Kind "ChoicePicker"}}<div class="ChoicePicker">{{if .Label}}<div class="FieldLabel">{{.Label}}</div>{{end}}{{range .Options}}<label class="option"><input type="{{$.InputType}}" name="{{$.ID}}" value="{{.Value}}" data-binding-path="{{$.BindingPath}}" {{if .Selected}}checked{{end}}{{if $.Disabled}} disabled{{end}}> {{.Label}}</label>{{end}}</div>{{else if eq .Kind "Markdown"}}{{.MarkdownHTML}}{{else if eq .Kind "Link"}}{{if .Href}}<a href="{{.Href}}" target="_blank" rel="noopener noreferrer">{{.Text}}</a>{{else}}{{.Text}}{{end}}{{else if eq .Kind "CollapsiblePanel"}}<details{{if .DefaultExpanded}} open{{end}}><summary>{{.Text}}</summary><div class="PanelContent" data-max-height="{{.MaxHeight}}"{{if .PanelStyle}} style="{{.PanelStyle}}"{{end}}>{{range .Children}}{{template "node" .}}{{end}}</div></details>{{else if eq .Kind "Button"}}<button type="button" class="ButtonControl {{.Variant}}" data-event-name="{{.EventName}}" data-surface-id="{{.EventSurfaceID}}"{{if .Disabled}} disabled{{end}}>{{if .Text}}{{.Text}}{{end}}{{range .Children}}{{template "node" .}}{{end}}</button>{{else}}{{if .Text}}{{.Text}}{{end}}{{range .Children}}{{template "node" .}}{{end}}{{end}}</section>{{end}}<script>
(()=>{const emit=(name,surfaceId)=>{if(!name)return;const model=collect();const payload={event:{name,context:{surfaceId,form:model.form??{}}},localPreview:true};document.getElementById("interaction-json").textContent=JSON.stringify(payload,null,2);document.getElementById("interaction-result").hidden=false;document.dispatchEvent(new CustomEvent("dws-a2ui-preview-action",{detail:payload}))};const markImage=(image)=>{const frame=image.closest(".ImageFrame");const fallback=frame.querySelector(".ImageFallback");const failed=!image.naturalWidth||image.naturalWidth<=1||image.naturalHeight<=1;frame.dataset.load=failed?"error":"ready";fallback.hidden=!failed};document.querySelectorAll(".Image").forEach(image=>{image.addEventListener("load",()=>markImage(image));image.addEventListener("error",()=>markImage(image));if(image.complete)markImage(image)});const assign=(target,path,value)=>{const parts=path.replace(/^\//,"").split("/").filter(Boolean);let current=target;parts.forEach((part,index)=>{if(index===parts.length-1){current[part]=value;return}current[part]??={};current=current[part]})};const collect=()=>{const model={};document.querySelectorAll("[data-binding-path]").forEach(control=>{if((control.type==="radio"||control.type==="checkbox")&&!control.checked)return;assign(model,control.dataset.bindingPath,control.value)});return model};document.querySelectorAll("button[data-event-name]").forEach(button=>button.addEventListener("click",()=>emit(button.dataset.eventName,button.dataset.surfaceId)));document.querySelectorAll("[data-preview-event]").forEach(link=>link.addEventListener("click",()=>emit(link.dataset.previewEvent,link.dataset.surfaceId)))})();
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
	view := node{ID: id, Kind: kind, InputType: "radio"}
	if variant, ok := component["variant"].(string); ok {
		view.Variant = variant
	}
	view.Theme, _ = component["theme"].(string)
	view.Color, _ = component["color"].(string)
	view.Align, _ = component["align"].(string)
	view.Justify, _ = component["justify"].(string)
	view.BackgroundToken, _ = component["backgroundColorToken"].(string)
	view.Gap = numberAttribute(component["gap"])
	view.Padding = numberAttribute(component["padding"])
	view.Radius = numberAttribute(component["cornerRadius"])
	view.Weight = numberAttribute(component["weight"])
	view.MaxLine = numberAttribute(component["maxLine"])
	view.MaxHeight = numberAttribute(component["maxHeight"])
	view.DefaultExpanded, _ = component["defaultExpanded"].(bool)
	view.Disabled, _ = component["disabled"].(bool)
	if kind == "Text" {
		if view.MaxLine == "" {
			view.MaxLine = "3"
		}
		view.LineStyle = template.CSS("-webkit-line-clamp:" + view.MaxLine) // #nosec G203 -- MaxLine is a validated finite number.
	}
	if kind == "CollapsiblePanel" {
		if view.MaxHeight == "" {
			view.MaxHeight = "300"
		}
		if view.MaxHeight != "0" {
			view.PanelStyle = template.CSS("max-height:" + view.MaxHeight + "px") // #nosec G203 -- MaxHeight is a validated finite number.
		}
	}
	if bold, _ := component["bold"].(bool); bold {
		view.Variant = strings.TrimSpace(view.Variant + " bold")
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
		if view.Variant == "multipleSelection" {
			view.InputType = "checkbox"
		}
	}
	if kind == "Link" {
		view.Href = actionURL(component)
	}
	if kind == "Image" {
		view.Align, _ = component["fit"].(string)
		view.ImageSrc = safeImageURL(view.Text)
		view.PreviewEnabled = true
		if enabled, ok := component["previewEnabled"].(bool); ok {
			view.PreviewEnabled = enabled
		}
		if accessibility, ok := component["accessibility"].(map[string]any); ok {
			view.Alt = resolve(accessibility["label"], surface.Data)
		}
	}
	if kind == "File" {
		view.Description = resolve(component["description"], surface.Data)
		view.MimeType = resolve(component["mimeType"], surface.Data)
		view.FileSize = formatFileSize(component["size"])
		view.FileType = fileType(view.Text)
		view.DownloadHref = resolve(component["url"], surface.Data)
		view.Href = resolve(component["previewUrl"], surface.Data)
		if view.Href == "" {
			view.Href = resolve(component["url"], surface.Data)
		}
		view.PreviewEventName, view.PreviewEventSurfaceID = eventInfoFrom(component["onPreview"])
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

func safeImageURL(value string) template.URL {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "data:image/png;base64,") || strings.HasPrefix(lower, "data:image/jpeg;base64,") ||
		strings.HasPrefix(lower, "data:image/webp;base64,") || strings.HasPrefix(lower, "data:image/svg+xml;base64,") {
		return template.URL(value) // #nosec G203 -- only explicit image URL schemes are admitted above.
	}
	return ""
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
	return eventInfoFrom(component["action"])
}

func eventInfoFrom(raw any) (string, string) {
	action, ok := raw.(map[string]any)
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

func numberAttribute(value any) string {
	number, ok := value.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return ""
	}
	return strconv.FormatFloat(number, 'f', -1, 64)
}

func formatFileSize(value any) string {
	size, ok := value.(float64)
	if !ok || size <= 0 || math.IsNaN(size) || math.IsInf(size, 0) {
		return ""
	}
	units := []string{"B", "KB", "MB", "GB"}
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	precision := 0
	if unit > 0 && size < 10 {
		precision = 1
	}
	return strconv.FormatFloat(size, 'f', precision, 64) + " " + units[unit]
}

func fileType(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 || index == len(name)-1 {
		return "FILE"
	}
	extension := strings.ToUpper(name[index+1:])
	if len(extension) > 4 {
		return "FILE"
	}
	return extension
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
