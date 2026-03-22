//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetAppProcesses(s *server.MCPServer) {
	tool := mcp.NewTool("get_app_processes",
		mcp.WithDescription(
			"List running app processes with memory usage from 'dumpsys meminfo'. "+
				"Shows PID, process name, and memory consumed (PSS).",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetAppProcesses)
}

func handleGetAppProcesses(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetAppProcesses")
	defer func() { logger.Tracef(ctx, "/handleGetAppProcesses") }()

	out, err := shellExec("dumpsys meminfo --package -S | head -80")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys meminfo: %v", err)), nil
		}
	}

	if out == "" {
		out = "no running app processes"
	}

	return mcp.NewToolResultText(out), nil
}
