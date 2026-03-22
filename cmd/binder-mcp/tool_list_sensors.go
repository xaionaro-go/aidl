//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListSensors(s *server.MCPServer) {
	tool := mcp.NewTool("list_sensors",
		mcp.WithDescription(
			"List available hardware sensors (accelerometer, gyroscope, magnetometer, etc.) "+
				"from 'dumpsys sensorservice'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListSensors)
}

func handleListSensors(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListSensors")
	defer func() { logger.Tracef(ctx, "/handleListSensors") }()

	out, err := shellExec("dumpsys sensorservice | grep -E '^[0-9]' | head -50")
	if err != nil {
		// dumpsys output format varies; try alternate parsing.
		altOut, altErr := shellExec("dumpsys sensorservice | head -100")
		if altErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys sensorservice: %v", err)), nil
		}
		out = altOut
	}

	if out == "" {
		out = "no sensors found"
	}

	return mcp.NewToolResultText(out), nil
}
