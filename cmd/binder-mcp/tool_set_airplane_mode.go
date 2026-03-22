//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetAirplaneMode(s *server.MCPServer) {
	tool := mcp.NewTool("set_airplane_mode",
		mcp.WithDescription(
			"Enable or disable airplane mode. Sets the setting and broadcasts "+
				"the AIRPLANE_MODE_CHANGED intent to apply the change.",
		),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("true to enable airplane mode, false to disable"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetAirplaneMode)
}

func handleSetAirplaneMode(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetAirplaneMode")
	defer func() { logger.Tracef(ctx, "/handleSetAirplaneMode") }()

	enabled, err := request.RequireBool("enabled")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var val string
	switch {
	case enabled:
		val = "1"
	default:
		val = "0"
	}

	cmd := fmt.Sprintf(
		"settings put global airplane_mode_on %s && "+
			"am broadcast -a android.intent.action.AIRPLANE_MODE --ez state %s",
		val, val,
	)

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("set airplane mode: %v", err)), nil
	}

	if out == "" {
		switch {
		case enabled:
			out = "airplane mode enabled"
		default:
			out = "airplane mode disabled"
		}
	}

	return mcp.NewToolResultText(out), nil
}
