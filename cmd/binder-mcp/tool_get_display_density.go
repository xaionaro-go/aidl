//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetDisplayDensity(s *server.MCPServer) {
	tool := mcp.NewTool("get_display_density",
		mcp.WithDescription(
			"Get the current display density (DPI) using 'wm density'. "+
				"Returns both physical and override density if set.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetDisplayDensity)
}

func handleGetDisplayDensity(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetDisplayDensity")
	defer func() { logger.Tracef(ctx, "/handleGetDisplayDensity") }()

	out, err := shellExec("wm density")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("wm density: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
