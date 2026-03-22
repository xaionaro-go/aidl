//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListConnectedBTDevices(s *server.MCPServer) {
	tool := mcp.NewTool("list_connected_bt_devices",
		mcp.WithDescription(
			"List currently connected Bluetooth devices from 'dumpsys bluetooth_manager'. "+
				"Shows active connections with device name and profile.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListConnectedBTDevices)
}

func handleListConnectedBTDevices(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListConnectedBTDevices")
	defer func() { logger.Tracef(ctx, "/handleListConnectedBTDevices") }()

	out, err := shellExec("dumpsys bluetooth_manager | grep -A 5 'Connected devices' | head -30")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys bluetooth_manager: %v", err)), nil
		}
	}

	if out == "" {
		out = "no connected Bluetooth devices"
	}

	return mcp.NewToolResultText(out), nil
}
