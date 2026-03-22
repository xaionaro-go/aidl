//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListAccounts(s *server.MCPServer) {
	tool := mcp.NewTool("list_accounts",
		mcp.WithDescription(
			"List accounts registered on the device from 'dumpsys account'. "+
				"Shows account type and name.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListAccounts)
}

func handleListAccounts(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListAccounts")
	defer func() { logger.Tracef(ctx, "/handleListAccounts") }()

	out, err := shellExec("dumpsys account | grep -E 'Account {|name=|type=' | head -30")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys account: %v", err)), nil
		}
	}

	if out == "" {
		out = "no accounts found"
	}

	return mcp.NewToolResultText(out), nil
}
