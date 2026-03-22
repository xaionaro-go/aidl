//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetAlarm(s *server.MCPServer) {
	tool := mcp.NewTool("set_alarm",
		mcp.WithDescription(
			"Set an alarm using the Android alarm clock intent. "+
				"Uses 'am start' with ACTION_SET_ALARM.",
		),
		mcp.WithNumber("hour",
			mcp.Required(),
			mcp.Description("Hour (0-23)"),
		),
		mcp.WithNumber("minute",
			mcp.Required(),
			mcp.Description("Minute (0-59)"),
		),
		mcp.WithString("message",
			mcp.Description("Alarm label/message"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleSetAlarm)
}

func handleSetAlarm(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetAlarm")
	defer func() { logger.Tracef(ctx, "/handleSetAlarm") }()

	hour, err := request.RequireInt("hour")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	minute, err := request.RequireInt("minute")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	message := request.GetString("message", "")

	cmd := fmt.Sprintf(
		"am start -a android.intent.action.SET_ALARM --ei android.intent.extra.alarm.HOUR %d --ei android.intent.extra.alarm.MINUTES %d",
		hour, minute,
	)
	if message != "" {
		cmd += fmt.Sprintf(" --es android.intent.extra.alarm.MESSAGE %s", shellQuote(message))
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("set alarm: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
