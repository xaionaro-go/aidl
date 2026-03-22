//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListDirectory(s *server.MCPServer) {
	tool := mcp.NewTool("list_directory",
		mcp.WithDescription(
			"List files in a directory on the device using 'ls -la'. "+
				"Returns file permissions, owner, size, date, and name.",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Directory path on device"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListDirectory)
}

func handleListDirectory(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListDirectory")
	defer func() { logger.Tracef(ctx, "/handleListDirectory") }()

	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("ls -la %s", shellQuote(path))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ls: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
