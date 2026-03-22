//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetMobileDataEnabled(s *server.MCPServer) {
	tool := mcp.NewTool("set_mobile_data_enabled",
		mcp.WithDescription(
			"Enable or disable mobile data using 'svc data'.",
		),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("true to enable mobile data, false to disable"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetMobileDataEnabled)
}

func handleSetMobileDataEnabled(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetMobileDataEnabled")
	defer func() { logger.Tracef(ctx, "/handleSetMobileDataEnabled") }()

	enabled, err := request.RequireBool("enabled")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var cmd string
	switch {
	case enabled:
		cmd = "svc data enable"
	default:
		cmd = "svc data disable"
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("svc data: %v", err)), nil
	}

	if out == "" {
		switch {
		case enabled:
			out = "mobile data enabled"
		default:
			out = "mobile data disabled"
		}
	}

	return mcp.NewToolResultText(out), nil
}
