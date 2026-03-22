//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultBugreportPath = "/data/local/tmp/bugreport.zip"

func registerBugreport(s *server.MCPServer) {
	tool := mcp.NewTool("bugreport",
		mcp.WithDescription(
			"Generate a bug report archive using 'bugreportz'. "+
				"The report is saved to a file on the device. "+
				"This operation can take several minutes.",
		),
		mcp.WithString("path",
			mcp.Description("Output path on device (default: /data/local/tmp/bugreport.zip)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleBugreport)
}

func handleBugreport(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleBugreport")
	defer func() { logger.Tracef(ctx, "/handleBugreport") }()

	path := request.GetString("path", defaultBugreportPath)

	cmd := fmt.Sprintf("bugreportz -o %s", shellQuote(path))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("bugreport: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("bugreport saved to %s", path)
	}

	return mcp.NewToolResultText(out), nil
}
