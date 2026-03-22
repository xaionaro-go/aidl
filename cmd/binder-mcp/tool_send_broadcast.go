//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSendBroadcast(s *server.MCPServer) {
	tool := mcp.NewTool("send_broadcast",
		mcp.WithDescription(
			"Send a broadcast intent using 'am broadcast'. "+
				"Specify the action and optionally a component, extras, and data URI.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Intent action (e.g. android.intent.action.BOOT_COMPLETED)"),
		),
		mcp.WithString("component",
			mcp.Description("Target component (e.g. com.example/.MyReceiver)"),
		),
		mcp.WithString("data_uri",
			mcp.Description("Intent data URI"),
		),
		mcp.WithString("extras",
			mcp.Description("Additional am broadcast flags (e.g. '--es key value --ei count 5')"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleSendBroadcast)
}

func handleSendBroadcast(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSendBroadcast")
	defer func() { logger.Tracef(ctx, "/handleSendBroadcast") }()

	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	component := request.GetString("component", "")
	dataURI := request.GetString("data_uri", "")
	extras := request.GetString("extras", "")

	cmd := fmt.Sprintf("am broadcast -a %s", shellQuote(action))
	if component != "" {
		cmd += fmt.Sprintf(" -n %s", shellQuote(component))
	}
	if dataURI != "" {
		cmd += fmt.Sprintf(" -d %s", shellQuote(dataURI))
	}
	if extras != "" {
		cmd += " " + extras
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("am broadcast: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
