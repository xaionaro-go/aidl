//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDeleteFile(s *server.MCPServer) {
	tool := mcp.NewTool("delete_file",
		mcp.WithDescription(
			"Delete a file on the device using 'rm'. "+
				"Does not delete directories (use 'rm -r' via shell_exec for that).",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("File path on device to delete"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleDeleteFile)
}

func handleDeleteFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleDeleteFile")
	defer func() { logger.Tracef(ctx, "/handleDeleteFile") }()

	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("rm -f %s", shellQuote(path))
	_, err = shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("delete file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("deleted %s", path)), nil
}
