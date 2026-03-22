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

// WifiStateResult holds the get_wifi_state response.
type WifiStateResult struct {
	Enabled bool   `json:"enabled"`
	SSID    string `json:"ssid,omitempty"`
	Error   string `json:"error,omitempty"`
}

func registerGetWifiState(s *server.MCPServer) {
	tool := mcp.NewTool("get_wifi_state",
		mcp.WithDescription(
			"Get the current WiFi state (enabled/disabled and connected SSID). "+
				"Uses 'cmd wifi status' on the device.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetWifiState)
}

func handleGetWifiState(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetWifiState")
	defer func() { logger.Tracef(ctx, "/handleGetWifiState") }()

	out, err := shellExec("cmd wifi status")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("wifi status: %v", err)), nil
	}

	result := parseWifiStatus(out)

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling wifi state: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

func parseWifiStatus(output string) WifiStateResult {
	result := WifiStateResult{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "Wifi is enabled":
			result.Enabled = true
		case line == "Wifi is disabled":
			result.Enabled = false
		case strings.HasPrefix(line, "Wifi is connected to"):
			// Line format: Wifi is connected to "SSID_NAME"
			if q1 := strings.Index(line, "\""); q1 >= 0 {
				if q2 := strings.Index(line[q1+1:], "\""); q2 >= 0 {
					result.SSID = line[q1+1 : q1+1+q2]
				}
			}
		}
	}

	return result
}
