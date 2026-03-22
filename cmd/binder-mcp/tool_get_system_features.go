//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetSystemFeatures(s *server.MCPServer) {
	tool := mcp.NewTool("get_system_features",
		mcp.WithDescription(
			"List hardware and software features available on the device "+
				"(camera, NFC, bluetooth, sensors, etc.) using 'pm list features'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetSystemFeatures)
}

func handleGetSystemFeatures(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetSystemFeatures")
	defer func() { logger.Tracef(ctx, "/handleGetSystemFeatures") }()

	out, err := shellExec("pm list features")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pm list features: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
