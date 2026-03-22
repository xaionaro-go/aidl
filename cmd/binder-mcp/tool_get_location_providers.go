//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetLocationProviders(s *server.MCPServer) {
	tool := mcp.NewTool("get_location_providers",
		mcp.WithDescription(
			"List available location providers (gps, network, fused, etc.) and their "+
				"status from 'dumpsys location'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetLocationProviders)
}

func handleGetLocationProviders(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetLocationProviders")
	defer func() { logger.Tracef(ctx, "/handleGetLocationProviders") }()

	out, err := shellExec("dumpsys location | grep -E 'provider|enabled' | head -30")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys location: %v", err)), nil
		}
	}

	if out == "" {
		out = "no location providers found"
	}

	return mcp.NewToolResultText(out), nil
}
