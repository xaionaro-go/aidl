//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerWriteFile(s *server.MCPServer) {
	tool := mcp.NewTool("write_file",
		mcp.WithDescription(
			"Write text content to a file on the device. "+
				"For binary content, use push_file with base64 encoding instead.",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("File path on device"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Text content to write"),
		),
		mcp.WithBoolean("append",
			mcp.Description("Append to file instead of overwriting (default: false)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleWriteFile)
}

func handleWriteFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleWriteFile")
	defer func() { logger.Tracef(ctx, "/handleWriteFile") }()

	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content, err := request.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	appendMode := request.GetBool("append", false)

	var operator string
	switch {
	case appendMode:
		operator = ">>"
	default:
		operator = ">"
	}

	cmd := fmt.Sprintf("printf '%%s' %s %s %s", shellQuote(content), operator, shellQuote(path))
	_, err = shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("write file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("wrote %d bytes to %s", len(content), path)), nil
}
