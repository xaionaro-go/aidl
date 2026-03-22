//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerLaunchApp(s *server.MCPServer) {
	tool := mcp.NewTool("launch_app",
		mcp.WithDescription(
			"Launch an app by package name using 'am start'. If a component "+
				"name is provided, launches that specific activity; otherwise "+
				"launches the default launcher activity.",
		),
		mcp.WithString("package",
			mcp.Required(),
			mcp.Description("Package name (e.g. 'com.android.chrome')"),
		),
		mcp.WithString("activity",
			mcp.Description("Activity component (e.g. 'com.android.chrome/.Main'). "+
				"If omitted, the launcher activity is started."),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleLaunchApp)
}

func handleLaunchApp(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleLaunchApp")
	defer func() { logger.Tracef(ctx, "/handleLaunchApp") }()

	pkg, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	activity := request.GetString("activity", "")

	var cmd string
	switch activity {
	case "":
		// Launch the default launcher activity via monkey.
		cmd = fmt.Sprintf(
			"monkey -p %s -c android.intent.category.LAUNCHER 1",
			shellQuote(pkg),
		)
	default:
		cmd = fmt.Sprintf(
			"am start -n %s",
			shellQuote(activity),
		)
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("launch: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
