//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MobileDataResult holds the mobile data state.
type MobileDataResult struct {
	Enabled bool   `json:"enabled"`
	Raw     string `json:"raw"`
}

func registerGetMobileDataState(s *server.MCPServer) {
	tool := mcp.NewTool("get_mobile_data_state",
		mcp.WithDescription(
			"Get the current mobile data enabled/disabled status "+
				"using 'settings get global mobile_data'.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetMobileDataState)
}

func handleGetMobileDataState(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetMobileDataState")
	defer func() { logger.Tracef(ctx, "/handleGetMobileDataState") }()

	out, err := shellExec("settings get global mobile_data")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("settings get: %v", err)), nil
	}

	result := MobileDataResult{
		Enabled: strings.TrimSpace(out) == "1",
		Raw:     out,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling mobile data state: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}
