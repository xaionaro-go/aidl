//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetRunningServices(s *server.MCPServer) {
	tool := mcp.NewTool("get_running_services",
		mcp.WithDescription(
			"List currently running Android services from 'dumpsys activity services'. "+
				"Optionally filter by package name.",
		),
		mcp.WithString("package",
			mcp.Description("Filter by package name (optional)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetRunningServices)
}

func handleGetRunningServices(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetRunningServices")
	defer func() { logger.Tracef(ctx, "/handleGetRunningServices") }()

	pkg := request.GetString("package", "")

	var cmd string
	switch pkg {
	case "":
		cmd = "dumpsys activity services | grep 'ServiceRecord' | head -50"
	default:
		cmd = fmt.Sprintf("dumpsys activity services %s", shellQuote(pkg))
	}

	out, err := shellExec(cmd)
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys activity services: %v", err)), nil
		}
	}

	if out == "" {
		out = "no running services found"
	}

	return mcp.NewToolResultText(out), nil
}
