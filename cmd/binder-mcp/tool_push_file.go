//go:build linux

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPushFile(s *server.MCPServer) {
	tool := mcp.NewTool("push_file",
		mcp.WithDescription(
			"Write base64-encoded content to a file on the device. "+
				"The content is decoded and written to the specified path.",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Destination file path on device"),
		),
		mcp.WithString("content_base64",
			mcp.Required(),
			mcp.Description("Base64-encoded file content"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handlePushFile)
}

func handlePushFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handlePushFile")
	defer func() { logger.Tracef(ctx, "/handlePushFile") }()

	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	contentB64, err := request.RequireString("content_base64")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(contentB64)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("base64 decode: %v", err)), nil
	}

	if err := os.WriteFile(path, decoded, 0644); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("write file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("wrote %d bytes to %s", len(decoded), path)), nil
}
