//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCollapsePanels(s *server.MCPServer) {
	tool := mcp.NewTool("collapse_panels",
		mcp.WithDescription(
			"Collapse the notification shade and quick settings panels "+
				"using 'cmd statusbar collapse'.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleCollapsePanels)
}

func handleCollapsePanels(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleCollapsePanels")
	defer func() { logger.Tracef(ctx, "/handleCollapsePanels") }()

	out, err := shellExec("cmd statusbar collapse")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("collapse panels: %v", err)), nil
	}

	if out == "" {
		out = "panels collapsed"
	}

	return mcp.NewToolResultText(out), nil
}
