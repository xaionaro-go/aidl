//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AndroidGoLab/binder/servicemanager"
)

// IsScreenOnResult holds the is_screen_on response.
type IsScreenOnResult struct {
	Interactive bool   `json:"interactive"`
	Error       string `json:"error,omitempty"`
}

func (ts *ToolSet) registerIsScreenOn(s *server.MCPServer) {
	tool := mcp.NewTool("is_screen_on",
		mcp.WithDescription(
			"Check whether the device screen is currently on (interactive). "+
				"Returns {interactive: true/false}.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleIsScreenOn)
}

func (ts *ToolSet) handleIsScreenOn(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleIsScreenOn")
	defer func() { logger.Tracef(ctx, "/handleIsScreenOn") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("power"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("power service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, powerManagerDescriptor, "isInteractive")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving isInteractive: %v", err)), nil
	}

	interactive, err := transactBool(ctx, svc, powerManagerDescriptor, code)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("isInteractive: %v", err)), nil
	}

	data, err := json.Marshal(IsScreenOnResult{Interactive: interactive})
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}
