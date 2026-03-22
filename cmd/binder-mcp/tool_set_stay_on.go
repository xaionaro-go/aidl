//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetStayOn(s *server.MCPServer) {
	tool := mcp.NewTool("set_stay_on",
		mcp.WithDescription(
			"Keep the screen on while the device is plugged in. "+
				"When enabled, sets screen_off_timeout to maximum value. "+
				"When disabled, restores a 30-second timeout.",
		),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("true to keep screen on, false to restore default timeout"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetStayOn)
}

func handleSetStayOn(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetStayOn")
	defer func() { logger.Tracef(ctx, "/handleSetStayOn") }()

	enabled, err := request.RequireBool("enabled")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// svc power stayon sets the stay_on_while_plugged_in setting.
	// Values: 0=disabled, 3=USB+AC, 7=USB+AC+Wireless.
	var cmd string
	switch {
	case enabled:
		cmd = "svc power stayon usb"
	default:
		cmd = "svc power stayon false"
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("svc power stayon: %v", err)), nil
	}

	if out == "" {
		switch {
		case enabled:
			out = "stay on while plugged in: enabled"
		default:
			out = "stay on while plugged in: disabled"
		}
	}

	return mcp.NewToolResultText(out), nil
}
