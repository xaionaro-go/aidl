//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// UptimeResult holds parsed uptime data.
type UptimeResult struct {
	UptimeSeconds float64 `json:"uptime_seconds"`
	IdleSeconds   float64 `json:"idle_seconds"`
	Formatted     string  `json:"formatted"`
}

func registerGetUptime(s *server.MCPServer) {
	tool := mcp.NewTool("get_uptime",
		mcp.WithDescription(
			"Get device uptime and idle time from /proc/uptime. "+
				"Returns uptime in seconds and a human-readable formatted string.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetUptime)
}

func handleGetUptime(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetUptime")
	defer func() { logger.Tracef(ctx, "/handleGetUptime") }()

	out, err := shellExec("cat /proc/uptime")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("uptime: %v", err)), nil
	}

	result := parseUptime(out)

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling uptime: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

func parseUptime(output string) UptimeResult {
	parts := strings.Fields(output)
	result := UptimeResult{}

	if len(parts) >= 1 {
		result.UptimeSeconds, _ = strconv.ParseFloat(parts[0], 64)
	}
	if len(parts) >= 2 {
		result.IdleSeconds, _ = strconv.ParseFloat(parts[1], 64)
	}

	totalSec := int(result.UptimeSeconds)
	days := totalSec / 86400
	hours := (totalSec % 86400) / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60
	result.Formatted = fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)

	return result
}
