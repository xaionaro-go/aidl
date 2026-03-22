//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultVibrateDurationMS = 500

func registerVibrate(s *server.MCPServer) {
	tool := mcp.NewTool("vibrate",
		mcp.WithDescription(
			"Trigger device vibration for the specified duration "+
				"using 'cmd vibrator_manager vibrate'.",
		),
		mcp.WithNumber("duration_ms",
			mcp.Description("Vibration duration in milliseconds (default: 500)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleVibrate)
}

func handleVibrate(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleVibrate")
	defer func() { logger.Tracef(ctx, "/handleVibrate") }()

	duration := request.GetInt("duration_ms", defaultVibrateDurationMS)

	cmd := fmt.Sprintf("cmd vibrator_manager vibrate -d %d waveform -a 255 -t %d", duration, duration)
	out, err := shellExec(cmd)
	if err != nil {
		// Try the legacy vibrator command as fallback.
		legacyCmd := fmt.Sprintf("cmd vibrator vibrate %d", duration)
		out, err = shellExec(legacyCmd)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("vibrate: %v", err)), nil
		}
	}

	if out == "" {
		out = fmt.Sprintf("vibrated for %dms", duration)
	}

	return mcp.NewToolResultText(out), nil
}
