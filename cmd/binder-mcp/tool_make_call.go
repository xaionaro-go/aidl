//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerMakeCall(s *server.MCPServer) {
	tool := mcp.NewTool("make_call",
		mcp.WithDescription(
			"Initiate a phone call to a number using 'am start -a android.intent.action.CALL'. "+
				"The call is placed immediately (not just dialed).",
		),
		mcp.WithString("number",
			mcp.Required(),
			mcp.Description("Phone number to call (e.g. '+1234567890')"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleMakeCall)
}

func handleMakeCall(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleMakeCall")
	defer func() { logger.Tracef(ctx, "/handleMakeCall") }()

	number, err := request.RequireString("number")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("am start -a android.intent.action.CALL -d tel:%s", shellQuote(number))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("make call: %v", err)), nil
	}

	return mcp.NewToolResultText(out), nil
}
