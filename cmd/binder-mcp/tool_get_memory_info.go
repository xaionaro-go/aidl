//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetMemoryInfo(s *server.MCPServer) {
	tool := mcp.NewTool("get_memory_info",
		mcp.WithDescription(
			"Get device memory (RAM) usage information. "+
				"Returns total, free, available, and buffer/cache memory from /proc/meminfo.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetMemoryInfo)
}

func handleGetMemoryInfo(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetMemoryInfo")
	defer func() { logger.Tracef(ctx, "/handleGetMemoryInfo") }()

	out, err := shellExec("cat /proc/meminfo | head -20")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("meminfo: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
