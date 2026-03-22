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

// ScreenOrientationResult holds parsed orientation data.
type ScreenOrientationResult struct {
	Rotation    string `json:"rotation"`
	UserLocked  string `json:"user_locked,omitempty"`
	RawRotation string `json:"raw_rotation,omitempty"`
}

func registerGetScreenOrientation(s *server.MCPServer) {
	tool := mcp.NewTool("get_screen_orientation",
		mcp.WithDescription(
			"Get the current screen orientation/rotation. "+
				"Returns rotation value (0=portrait, 1=landscape, 2=reverse-portrait, 3=reverse-landscape).",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetScreenOrientation)
}

func handleGetScreenOrientation(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetScreenOrientation")
	defer func() { logger.Tracef(ctx, "/handleGetScreenOrientation") }()

	out, err := shellExec("dumpsys window displays | grep -E 'mCurrentRotation|mUserRotation'")
	if err != nil {
		// dumpsys may still return partial output.
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys window: %v", err)), nil
		}
	}

	result := parseScreenOrientation(out)

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling orientation: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

func parseScreenOrientation(output string) ScreenOrientationResult {
	result := ScreenOrientationResult{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, "mCurrentRotation"):
			// Format: mCurrentRotation=ROTATION_0
			if idx := strings.Index(line, "="); idx >= 0 {
				result.RawRotation = strings.TrimSpace(line[idx+1:])
				result.Rotation = rotationToName(result.RawRotation)
			}
		case strings.Contains(line, "mUserRotation"):
			if idx := strings.Index(line, "="); idx >= 0 {
				result.UserLocked = strings.TrimSpace(line[idx+1:])
			}
		}
	}

	return result
}

func rotationToName(raw string) string {
	switch {
	case strings.Contains(raw, "0"):
		return "portrait"
	case strings.Contains(raw, "1"):
		return "landscape"
	case strings.Contains(raw, "2"):
		return "reverse-portrait"
	case strings.Contains(raw, "3"):
		return "reverse-landscape"
	default:
		return raw
	}
}
