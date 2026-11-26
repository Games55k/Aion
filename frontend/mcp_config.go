package frontend

import (
	"encoding/json"
	"strings"
)

// MCPCapability describes the local MCP/IPC bridge status and launch command details
// shown in Settings so users can connect Antigravity or Codex with one click.
type MCPCapability struct {
	Ready          bool
	SocketPath     string
	ExecutablePath string
	DataDirArg     string
	LastError      string
}

// MCPCapability returns a snapshot of the local MCP bridge state without acquiring Store locks.
func (w *Workbench) MCPCapability() MCPCapability {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.mcpInfo
}

// SetMCPCapability updates the cached MCP bridge state without acquiring Store locks.
func (w *Workbench) SetMCPCapability(info MCPCapability) {
	w.mu.Lock()
	w.mcpInfo = info
	w.mu.Unlock()
}

func mcpCommandAndArgs(info MCPCapability) (string, []string) {
	cmd := strings.TrimSpace(info.ExecutablePath)
	if cmd == "" {
		cmd = "/Applications/Aion.app/Contents/MacOS/Aion"
	}
	args := []string{"mcp"}
	if dir := strings.TrimSpace(info.DataDirArg); dir != "" {
		args = append(args, "-data-dir", dir)
	}
	return cmd, args
}

// FormatMCPJSONConfig formats a standard mcpServers JSON snippet compatible with Antigravity.
func FormatMCPJSONConfig(info MCPCapability) string {
	cmd, args := mcpCommandAndArgs(info)
	payload := struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}{
		MCPServers: map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}{
			"aion": {
				Command: cmd,
				Args:    args,
			},
		},
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

// FormatCodexMCPCommand formats a shell command that registers Aion as a stdio MCP server in Codex.
func FormatCodexMCPCommand(info MCPCapability) string {
	cmd, args := mcpCommandAndArgs(info)
	parts := make([]string, 0, 5+len(args))
	parts = append(parts, "codex", "mcp", "add", "aion", "--", shellQuote(cmd))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '/' || r == '.' || r == '-' || r == '_' || r == ':' || r == '@') {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
