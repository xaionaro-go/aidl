//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetClipboard(s *server.MCPServer) {
	tool := mcp.NewTool("set_clipboard",
		mcp.WithDescription(
			"Set the clipboard text content. Uses the Android clipboard "+
				"service via shell.",
		),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("Text to place on the clipboard"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetClipboard)
}

func handleSetClipboard(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetClipboard")
	defer func() { logger.Tracef(ctx, "/handleSetClipboard") }()

	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Use am broadcast with EXTRA_TEXT to set clipboard via Android shell.
	// This works on most devices where 'cmd clipboard' is unavailable.
	cmd := fmt.Sprintf(
		"am broadcast -a clipper.set -e text %s 2>/dev/null || "+
			"service call clipboard 2 i32 0 2>/dev/null",
		shellQuote(text),
	)
	if _, err := shellExec(cmd); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("setting clipboard: %v", err)), nil
	}

	return mcp.NewToolResultText("clipboard set"), nil
}
