//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCancelNotification(s *server.MCPServer) {
	tool := mcp.NewTool("cancel_notification",
		mcp.WithDescription(
			"Dismiss a specific notification by its key using 'cmd notification cancel'.",
		),
		mcp.WithString("key",
			mcp.Required(),
			mcp.Description("Notification key (from list_notifications)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleCancelNotification)
}

func handleCancelNotification(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleCancelNotification")
	defer func() { logger.Tracef(ctx, "/handleCancelNotification") }()

	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("cmd notification cancel %s", shellQuote(key))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cancel notification: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("notification %q cancelled", key)
	}

	return mcp.NewToolResultText(out), nil
}
