//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerClearLogcat(s *server.MCPServer) {
	tool := mcp.NewTool("clear_logcat",
		mcp.WithDescription(
			"Clear the logcat buffer using 'logcat -c'. "+
				"Useful before running a test to get a clean log.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleClearLogcat)
}

func handleClearLogcat(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleClearLogcat")
	defer func() { logger.Tracef(ctx, "/handleClearLogcat") }()

	_, err := shellExec("logcat -c")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("logcat -c: %v", err)), nil
	}

	return mcp.NewToolResultText("logcat buffer cleared"), nil
}
