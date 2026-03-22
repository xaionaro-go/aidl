//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListSettings(s *server.MCPServer) {
	tool := mcp.NewTool("list_settings",
		mcp.WithDescription(
			"List all settings in a given namespace (system, secure, or global) "+
				"using 'settings list'.",
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Settings namespace"),
			mcp.Enum("system", "secure", "global"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListSettings)
}

func handleListSettings(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListSettings")
	defer func() { logger.Tracef(ctx, "/handleListSettings") }()

	namespace, err := request.RequireString("namespace")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("settings list %s", shellQuote(namespace))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("settings list: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
