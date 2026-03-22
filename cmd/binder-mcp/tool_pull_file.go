//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPullFile(s *server.MCPServer) {
	tool := mcp.NewTool("pull_file",
		mcp.WithDescription(
			"Read a file from the device and return it as base64-encoded content. "+
				"Useful for pulling binary files like images or databases.",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("File path on device to read"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handlePullFile)
}

func handlePullFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handlePullFile")
	defer func() { logger.Tracef(ctx, "/handlePullFile") }()

	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b64, err := shellExec(fmt.Sprintf("base64 -w 0 %s", shellQuote(path)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("read file: %v", err)), nil
	}

	return mcp.NewToolResultText(b64), nil
}
