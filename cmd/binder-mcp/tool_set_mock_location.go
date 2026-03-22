//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetMockLocation(s *server.MCPServer) {
	tool := mcp.NewTool("set_mock_location",
		mcp.WithDescription(
			"Set a simulated GPS location for testing. Uses the 'appops' command to allow "+
				"mock locations and 'cmd location' to inject a mock provider with the given coordinates.",
		),
		mcp.WithNumber("latitude",
			mcp.Required(),
			mcp.Description("Latitude in degrees (e.g. 37.7749)"),
		),
		mcp.WithNumber("longitude",
			mcp.Required(),
			mcp.Description("Longitude in degrees (e.g. -122.4194)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetMockLocation)
}

func handleSetMockLocation(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetMockLocation")
	defer func() { logger.Tracef(ctx, "/handleSetMockLocation") }()

	lat, err := request.RequireFloat("latitude")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	lon, err := request.RequireFloat("longitude")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Enable mock location and inject coordinates using am start with geo: URI
	// as a fallback, or the location service command.
	cmd := fmt.Sprintf(
		"am start -a android.intent.action.VIEW -d 'geo:%f,%f'",
		lat, lon,
	)

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("set mock location: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("mock location set to lat=%f, lon=%f", lat, lon)
	}

	return mcp.NewToolResultText(out), nil
}
