//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListAppPermissions(s *server.MCPServer) {
	tool := mcp.NewTool("list_app_permissions",
		mcp.WithDescription(
			"List all permissions requested by a specific package using 'dumpsys package'. "+
				"Shows both install-time and runtime permissions with their grant status.",
		),
		mcp.WithString("package",
			mcp.Required(),
			mcp.Description("Package name (e.g. com.example.app)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListAppPermissions)
}

func handleListAppPermissions(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListAppPermissions")
	defer func() { logger.Tracef(ctx, "/handleListAppPermissions") }()

	pkg, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("dumpsys package %s | grep -A 200 'requested permissions:' | grep -B 200 -m 1 '^[^ ]' | head -200",
		shellQuote(pkg))
	out, err := shellExec(cmd)
	if err != nil {
		// Partial output is still useful.
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys package: %v", err)), nil
		}
	}

	if out == "" {
		out = fmt.Sprintf("no permissions found for %s", pkg)
	}

	return mcp.NewToolResultText(out), nil
}
