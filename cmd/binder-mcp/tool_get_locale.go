//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetLocale(s *server.MCPServer) {
	tool := mcp.NewTool("get_locale",
		mcp.WithDescription(
			"Get the current device language and locale "+
				"from system properties (persist.sys.locale, ro.product.locale).",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetLocale)
}

func handleGetLocale(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetLocale")
	defer func() { logger.Tracef(ctx, "/handleGetLocale") }()

	out, err := shellExec("getprop persist.sys.locale && getprop ro.product.locale")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getprop: %v", err)), nil
	}

	if out == "" {
		out = "locale not set"
	}

	return mcp.NewToolResultText(out), nil
}
