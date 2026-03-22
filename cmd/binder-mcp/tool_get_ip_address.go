//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetIPAddress(s *server.MCPServer) {
	tool := mcp.NewTool("get_ip_address",
		mcp.WithDescription(
			"Get the device IP address(es) from network interfaces using 'ip addr show'. "+
				"Returns addresses for all active interfaces.",
		),
		mcp.WithString("interface",
			mcp.Description("Specific interface name (e.g. wlan0, rmnet0). Empty for all."),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetIPAddress)
}

func handleGetIPAddress(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetIPAddress")
	defer func() { logger.Tracef(ctx, "/handleGetIPAddress") }()

	iface := request.GetString("interface", "")

	var cmd string
	switch iface {
	case "":
		cmd = "ip addr show | grep 'inet '"
	default:
		cmd = fmt.Sprintf("ip addr show %s | grep 'inet '", shellQuote(iface))
	}

	out, err := shellExec(cmd)
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("ip addr: %v", err)), nil
		}
	}

	if out == "" {
		out = "no IP addresses found"
	}

	return mcp.NewToolResultText(out), nil
}
