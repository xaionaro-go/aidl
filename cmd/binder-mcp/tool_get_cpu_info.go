//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetCPUInfo(s *server.MCPServer) {
	tool := mcp.NewTool("get_cpu_info",
		mcp.WithDescription(
			"Get CPU information including processor model, core count, "+
				"frequencies, and current usage from /proc/cpuinfo and /proc/stat.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetCPUInfo)
}

func handleGetCPUInfo(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetCPUInfo")
	defer func() { logger.Tracef(ctx, "/handleGetCPUInfo") }()

	out, err := shellExec(
		"echo '=== CPU Info ===' && cat /proc/cpuinfo | head -30 && " +
			"echo '=== CPU Usage ===' && cat /proc/stat | head -10",
	)
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("cpu info: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(out), nil
}
