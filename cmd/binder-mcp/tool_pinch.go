//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultPinchDurationMS = 500

func registerPinch(s *server.MCPServer) {
	tool := mcp.NewTool("pinch",
		mcp.WithDescription(
			"Perform a pinch gesture (zoom in or out) centered at (x, y). "+
				"Simulated using two converging or diverging swipes. "+
				"'in' pinches fingers together (zoom out), 'out' spreads them apart (zoom in).",
		),
		mcp.WithNumber("x",
			mcp.Required(),
			mcp.Description("Center X coordinate"),
		),
		mcp.WithNumber("y",
			mcp.Required(),
			mcp.Description("Center Y coordinate"),
		),
		mcp.WithString("direction",
			mcp.Required(),
			mcp.Description("Pinch direction: 'in' (zoom out) or 'out' (zoom in)"),
			mcp.Enum("in", "out"),
		),
		mcp.WithNumber("distance",
			mcp.Description("Distance of pinch from center in pixels (default: 200)"),
		),
		mcp.WithNumber("duration_ms",
			mcp.Description("Pinch duration in milliseconds (default: 500)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handlePinch)
}

func handlePinch(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handlePinch")
	defer func() { logger.Tracef(ctx, "/handlePinch") }()

	x, err := request.RequireInt("x")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	y, err := request.RequireInt("y")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	direction, err := request.RequireString("direction")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	distance := request.GetInt("distance", 200)
	duration := request.GetInt("duration_ms", defaultPinchDurationMS)

	// Simulate pinch with two swipes in sequence (approximate since we cannot
	// do real multi-touch via 'input'). Use swipes from/to center.
	var cmd string
	switch direction {
	case "in":
		// Two swipes converging toward center.
		cmd = fmt.Sprintf(
			"input swipe %d %d %d %d %d && input swipe %d %d %d %d %d",
			x-distance, y, x, y, duration,
			x+distance, y, x, y, duration,
		)
	case "out":
		// Two swipes diverging from center.
		cmd = fmt.Sprintf(
			"input swipe %d %d %d %d %d && input swipe %d %d %d %d %d",
			x, y, x-distance, y, duration,
			x, y, x+distance, y, duration,
		)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported direction: %s", direction)), nil
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pinch: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("pinch %s at (%d, %d) with distance=%d", direction, x, y, distance)
	}

	return mcp.NewToolResultText(out), nil
}
