//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerStopScreenRecording(s *server.MCPServer) {
	tool := mcp.NewTool("stop_screen_recording",
		mcp.WithDescription(
			"Stop any active screen recording by sending SIGINT to the screenrecord process. "+
				"This allows the recording to finalize the mp4 file properly.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleStopScreenRecording)
}

func handleStopScreenRecording(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleStopScreenRecording")
	defer func() { logger.Tracef(ctx, "/handleStopScreenRecording") }()

	// Send SIGINT so screenrecord finalizes the mp4.
	out, err := shellExec("pkill -INT screenrecord")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("stop screenrecord: %v", err)), nil
	}

	if out == "" {
		out = "screen recording stopped"
	}

	return mcp.NewToolResultText(out), nil
}
