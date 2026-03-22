//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetBatteryStats(s *server.MCPServer) {
	tool := mcp.NewTool("get_battery_stats",
		mcp.WithDescription(
			"Get detailed battery consumption statistics from 'dumpsys batterystats'. "+
				"Returns estimated power use by apps and system components.",
		),
		mcp.WithString("package",
			mcp.Description("Filter stats for a specific package (optional)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetBatteryStats)
}

func handleGetBatteryStats(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetBatteryStats")
	defer func() { logger.Tracef(ctx, "/handleGetBatteryStats") }()

	pkg := request.GetString("package", "")

	var cmd string
	switch pkg {
	case "":
		cmd = "dumpsys batterystats | grep -E 'Estimated power|Uid|Screen|Wifi|Cell' | head -50"
	default:
		cmd = fmt.Sprintf("dumpsys batterystats %s | head -80", shellQuote(pkg))
	}

	out, err := shellExec(cmd)
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("dumpsys batterystats: %v", err)), nil
		}
	}

	if out == "" {
		out = "no battery stats available"
	}

	return mcp.NewToolResultText(out), nil
}
