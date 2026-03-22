//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerMoveTaskToFront(s *server.MCPServer) {
	tool := mcp.NewTool("move_task_to_front",
		mcp.WithDescription(
			"Bring a task to the foreground using 'am task lock' or 'am stack movetask'. "+
				"Requires the task ID (from list_recent_tasks).",
		),
		mcp.WithNumber("task_id",
			mcp.Required(),
			mcp.Description("Task ID to bring to front"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleMoveTaskToFront)
}

func handleMoveTaskToFront(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleMoveTaskToFront")
	defer func() { logger.Tracef(ctx, "/handleMoveTaskToFront") }()

	taskID, err := request.RequireInt("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("am task lock %d", taskID)
	out, err := shellExec(cmd)
	if err != nil {
		// Fallback: try am stack movetask.
		fallbackCmd := fmt.Sprintf("am stack movetask %d 0", taskID)
		out, err = shellExec(fallbackCmd)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("move task to front: %v", err)), nil
		}
	}

	if out == "" {
		out = fmt.Sprintf("task %d moved to front", taskID)
	}

	return mcp.NewToolResultText(out), nil
}
