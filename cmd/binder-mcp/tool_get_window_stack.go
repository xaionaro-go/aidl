//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetWindowStack(s *server.MCPServer) {
	tool := mcp.NewTool("get_window_stack",
		mcp.WithDescription(
			"Get the current window/activity stack from 'dumpsys activity activities'. "+
				"Shows the activity back stack for all tasks.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetWindowStack)
}

func handleGetWindowStack(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetWindowStack")
	defer func() { logger.Tracef(ctx, "/handleGetWindowStack") }()

	out, err := shellExec("dumpsys activity activities | grep -E 'Task|Activities|Hist #|topActivity' | head -50")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys activity: %v", err)), nil
		}
	}

	if out == "" {
		out = "no window stack info"
	}

	return mcp.NewToolResultText(out), nil
}
