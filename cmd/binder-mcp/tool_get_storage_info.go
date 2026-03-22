//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetStorageInfo(s *server.MCPServer) {
	tool := mcp.NewTool("get_storage_info",
		mcp.WithDescription(
			"Get device storage usage including internal and external partitions. "+
				"Returns filesystem, size, used, available, and mount point from 'df -h'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetStorageInfo)
}

func handleGetStorageInfo(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetStorageInfo")
	defer func() { logger.Tracef(ctx, "/handleGetStorageInfo") }()

	out, err := shellExec("df -h")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("df: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
