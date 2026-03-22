//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCancelVibration(s *server.MCPServer) {
	tool := mcp.NewTool("cancel_vibration",
		mcp.WithDescription(
			"Cancel any ongoing vibration using 'cmd vibrator_manager cancel'.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleCancelVibration)
}

func handleCancelVibration(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleCancelVibration")
	defer func() { logger.Tracef(ctx, "/handleCancelVibration") }()

	out, err := shellExec("cmd vibrator_manager cancel")
	if err != nil {
		// Try legacy command.
		out, err = shellExec("cmd vibrator cancel")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("cancel vibration: %v", err)), nil
		}
	}

	if out == "" {
		out = "vibration cancelled"
	}

	return mcp.NewToolResultText(out), nil
}
