// Package mcp implements Aion's local stdio Model Context Protocol (MCP) server.
// It exposes safe read/write/soft-delete tools over newline-delimited JSON-RPC 2.0
// backed by Aion's local IPC/direct router.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	defaultProtocolVersion = "2025-03-26"
	maxFrameBytes          = 8 * 1024 * 1024
)

var supportedProtocolVersions = map[string]bool{
	"2025-06-18": true,
	"2025-03-26": true,
	"2024-11-05": true,
}

const serverInstructions = "Aion is a local-first macOS personal workbench with Projects/Kanban Tasks and Job Applications (求职). " +
	"Start with get_workspace_summary to inspect active projects and job applications. " +
	"All update_* tools perform partial updates (omitted fields are preserved). " +
	"All dates use YYYY-MM-DD. Deletions are soft-deletes into Trash and can be undone with restore_trash_item."

// Caller abstracts the local IPC/direct backend router.
type Caller interface {
	Call(ctx context.Context, method string, params any) (json.RawMessage, error)
}

// ServerInfo describes server metadata returned during MCP initialization.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Server serves MCP JSON-RPC 2.0 requests over an io.Reader / io.Writer pair.
type Server struct {
	caller   Caller
	info     ServerInfo
	tools    []toolSpec
	toolByID map[string]toolSpec
}

// NewServer constructs an Aion MCP server backed by caller.
func NewServer(caller Caller, info ServerInfo) *Server {
	if strings.TrimSpace(info.Name) == "" {
		info.Name = "aion"
	}
	if strings.TrimSpace(info.Version) == "" {
		info.Version = "dev"
	}
	tools := builtinTools()
	byID := make(map[string]toolSpec, len(tools))
	for _, t := range tools {
		byID[t.Name] = t
	}
	return &Server{
		caller:   caller,
		info:     info,
		tools:    tools,
		toolByID: byID,
	}
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	errParse          = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
)

// Serve reads newline-delimited JSON-RPC 2.0 messages from in and writes responses to out
// until EOF or ctx cancellation.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReaderSize(in, 64*1024)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, err := readLine(reader, maxFrameBytes)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(trimmed, &req); err != nil {
			_ = encoder.Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &jsonRPCError{Code: errParse, Message: "parse error: " + err.Error()},
			})
			continue
		}

		isNotification := len(bytes.TrimSpace(req.ID)) == 0
		if req.JSONRPC != "" && req.JSONRPC != "2.0" {
			if !isNotification {
				_ = encoder.Encode(jsonRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &jsonRPCError{Code: errInvalidRequest, Message: "unsupported jsonrpc version"},
				})
			}
			continue
		}

		resp, shouldReply := s.handleRequest(ctx, req, isNotification)
		if !shouldReply {
			continue
		}
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req jsonRPCRequest, isNotification bool) (jsonRPCResponse, bool) {
	// Notifications never produce JSON-RPC responses.
	if isNotification || strings.HasPrefix(req.Method, "notifications/") {
		return jsonRPCResponse{}, false
	}

	okResp := func(result any) (jsonRPCResponse, bool) {
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result}, true
	}
	errResp := func(code int, msg string) (jsonRPCResponse, bool) {
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: code, Message: msg}}, true
	}

	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &params)
		}
		proto := defaultProtocolVersion
		if supportedProtocolVersions[params.ProtocolVersion] {
			proto = params.ProtocolVersion
		}
		return okResp(map[string]any{
			"protocolVersion": proto,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo":   s.info,
			"instructions": serverInstructions,
		})

	case "ping":
		return okResp(map[string]any{})

	case "tools/list":
		return okResp(map[string]any{"tools": s.tools})

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || strings.TrimSpace(params.Name) == "" {
			return errResp(errInvalidParams, "tools/call requires name and optional arguments object")
		}
		spec, ok := s.toolByID[params.Name]
		if !ok {
			return errResp(errInvalidParams, fmt.Sprintf("unknown tool %q", params.Name))
		}
		args := params.Arguments
		if len(bytes.TrimSpace(args)) == 0 || string(bytes.TrimSpace(args)) == "null" {
			args = json.RawMessage("{}")
		}
		rawRes, callErr := s.caller.Call(ctx, spec.ipcMethod, args)
		if callErr != nil {
			return okResp(map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": callErr.Error()},
				},
				"isError": true,
			})
		}
		return okResp(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": prettyJSON(rawRes)},
			},
			"isError": false,
		})

	case "resources/list":
		return okResp(map[string]any{"resources": []any{}})

	case "prompts/list":
		return okResp(map[string]any{"prompts": []any{}})

	default:
		return errResp(errMethodNotFound, fmt.Sprintf("method %q not found", req.Method))
	}
}

func prettyJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
}

func readLine(r *bufio.Reader, limit int) ([]byte, error) {
	var out []byte
	for {
		frag, isPrefix, err := r.ReadLine()
		if err != nil {
			if len(out) > 0 && errors.Is(err, io.EOF) {
				return out, nil
			}
			return nil, err
		}
		if len(out)+len(frag) > limit {
			return nil, errors.New("frame exceeds maximum size")
		}
		out = append(out, frag...)
		if !isPrefix {
			return out, nil
		}
	}
}
