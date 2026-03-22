//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerStartService(s *server.MCPServer) {
	tool := mcp.NewTool("start_service",
		mcp.WithDescription(
			"Start a background service using 'am startservice'. "+
				"The component should be in package/.Service format.",
		),
		mcp.WithString("component",
			mcp.Required(),
			mcp.Description("Service component (e.g. com.example.app/.MyService)"),
		),
		mcp.WithString("extras",
			mcp.Description("Additional am startservice flags (e.g. '--es key value')"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleStartService)
}

func handleStartService(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleStartService")
	defer func() { logger.Tracef(ctx, "/handleStartService") }()

	component, err := request.RequireString("component")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	extras := request.GetString("extras", "")

	cmd := fmt.Sprintf("am startservice -n %s", shellQuote(component))
	if extras != "" {
		cmd += " " + extras
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("am startservice: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
