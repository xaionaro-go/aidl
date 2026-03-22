//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListAlarms(s *server.MCPServer) {
	tool := mcp.NewTool("list_alarms",
		mcp.WithDescription(
			"List pending alarms from 'dumpsys alarm'. "+
				"Optionally filter by package name.",
		),
		mcp.WithString("package",
			mcp.Description("Filter alarms by package name (optional)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListAlarms)
}

func handleListAlarms(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListAlarms")
	defer func() { logger.Tracef(ctx, "/handleListAlarms") }()

	pkg := request.GetString("package", "")

	var cmd string
	switch pkg {
	case "":
		cmd = "dumpsys alarm | grep -E 'Batch|tag|when|type' | head -60"
	default:
		cmd = fmt.Sprintf("dumpsys alarm | grep -A 3 %s | head -40", shellQuote(pkg))
	}

	out, err := shellExec(cmd)
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys alarm: %v", err)), nil
		}
	}

	if out == "" {
		out = "no pending alarms"
	}

	return mcp.NewToolResultText(out), nil
}
