//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetNetworkInfo(s *server.MCPServer) {
	tool := mcp.NewTool("get_network_info",
		mcp.WithDescription(
			"Get active network connectivity information including network type, "+
				"transport, and link properties from 'dumpsys connectivity'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetNetworkInfo)
}

func handleGetNetworkInfo(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetNetworkInfo")
	defer func() { logger.Tracef(ctx, "/handleGetNetworkInfo") }()

	out, err := shellExec("dumpsys connectivity | grep -A 5 'Active default network' | head -20")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys connectivity: %v", err)), nil
		}
	}

	if out == "" {
		out = "no active network"
	}

	return mcp.NewToolResultText(out), nil
}
