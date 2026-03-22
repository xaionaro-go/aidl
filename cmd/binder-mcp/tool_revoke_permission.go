//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerRevokePermission(s *server.MCPServer) {
	tool := mcp.NewTool("revoke_permission",
		mcp.WithDescription(
			"Revoke a runtime permission from an app using 'pm revoke'. "+
				"Only runtime (dangerous) permissions can be revoked this way.",
		),
		mcp.WithString("package",
			mcp.Required(),
			mcp.Description("Package name (e.g. com.example.app)"),
		),
		mcp.WithString("permission",
			mcp.Required(),
			mcp.Description("Permission name (e.g. android.permission.CAMERA)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleRevokePermission)
}

func handleRevokePermission(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleRevokePermission")
	defer func() { logger.Tracef(ctx, "/handleRevokePermission") }()

	pkg, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	perm, err := request.RequireString("permission")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("pm revoke %s %s", shellQuote(pkg), shellQuote(perm))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pm revoke: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("revoked %s from %s", perm, pkg)
	}

	return mcp.NewToolResultText(out), nil
}
