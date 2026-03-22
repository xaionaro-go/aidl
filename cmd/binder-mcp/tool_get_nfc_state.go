//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetNFCState(s *server.MCPServer) {
	tool := mcp.NewTool("get_nfc_state",
		mcp.WithDescription(
			"Get the NFC enabled/disabled status from 'dumpsys nfc'. "+
				"Returns the NFC adapter state.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetNFCState)
}

func handleGetNFCState(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetNFCState")
	defer func() { logger.Tracef(ctx, "/handleGetNFCState") }()

	out, err := shellExec("dumpsys nfc | grep -E 'mState|enabled' | head -5")
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys nfc: %v", err)), nil
		}
	}

	if out == "" {
		out = "NFC state: unknown (service may not be available)"
	}

	return mcp.NewToolResultText(out), nil
}
