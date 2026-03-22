//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerShutdownDevice(s *server.MCPServer) {
	tool := mcp.NewTool("shutdown_device",
		mcp.WithDescription(
			"Shut down the device using 'svc power shutdown'. "+
				"The device will power off completely.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleShutdownDevice)
}

func handleShutdownDevice(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleShutdownDevice")
	defer func() { logger.Tracef(ctx, "/handleShutdownDevice") }()

	out, err := shellExec("svc power shutdown")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("shutdown: %v", err)), nil
	}

	if out == "" {
		out = "shutdown initiated"
	}

	return mcp.NewToolResultText(out), nil
}
