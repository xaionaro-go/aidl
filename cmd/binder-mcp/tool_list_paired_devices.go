//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListPairedDevices(s *server.MCPServer) {
	tool := mcp.NewTool("list_paired_devices",
		mcp.WithDescription(
			"List paired Bluetooth devices from 'dumpsys bluetooth_manager'. "+
				"Shows device name, address, and bond state.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListPairedDevices)
}

func handleListPairedDevices(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListPairedDevices")
	defer func() { logger.Tracef(ctx, "/handleListPairedDevices") }()

	out, err := shellExec("dumpsys bluetooth_manager | grep -A 3 'Bonded devices' | head -30")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys bluetooth_manager: %v", err)), nil
		}
	}

	if out == "" {
		out = "no paired Bluetooth devices"
	}

	return mcp.NewToolResultText(out), nil
}
