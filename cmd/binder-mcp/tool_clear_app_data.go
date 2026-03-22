//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerClearAppData(s *server.MCPServer) {
	tool := mcp.NewTool("clear_app_data",
		mcp.WithDescription(
			"Clear all data for an application using 'pm clear'. "+
				"This removes the app's databases, caches, shared preferences, and files.",
		),
		mcp.WithString("package",
			mcp.Required(),
			mcp.Description("Package name (e.g. com.example.app)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleClearAppData)
}

func handleClearAppData(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleClearAppData")
	defer func() { logger.Tracef(ctx, "/handleClearAppData") }()

	pkg, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("pm clear %s", shellQuote(pkg))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pm clear: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
