//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultReadFileLines = 500

func registerReadFile(s *server.MCPServer) {
	tool := mcp.NewTool("read_file",
		mcp.WithDescription(
			"Read the text contents of a file on the device. "+
				"Returns up to N lines (default: 500). For binary files, use pull_file instead.",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("File path on device"),
		),
		mcp.WithNumber("lines",
			mcp.Description("Maximum number of lines to read (default: 500)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleReadFile)
}

func handleReadFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleReadFile")
	defer func() { logger.Tracef(ctx, "/handleReadFile") }()

	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	lines := request.GetInt("lines", defaultReadFileLines)

	cmd := fmt.Sprintf("head -n %d %s", lines, shellQuote(path))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("read file: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
