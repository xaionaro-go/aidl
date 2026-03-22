//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetDisplaySize(s *server.MCPServer) {
	tool := mcp.NewTool("set_display_size",
		mcp.WithDescription(
			"Override the display resolution using 'wm size'. "+
				"Pass width=0 and height=0 to reset to physical default.",
		),
		mcp.WithNumber("width",
			mcp.Required(),
			mcp.Description("Width in pixels (0 to reset)"),
		),
		mcp.WithNumber("height",
			mcp.Required(),
			mcp.Description("Height in pixels (0 to reset)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetDisplaySize)
}

func handleSetDisplaySize(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetDisplaySize")
	defer func() { logger.Tracef(ctx, "/handleSetDisplaySize") }()

	width, err := request.RequireInt("width")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	height, err := request.RequireInt("height")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var cmd string
	switch {
	case width == 0 && height == 0:
		cmd = "wm size reset"
	default:
		cmd = fmt.Sprintf("wm size %dx%d", width, height)
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("wm size: %v", err)), nil
	}

	if out == "" {
		switch {
		case width == 0 && height == 0:
			out = "display size reset to physical default"
		default:
			out = fmt.Sprintf("display size set to %dx%d", width, height)
		}
	}

	return mcp.NewToolResultText(out), nil
}
