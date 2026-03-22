//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetDateTime(s *server.MCPServer) {
	tool := mcp.NewTool("get_date_time",
		mcp.WithDescription(
			"Get the current device date and time using 'date' command. "+
				"Returns the date in ISO-8601 format.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetDateTime)
}

func handleGetDateTime(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetDateTime")
	defer func() { logger.Tracef(ctx, "/handleGetDateTime") }()

	out, err := shellExec("date '+%Y-%m-%dT%H:%M:%S%z'")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("date: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
