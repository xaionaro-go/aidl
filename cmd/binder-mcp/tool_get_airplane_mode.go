//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AirplaneModeResult holds the airplane mode state.
type AirplaneModeResult struct {
	Enabled bool `json:"enabled"`
}

func registerGetAirplaneMode(s *server.MCPServer) {
	tool := mcp.NewTool("get_airplane_mode",
		mcp.WithDescription(
			"Check if airplane mode is currently enabled "+
				"using 'settings get global airplane_mode_on'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetAirplaneMode)
}

func handleGetAirplaneMode(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetAirplaneMode")
	defer func() { logger.Tracef(ctx, "/handleGetAirplaneMode") }()

	out, err := shellExec("settings get global airplane_mode_on")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("settings get: %v", err)), nil
	}

	result := AirplaneModeResult{
		Enabled: strings.TrimSpace(out) == "1",
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling airplane mode: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}
