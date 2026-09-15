// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/authoring"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/delivery"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/pkg/edition"
	"github.com/spf13/cobra"
)

type cardTestCall struct {
	product, tool string
	args          map[string]any
}
type cardTestCaller struct {
	calls  []cardTestCall
	dryRun bool
}

func (c *cardTestCaller) CallTool(_ context.Context, product, tool string, args map[string]any) (*edition.ToolResult, error) {
	c.calls = append(c.calls, cardTestCall{product: product, tool: tool, args: args})
	response := `{"success":true}`
	if tool == "create_and_send_a2ui_card" {
		response = `{"result":{"bizId":"biz-test-42"}}`
	}
	return &edition.ToolResult{Content: []edition.ContentBlock{{Type: "text", Text: response}}}, nil
}
func (*cardTestCaller) Format() string { return "json" }
func (c *cardTestCaller) DryRun() bool { return c.dryRun }
func (*cardTestCaller) Fields() string { return "" }
func (*cardTestCaller) JQ() string     { return "" }

func runCardCommand(t *testing.T, caller *cardTestCaller, args ...string) (string, error) {
	t.Helper()
	InitDepsForTest(t, caller)
	root := &cobra.Command{Use: "dws", SilenceErrors: true, SilenceUsage: true}
	ctx, _ := output.WithResultStore(context.Background())
	root.SetContext(ctx)
	root.PersistentFlags().String("format", "json", "")
	root.PersistentFlags().String("fields", "", "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().Bool("dry-run", false, "")
	root.PersistentFlags().Bool("yes", false, "")
	root.PersistentFlags().String("profile", "", "")
	root.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error { _, _, err := output.EmitStoredResult(cmd); return err }
	root.AddCommand(newCardCommand())
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append([]string{"card"}, args...))
	err := root.Execute()
	return stdout.String(), err
}

func writeCardMessages(t *testing.T) string {
	t.Helper()
	messages, err := authoring.Compile(authoring.Spec{Recipe: "approval", SurfaceID: "test-surface", Title: "发布审批", Status: "待确认", Body: "请检查本次变更。", PrimaryCTA: "批准", Secondary: "退回"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(messages)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCrossPlatformCoverageCardOfflineComposeLintPreview(t *testing.T) {
	caller := &cardTestCaller{}
	messagePath := writeCardMessages(t)
	stdout, err := runCardCommand(t, caller, "lint", "--file", messagePath, "--mode", "create")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"valid": true`) {
		t.Fatalf("lint output=%s", stdout)
	}
	previewPath := filepath.Join(t.TempDir(), "preview.html")
	stdout, err = runCardCommand(t, caller, "preview", "--file", messagePath, "--output", previewPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(previewPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("reference_preview")) || !bytes.Contains(raw, []byte("发布审批")) {
		t.Fatalf("preview=%s", raw)
	}
	if len(caller.calls) != 0 {
		t.Fatalf("offline commands made calls: %+v", caller.calls)
	}
}

func TestCrossPlatformCoverageCardLintFailureIsStructuredAndOffline(t *testing.T) {
	caller := &cardTestCaller{}
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte(`[{"version":"v1.0","createSurface":{"surfaceId":"s"}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, err := runCardCommand(t, caller, "lint", "--file", path, "--mode", "create")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"outcome": "failure"`, `"subtype": "a2ui_validation_failed"`, `"A2UI_CATALOG_REQUIRED"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("lint output missing %s: %s", want, stdout)
		}
	}
	if len(caller.calls) != 0 {
		t.Fatalf("invalid lint made calls: %+v", caller.calls)
	}
}

func TestCrossPlatformCoverageCardSendUpdateUsesFakeIMAndLedger(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("DWS_CARD_STATE_DIR", stateDir)
	caller := &cardTestCaller{}
	messagePath := writeCardMessages(t)
	stdout, err := runCardCommand(t, caller, "send", "--profile", "test", "--conversation-id", "cid-test", "--file", messagePath)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data struct{ Handle, BizID string } `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("decode %s: %v", stdout, err)
	}
	if envelope.Data.Handle == "" || envelope.Data.BizID != "biz-test-42" {
		t.Fatalf("send output=%s", stdout)
	}
	if len(caller.calls) != 1 || caller.calls[0].product != "im" || caller.calls[0].tool != "create_and_send_a2ui_card" {
		t.Fatalf("calls=%+v", caller.calls)
	}
	wire, ok := caller.calls[0].args["a2uiMessages"].([]string)
	if !ok || len(wire) != 3 {
		t.Fatalf("wire=%#v", caller.calls[0].args["a2uiMessages"])
	}
	deltaPath := filepath.Join(t.TempDir(), "delta.json")
	delta := []map[string]any{{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": "test-surface", "path": "/content/status", "value": "已完成"}}}
	raw, _ := json.Marshal(delta)
	if err := os.WriteFile(deltaPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, err = runCardCommand(t, caller, "finish", "--profile", "test", "--handle", envelope.Data.Handle, "--file", deltaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"flowStatus": "FINISH"`) || !strings.Contains(stdout, `"revision": 2`) {
		t.Fatalf("finish output=%s", stdout)
	}
	if len(caller.calls) != 2 || caller.calls[1].tool != "update_a2ui_card" || caller.calls[1].args["bizId"] != "biz-test-42" {
		t.Fatalf("calls=%+v", caller.calls)
	}
	record, err := (delivery.Store{Dir: stateDir}).Load(envelope.Data.Handle)
	if err != nil {
		t.Fatal(err)
	}
	if record.Revision != 2 || record.FlowStatus != "FINISH" {
		t.Fatalf("record=%+v", record)
	}
	stdout, err = runCardCommand(t, caller, "finish", "--profile", "test", "--handle", envelope.Data.Handle, "--flow-status", "ABORTED")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"flowStatus": "ABORTED"`) || len(caller.calls) != 3 {
		t.Fatalf("status-only finish output=%s calls=%+v", stdout, caller.calls)
	}
	if wire, ok := caller.calls[2].args["a2uiMessages"].([]string); !ok || len(wire) != 0 {
		t.Fatalf("status-only wire=%#v", caller.calls[2].args["a2uiMessages"])
	}
}

func TestCrossPlatformCoverageCardSendDryRunDoesNotCallOrPersist(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("DWS_CARD_STATE_DIR", stateDir)
	caller := &cardTestCaller{dryRun: true}
	stdout, err := runCardCommand(t, caller, "send", "--profile", "test", "--conversation-id", "cid-test", "--file", writeCardMessages(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(caller.calls) != 0 || !strings.Contains(stdout, `"dry_run": true`) {
		t.Fatalf("calls=%+v output=%s", caller.calls, stdout)
	}
	records, err := (delivery.Store{Dir: stateDir}).List()
	if err != nil || len(records) != 0 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
}

func TestCrossPlatformCoverageCardSendResolvesQueryTargets(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("DWS_CARD_STATE_DIR", stateDir)
	messagePath := writeCardMessages(t)

	t.Run("chat query", func(t *testing.T) {
		caller := &cardTestCaller{dryRun: true}
		stdout, err := runCardCommand(t, caller, "send", "--profile", "test", "--chat-query", "cid123456789", "--file", messagePath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout, `"openConversationId": "cid123456789"`) || len(caller.calls) != 0 {
			t.Fatalf("output=%s calls=%+v", stdout, caller.calls)
		}
	})

	t.Run("user query", func(t *testing.T) {
		caller := &cardTestCaller{dryRun: true}
		stdout, err := runCardCommand(t, caller, "send", "--profile", "test", "--user-query", helperCurrentDOpenID, "--file", messagePath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout, `"receiverOpenDingTalkId": "`+helperCurrentDOpenID+`"`) || len(caller.calls) != 0 {
			t.Fatalf("output=%s calls=%+v", stdout, caller.calls)
		}
	})
}
