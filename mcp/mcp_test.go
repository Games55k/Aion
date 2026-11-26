package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aion/backend"
	"aion/backend/model"
	"aion/ipc"
)

func sendSession(t *testing.T, srv *Server, requests []map[string]any) []jsonRPCResponse {
	t.Helper()
	var in bytes.Buffer
	enc := json.NewEncoder(&in)
	for _, r := range requests {
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	if err := srv.Serve(context.Background(), &in, &out); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}
	dec := json.NewDecoder(&out)
	var responses []jsonRPCResponse
	for dec.More() {
		var resp jsonRPCResponse
		if err := dec.Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		responses = append(responses, resp)
	}
	return responses
}

func extractToolText(t *testing.T, resp jsonRPCResponse) (string, bool) {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %+v", resp.Error)
	}
	b, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("empty tool content: %s", string(b))
	}
	return res.Content[0].Text, res.IsError
}

// MCP stdio server negotiates initialize, lists only safe tools (no permanent delete or backup restore), and executes end-to-end project/task/job/trash flows.
func TestMCPServerEndToEndAndSafetyBoundary(t *testing.T) {
	root := t.TempDir()
	fixedNow := func() time.Time { return time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC) }
	router, err := ipc.NewRouter(ipc.RouterOptions{DataDir: root, Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(router, ServerInfo{Version: "0.2.0-test"})

	// 1. Handshake + tools/list + notification (which must not produce a response).
	resps := sendSession(t, srv, []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{"protocolVersion": "2025-06-18"},
		},
		{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		},
		{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		},
		{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "ping",
		},
	})
	if len(resps) != 3 {
		t.Fatalf("expected 3 responses (notifications produce none), got %d", len(resps))
	}

	initBytes, _ := json.Marshal(resps[0].Result)
	if !strings.Contains(string(initBytes), `"2025-06-18"`) {
		t.Fatalf("expected negotiated protocol version 2025-06-18, got %s", string(initBytes))
	}

	listBytes, _ := json.Marshal(resps[1].Result)
	var listRes struct {
		Tools []toolSpec `json:"tools"`
	}
	if err := json.Unmarshal(listBytes, &listRes); err != nil {
		t.Fatal(err)
	}
	if len(listRes.Tools) < 20 {
		t.Fatalf("expected full safe toolset, got %d tools", len(listRes.Tools))
	}
	for _, tool := range listRes.Tools {
		lower := strings.ToLower(tool.Name)
		if strings.Contains(lower, "permanent") || strings.Contains(lower, "clear_trash") || strings.Contains(lower, "restore_backup") || strings.Contains(lower, "settings") {
			t.Fatalf("forbidden destructive/settings tool exposed: %s", tool.Name)
		}
	}

	// 2. Create project -> create task -> partial update task -> soft delete -> restore.
	resps = sendSession(t, srv, []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      10,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "create_project",
				"arguments": map[string]any{"name": "Interview Prep", "description": "Prep plan"},
			},
		},
	})
	text, isErr := extractToolText(t, resps[0])
	if isErr {
		t.Fatalf("create_project failed: %s", text)
	}
	var proj model.Project
	if err := json.Unmarshal([]byte(text), &proj); err != nil {
		t.Fatal(err)
	}

	resps = sendSession(t, srv, []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      11,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "create_task",
				"arguments": map[string]any{
					"project_id":  proj.ID,
					"title":       "Review Raft",
					"description": "Read paper sections 1-5",
					"priority":    "high",
					"tags":        []string{"distributed"},
				},
			},
		},
	})
	text, isErr = extractToolText(t, resps[0])
	if isErr {
		t.Fatalf("create_task failed: %s", text)
	}
	var task model.Task
	if err := json.Unmarshal([]byte(text), &task); err != nil {
		t.Fatal(err)
	}

	// Update only status; description and tags must remain intact.
	resps = sendSession(t, srv, []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      12,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "update_task",
				"arguments": map[string]any{
					"id":     task.ID,
					"status": "in_progress",
				},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      13,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "create_job_application",
				"arguments": map[string]any{"company_name": "Stripe", "position_name": "Backend Engineer"},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      14,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "trash_task",
				"arguments": map[string]any{"id": task.ID},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      15,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "get_task",
				"arguments": map[string]any{"id": task.ID},
			},
		},
	})
	updText, updErr := extractToolText(t, resps[0])
	if updErr {
		t.Fatalf("update_task failed: %s", updText)
	}
	var updated model.Task
	if err := json.Unmarshal([]byte(updText), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status != model.InProgress || updated.Description != "Read paper sections 1-5" || len(updated.Tags) != 1 {
		t.Fatalf("partial update_task unexpectedly mutated fields: %+v", updated)
	}

	if _, jobErr := extractToolText(t, resps[1]); jobErr {
		t.Fatalf("create_job_application returned error")
	}
	if _, trashErr := extractToolText(t, resps[2]); trashErr {
		t.Fatalf("trash_task returned error")
	}
	// Fetching a trashed task returns a tool-level error (isError=true) so the agent can handle it gracefully.
	notFoundText, notFoundErr := extractToolText(t, resps[3])
	if !notFoundErr || !strings.Contains(strings.ToLower(notFoundText), "not found") {
		t.Fatalf("expected tool-level not_found error, got isError=%v text=%q", notFoundErr, notFoundText)
	}

	// Verify GUI live-server routing works identically through MCP.
	app, err := backend.Open(context.Background(), backend.Options{DataDir: root, DisableScheduler: true, Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	ipcSrv, err := ipc.StartServer(root, ipc.NewHandler(app, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer ipcSrv.Close()

	resps = sendSession(t, srv, []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      20,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "get_workspace_summary",
				"arguments": map[string]any{},
			},
		},
	})
	sumText, sumErr := extractToolText(t, resps[0])
	if sumErr {
		t.Fatalf("get_workspace_summary via live socket failed: %s", sumText)
	}
	var summary ipc.WorkspaceSummary
	if err := json.Unmarshal([]byte(sumText), &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Projects) != 1 || summary.JobStats.Total != 1 || summary.TrashCount != 1 {
		t.Fatalf("unexpected summary via live socket: %+v", summary)
	}

	// Verify restore -> attach_file_to_task -> search_tasks flow through MCP.
	sampleFile := filepath.Join(t.TempDir(), "design.md")
	if err := os.WriteFile(sampleFile, []byte("# Design"), 0o600); err != nil {
		t.Fatal(err)
	}
	resps = sendSession(t, srv, []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      21,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "restore_trash_item",
				"arguments": map[string]any{"kind": "task", "id": task.ID},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      22,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "attach_file_to_task",
				"arguments": map[string]any{"task_id": task.ID, "path": sampleFile},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      23,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "search_tasks",
				"arguments": map[string]any{"tags": []string{"distributed"}, "status": "in_progress"},
			},
		},
	})
	if txt, errFlag := extractToolText(t, resps[0]); errFlag {
		t.Fatalf("restore_trash_item failed: %s", txt)
	}
	attTxt, attErr := extractToolText(t, resps[1])
	if attErr || !strings.Contains(attTxt, `"name": "design.md"`) {
		t.Fatalf("attach_file_to_task failed: err=%v text=%s", attErr, attTxt)
	}
	searchTxt, searchErr := extractToolText(t, resps[2])
	if searchErr || !strings.Contains(searchTxt, `"project_name": "Interview Prep"`) || !strings.Contains(searchTxt, task.ID) {
		t.Fatalf("search_tasks failed: err=%v text=%s", searchErr, searchTxt)
	}
}
