//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetDisplayDensity(s *server.MCPServer) {
	tool := mcp.NewTool("set_display_density",
		mcp.WithDescription(
			"Override the display density (DPI) using 'wm density'. "+
				"Pass dpi=0 to reset to physical default.",
		),
		mcp.WithNumber("dpi",
			mcp.Required(),
			mcp.Description("Density in DPI (0 to reset to default)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetDisplayDensity)
}

func handleSetDisplayDensity(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetDisplayDensity")
	defer func() { logger.Tracef(ctx, "/handleSetDisplayDensity") }()

	dpi, err := request.RequireInt("dpi")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var cmd string
	switch {
	case dpi == 0:
		cmd = "wm density reset"
	default:
		cmd = fmt.Sprintf("wm density %d", dpi)
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("wm density: %v", err)), nil
	}

	if out == "" {
		switch {
		case dpi == 0:
			out = "display density reset to physical default"
		default:
			out = fmt.Sprintf("display density set to %d DPI", dpi)
		}
	}

	return mcp.NewToolResultText(out), nil
}
