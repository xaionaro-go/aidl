//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSetNFCEnabled(s *server.MCPServer) {
	tool := mcp.NewTool("set_nfc_enabled",
		mcp.WithDescription(
			"Enable or disable NFC using 'svc nfc'.",
		),
		mcp.WithBoolean("enabled",
			mcp.Required(),
			mcp.Description("true to enable NFC, false to disable"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleSetNFCEnabled)
}

func handleSetNFCEnabled(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetNFCEnabled")
	defer func() { logger.Tracef(ctx, "/handleSetNFCEnabled") }()

	enabled, err := request.RequireBool("enabled")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var cmd string
	switch {
	case enabled:
		cmd = "svc nfc enable"
	default:
		cmd = "svc nfc disable"
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("svc nfc: %v", err)), nil
	}

	if out == "" {
		switch {
		case enabled:
			out = "NFC enabled"
		default:
			out = "NFC disabled"
		}
	}

	return mcp.NewToolResultText(out), nil
}
