//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetCallState(s *server.MCPServer) {
	tool := mcp.NewTool("get_call_state",
		mcp.WithDescription(
			"Get the current phone call state (idle, ringing, offhook) "+
				"from 'dumpsys telephony.registry'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetCallState)
}

func handleGetCallState(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetCallState")
	defer func() { logger.Tracef(ctx, "/handleGetCallState") }()

	out, err := shellExec("dumpsys telephony.registry | grep mCallState | head -5")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys telephony: %v", err)), nil
		}
	}

	if out == "" {
		out = "call state: unknown"
	}

	return mcp.NewToolResultText(out), nil
}
