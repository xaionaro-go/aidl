//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetTimezone(s *server.MCPServer) {
	tool := mcp.NewTool("get_timezone",
		mcp.WithDescription(
			"Get the current device timezone from 'getprop persist.sys.timezone'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetTimezone)
}

func handleGetTimezone(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetTimezone")
	defer func() { logger.Tracef(ctx, "/handleGetTimezone") }()

	out, err := shellExec("getprop persist.sys.timezone")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getprop: %v", err)), nil
	}

	if out == "" {
		out = "timezone not set"
	}

	return mcp.NewToolResultText(out), nil
}
