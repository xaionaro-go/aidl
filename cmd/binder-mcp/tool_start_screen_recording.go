//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultRecordingPath = "/data/local/tmp/recording.mp4"

func registerStartScreenRecording(s *server.MCPServer) {
	tool := mcp.NewTool("start_screen_recording",
		mcp.WithDescription(
			"Start recording the device screen to a file using 'screenrecord'. "+
				"The recording runs in the background. Use stop_screen_recording to stop it. "+
				"Default path is /data/local/tmp/recording.mp4. Max duration is 180 seconds.",
		),
		mcp.WithString("path",
			mcp.Description("Output file path on device (default: /data/local/tmp/recording.mp4)"),
		),
		mcp.WithNumber("time_limit",
			mcp.Description("Maximum recording duration in seconds (default: 180, max: 180)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleStartScreenRecording)
}

func handleStartScreenRecording(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleStartScreenRecording")
	defer func() { logger.Tracef(ctx, "/handleStartScreenRecording") }()

	path := request.GetString("path", defaultRecordingPath)
	timeLimit := request.GetInt("time_limit", 180)

	if timeLimit > 180 {
		timeLimit = 180
	}

	// Start screenrecord in the background, detached from this process.
	cmd := fmt.Sprintf(
		"nohup screenrecord --time-limit %d %s > /dev/null 2>&1 &",
		timeLimit, shellQuote(path),
	)

	_, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("screenrecord: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"screen recording started: path=%s, time_limit=%ds", path, timeLimit,
	)), nil
}
