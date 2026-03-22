//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetWifiEnabled(s *server.MCPServer) {
	tool := mcp.NewTool("set_wifi_enabled",
		mcp.WithDescription(
			"Enable or disable WiFi using 'svc wifi'.",
		),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("true to enable WiFi, false to disable"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetWifiEnabled)
}

func handleSetWifiEnabled(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetWifiEnabled")
	defer func() { logger.Tracef(ctx, "/handleSetWifiEnabled") }()

	enabled, err := request.RequireBool("enabled")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var cmd string
	switch {
	case enabled:
		cmd = "svc wifi enable"
	default:
		cmd = "svc wifi disable"
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("svc wifi: %v", err)), nil
	}

	if out == "" {
		switch {
		case enabled:
			out = "WiFi enabled"
		default:
			out = "WiFi disabled"
		}
	}

	return mcp.NewToolResultText(out), nil
}
