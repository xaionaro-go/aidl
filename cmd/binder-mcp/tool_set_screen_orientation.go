//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetScreenOrientation(s *server.MCPServer) {
	tool := mcp.NewTool("set_screen_orientation",
		mcp.WithDescription(
			"Force the screen orientation. Uses 'settings put system user_rotation' "+
				"and 'settings put system accelerometer_rotation'. "+
				"Set to 'auto' to re-enable auto-rotation.",
		),
		mcp.WithString("orientation",
			mcp.Required(),
			mcp.Description("Orientation: auto, portrait, landscape, reverse-portrait, reverse-landscape"),
			mcp.Enum("auto", "portrait", "landscape", "reverse-portrait", "reverse-landscape"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetScreenOrientation)
}

func handleSetScreenOrientation(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetScreenOrientation")
	defer func() { logger.Tracef(ctx, "/handleSetScreenOrientation") }()

	orientation, err := request.RequireString("orientation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch orientation {
	case "auto":
		// Re-enable accelerometer-based rotation.
		_, err := shellExec("settings put system accelerometer_rotation 1")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("set accelerometer_rotation: %v", err)), nil
		}
		return mcp.NewToolResultText("screen orientation set to auto-rotate"), nil

	case "portrait", "landscape", "reverse-portrait", "reverse-landscape":
		var rotation int
		switch orientation {
		case "portrait":
			rotation = 0
		case "landscape":
			rotation = 1
		case "reverse-portrait":
			rotation = 2
		case "reverse-landscape":
			rotation = 3
		}

		// Disable auto-rotate first, then set the fixed rotation.
		cmd := fmt.Sprintf(
			"settings put system accelerometer_rotation 0 && settings put system user_rotation %d",
			rotation,
		)
		if _, err := shellExec(cmd); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("set orientation: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("screen orientation set to %s (rotation=%d)", orientation, rotation)), nil

	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported orientation: %s", orientation)), nil
	}
}
