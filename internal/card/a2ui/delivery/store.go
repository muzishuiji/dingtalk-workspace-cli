// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package delivery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/card/a2ui/state"
)

type Record struct {
	Handle         string        `json:"handle"`
	BizID          string        `json:"bizId"`
	ConversationID string        `json:"conversationId,omitempty"`
	ReceiverID     string        `json:"receiverOpenDingTalkId,omitempty"`
	Surface        state.Surface `json:"surface"`
	FlowStatus     string        `json:"flowStatus"`
	Revision       int           `json:"revision"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

type Store struct{ Dir string }

func DefaultDir() string {
	if value := strings.TrimSpace(os.Getenv("DWS_CARD_STATE_DIR")); value != "" {
		return value
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return ".dws-card-state"
	}
	return filepath.Join(config, "dws", "cards")
}

func (s Store) Save(record Record) error {
	if record.Handle == "" {
		return fmt.Errorf("card handle is required")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	record.UpdatedAt = time.Now().UTC()
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, ".card-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(s.Dir, record.Handle+".json"))
}

func (s Store) Load(handle string) (Record, error) {
	if strings.ContainsAny(handle, `/\\`) || handle == "" {
		return Record{}, fmt.Errorf("invalid card handle")
	}
	raw, err := os.ReadFile(filepath.Join(s.Dir, handle+".json"))
	if err != nil {
		return Record{}, err
	}
	var record Record
	if err := json.Unmarshal(raw, &record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s Store) List() ([]Record, error) {
	entries, err := os.ReadDir(s.Dir)
	if os.IsNotExist(err) {
		return []Record{}, nil
	}
	if err != nil {
		return nil, err
	}
	var records []Record
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		record, err := s.Load(strings.TrimSuffix(entry.Name(), ".json"))
		if err == nil {
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].UpdatedAt.After(records[j].UpdatedAt) })
	return records, nil
}

func ExtractBizID(response any) string {
	keys := map[string]bool{"bizId": true, "bizID": true, "bizCardId": true}
	var visit func(any) string
	visit = func(value any) string {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if keys[key] {
					if text, ok := child.(string); ok && text != "" {
						return text
					}
				}
			}
			for _, child := range typed {
				if found := visit(child); found != "" {
					return found
				}
			}
		case []any:
			for _, child := range typed {
				if found := visit(child); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return visit(response)
}
