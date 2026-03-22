//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerRebootDevice(s *server.MCPServer) {
	tool := mcp.NewTool("reboot_device",
		mcp.WithDescription(
			"Reboot the Android device. Supports normal reboot, "+
				"bootloader, and recovery modes via 'svc power reboot'.",
		),
		mcp.WithString("mode",
			mcp.Description("Reboot mode: empty for normal, 'bootloader', or 'recovery'"),
			mcp.Enum("", "bootloader", "recovery"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleRebootDevice)
}

func handleRebootDevice(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleRebootDevice")
	defer func() { logger.Tracef(ctx, "/handleRebootDevice") }()

	mode := request.GetString("mode", "")

	var cmd string
	switch mode {
	case "":
		cmd = "svc power reboot"
	case "bootloader", "recovery":
		cmd = fmt.Sprintf("svc power reboot %s", mode)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported reboot mode: %s", mode)), nil
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reboot: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("reboot initiated (mode: %s)", mode)
		if mode == "" {
			out = "reboot initiated"
		}
	}

	return mcp.NewToolResultText(out), nil
}
