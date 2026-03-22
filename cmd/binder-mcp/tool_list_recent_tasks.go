//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListRecentTasks(s *server.MCPServer) {
	tool := mcp.NewTool("list_recent_tasks",
		mcp.WithDescription(
			"List recent tasks from 'dumpsys activity recents'. "+
				"Shows the recent tasks stack with activity info.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListRecentTasks)
}

func handleListRecentTasks(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListRecentTasks")
	defer func() { logger.Tracef(ctx, "/handleListRecentTasks") }()

	out, err := shellExec("dumpsys activity recents | grep -E 'Recent #|realActivity|baseIntent' | head -50")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys activity recents: %v", err)), nil
		}
	}

	if out == "" {
		out = "no recent tasks"
	}

	return mcp.NewToolResultText(out), nil
}
