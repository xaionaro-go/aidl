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

// FocusedActivityResult holds the get_focused_activity response.
type FocusedActivityResult struct {
	Component string `json:"component"`
	Error     string `json:"error,omitempty"`
}

func registerGetFocusedActivity(s *server.MCPServer) {
	tool := mcp.NewTool("get_focused_activity",
		mcp.WithDescription(
			"Get the component name of the currently focused activity "+
				"(e.g. 'com.android.chrome/.ChromeTabbedActivity').",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetFocusedActivity)
}

func handleGetFocusedActivity(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetFocusedActivity")
	defer func() { logger.Tracef(ctx, "/handleGetFocusedActivity") }()

	// Parse the topResumedActivity, ResumedActivity, or mFocusedActivity line.
	out, err := shellExec("dumpsys activity activities | grep -E 'topResumedActivity=|ResumedActivity:|mFocusedActivity:' | head -1")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("dumpsys: %v", err)), nil
	}

	component := extractComponent(out)
	if component == "" {
		return mcp.NewToolResultError("no focused activity found"), nil
	}

	data, err := json.Marshal(FocusedActivityResult{Component: component})
	if err != nil {
		return nil, fmt.Errorf("marshaling focused activity: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

// extractComponent extracts the component name (e.g. "com.foo/.Bar") from
// a dumpsys line like: "mResumedActivity: ActivityRecord{abc u0 com.foo/.Bar t5}"
func extractComponent(line string) string {
	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.Contains(p, "/") && !strings.HasPrefix(p, "{") {
			return strings.TrimRight(p, "}")
		}
	}
	return ""
}
