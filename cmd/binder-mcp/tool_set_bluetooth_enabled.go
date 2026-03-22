//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetBluetoothEnabled(s *server.MCPServer) {
	tool := mcp.NewTool("set_bluetooth_enabled",
		mcp.WithDescription(
			"Enable or disable Bluetooth using 'svc bluetooth'.",
		),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("true to enable Bluetooth, false to disable"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetBluetoothEnabled)
}

func handleSetBluetoothEnabled(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetBluetoothEnabled")
	defer func() { logger.Tracef(ctx, "/handleSetBluetoothEnabled") }()

	enabled, err := request.RequireBool("enabled")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var cmd string
	switch {
	case enabled:
		cmd = "svc bluetooth enable"
	default:
		cmd = "svc bluetooth disable"
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("svc bluetooth: %v", err)), nil
	}

	if out == "" {
		switch {
		case enabled:
			out = "Bluetooth enabled"
		default:
			out = "Bluetooth disabled"
		}
	}

	return mcp.NewToolResultText(out), nil
}
