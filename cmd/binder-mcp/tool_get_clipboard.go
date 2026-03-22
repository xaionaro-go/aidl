//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ClipboardResult holds the get_clipboard response.
type ClipboardResult struct {
	Text  string `json:"text"`
	Error string `json:"error,omitempty"`
}

func registerGetClipboard(s *server.MCPServer) {
	tool := mcp.NewTool("get_clipboard",
		mcp.WithDescription(
			"Get the current clipboard text content. Uses the Android "+
				"clipboard service via shell.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetClipboard)
}

func handleGetClipboard(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetClipboard")
	defer func() { logger.Tracef(ctx, "/handleGetClipboard") }()

	// Try multiple approaches to read clipboard content.
	// 'service call clipboard 11' (hasClipboardText) is not useful directly.
	// Use am broadcast or content query when available; fall back to
	// dumpsys clipboard for basic info.
	text, err := shellExec(
		"dumpsys clipboard 2>/dev/null | grep -A 5 'Primary clip' | head -10",
	)
	if err != nil || text == "" {
		text = "(clipboard contents not accessible from shell UID)"
	}

	out, err := json.Marshal(ClipboardResult{Text: text})
	if err != nil {
		return nil, fmt.Errorf("marshaling clipboard: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}
