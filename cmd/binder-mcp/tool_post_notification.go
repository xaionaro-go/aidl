//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPostNotification(s *server.MCPServer) {
	tool := mcp.NewTool("post_notification",
		mcp.WithDescription(
			"Post a notification using 'cmd notification post'. "+
				"Creates a basic notification with the specified tag, title, and text.",
		),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("Notification tag (used as identifier)"),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Notification title"),
		),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("Notification body text"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handlePostNotification)
}

func handlePostNotification(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handlePostNotification")
	defer func() { logger.Tracef(ctx, "/handlePostNotification") }()

	tag, err := request.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("cmd notification post -S bigtext -t %s %s %s",
		shellQuote(title), shellQuote(tag), shellQuote(text))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("post notification: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("notification posted with tag %q", tag)
	}

	return mcp.NewToolResultText(out), nil
}
