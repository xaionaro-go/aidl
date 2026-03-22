//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultDragDurationMS = 1000

func registerDragDrop(s *server.MCPServer) {
	tool := mcp.NewTool("drag_drop",
		mcp.WithDescription(
			"Drag from one screen position to another using 'input draganddrop'. "+
				"This simulates a long press, drag, and release gesture.",
		),
		mcp.WithNumber("x1",
			mcp.Required(),
			mcp.Description("Start X coordinate"),
		),
		mcp.WithNumber("y1",
			mcp.Required(),
			mcp.Description("Start Y coordinate"),
		),
		mcp.WithNumber("x2",
			mcp.Required(),
			mcp.Description("End X coordinate"),
		),
		mcp.WithNumber("y2",
			mcp.Required(),
			mcp.Description("End Y coordinate"),
		),
		mcp.WithNumber("duration_ms",
			mcp.Description("Drag duration in milliseconds (default: 1000)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleDragDrop)
}

func handleDragDrop(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleDragDrop")
	defer func() { logger.Tracef(ctx, "/handleDragDrop") }()

	x1, err := request.RequireInt("x1")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	y1, err := request.RequireInt("y1")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	x2, err := request.RequireInt("x2")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	y2, err := request.RequireInt("y2")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	duration := request.GetInt("duration_ms", defaultDragDurationMS)

	cmd := fmt.Sprintf("input draganddrop %d %d %d %d %d", x1, y1, x2, y2, duration)
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("input draganddrop: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("dragged from (%d, %d) to (%d, %d) in %dms", x1, y1, x2, y2, duration)
	}

	return mcp.NewToolResultText(out), nil
}
