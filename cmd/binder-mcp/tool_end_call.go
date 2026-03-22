//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerEndCall(s *server.MCPServer) {
	tool := mcp.NewTool("end_call",
		mcp.WithDescription(
			"Hang up the current phone call using 'input keyevent KEYCODE_ENDCALL'.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleEndCall)
}

func handleEndCall(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleEndCall")
	defer func() { logger.Tracef(ctx, "/handleEndCall") }()

	out, err := shellExec("input keyevent KEYCODE_ENDCALL")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("end call: %v", err)), nil
	}

	if out == "" {
		out = "call ended"
	}

	return mcp.NewToolResultText(out), nil
}
