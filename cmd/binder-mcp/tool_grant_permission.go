//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGrantPermission(s *server.MCPServer) {
	tool := mcp.NewTool("grant_permission",
		mcp.WithDescription(
			"Grant a runtime permission to an app using 'pm grant'. "+
				"Only runtime (dangerous) permissions can be granted this way.",
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

	s.AddTool(tool, handleGrantPermission)
}

func handleGrantPermission(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGrantPermission")
	defer func() { logger.Tracef(ctx, "/handleGrantPermission") }()

	pkg, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	perm, err := request.RequireString("permission")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("pm grant %s %s", shellQuote(pkg), shellQuote(perm))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pm grant: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("granted %s to %s", perm, pkg)
	}

	return mcp.NewToolResultText(out), nil
}
