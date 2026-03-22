//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSendSMS(s *server.MCPServer) {
	tool := mcp.NewTool("send_sms",
		mcp.WithDescription(
			"Send an SMS message using 'am start' with the SENDTO action. "+
				"Opens the default SMS app with the specified number and message pre-filled.",
		),
		mcp.WithString("number",
			mcp.Required(),
			mcp.Description("Recipient phone number"),
		),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("SMS message text"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleSendSMS)
}

func handleSendSMS(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSendSMS")
	defer func() { logger.Tracef(ctx, "/handleSendSMS") }()

	number, err := request.RequireString("number")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	message, err := request.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf(
		"am start -a android.intent.action.SENDTO -d sms:%s --es sms_body %s --ez exit_on_sent true",
		shellQuote(number), shellQuote(message),
	)
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("send sms: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
