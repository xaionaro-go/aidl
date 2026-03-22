//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListAudioDevices(s *server.MCPServer) {
	tool := mcp.NewTool("list_audio_devices",
		mcp.WithDescription(
			"List connected audio input and output devices "+
				"from 'dumpsys audio'. Shows device type, address, and state.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListAudioDevices)
}

func handleListAudioDevices(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListAudioDevices")
	defer func() { logger.Tracef(ctx, "/handleListAudioDevices") }()

	out, err := shellExec("dumpsys audio | grep -A 3 'Devices' | head -40")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys audio: %v", err)), nil
		}
	}

	if out == "" {
		out = "no audio devices found"
	}

	return mcp.NewToolResultText(out), nil
}
