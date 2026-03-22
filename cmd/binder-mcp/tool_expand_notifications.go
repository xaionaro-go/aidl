//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerExpandNotifications(s *server.MCPServer) {
	tool := mcp.NewTool("expand_notifications",
		mcp.WithDescription(
			"Pull down the notification shade using 'cmd statusbar expand-notifications'.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleExpandNotifications)
}

func handleExpandNotifications(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleExpandNotifications")
	defer func() { logger.Tracef(ctx, "/handleExpandNotifications") }()

	out, err := shellExec("cmd statusbar expand-notifications")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("expand notifications: %v", err)), nil
	}

	if out == "" {
		out = "notification shade expanded"
	}

	return mcp.NewToolResultText(out), nil
}
