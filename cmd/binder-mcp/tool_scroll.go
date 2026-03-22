//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerScroll(s *server.MCPServer) {
	tool := mcp.NewTool("scroll",
		mcp.WithDescription(
			"Scroll at a screen position using 'input roll'. "+
				"Positive dy scrolls down, negative dy scrolls up. "+
				"Positive dx scrolls right, negative dx scrolls left.",
		),
		mcp.WithNumber("dx",
			mcp.Required(),
			mcp.Description("Horizontal scroll amount (positive=right, negative=left)"),
		),
		mcp.WithNumber("dy",
			mcp.Required(),
			mcp.Description("Vertical scroll amount (positive=down, negative=up)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleScroll)
}

func handleScroll(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleScroll")
	defer func() { logger.Tracef(ctx, "/handleScroll") }()

	dx, err := request.RequireInt("dx")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dy, err := request.RequireInt("dy")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("input roll %d %d", dx, dy)
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("input roll: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("scrolled dx=%d dy=%d", dx, dy)
	}

	return mcp.NewToolResultText(out), nil
}
