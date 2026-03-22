//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerInsertContent(s *server.MCPServer) {
	tool := mcp.NewTool("insert_content",
		mcp.WithDescription(
			"Insert data into a content provider URI using 'content insert'. "+
				"Values are specified as key=value pairs.",
		),
		mcp.WithString("uri",
			mcp.Required(),
			mcp.Description("Content provider URI"),
		),
		mcp.WithString("values",
			mcp.Required(),
			mcp.Description("Values to insert as 'content insert' bind arguments (e.g. '--bind name:s:value --bind count:i:42')"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleInsertContent)
}

func handleInsertContent(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleInsertContent")
	defer func() { logger.Tracef(ctx, "/handleInsertContent") }()

	uri, err := request.RequireString("uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	values, err := request.RequireString("values")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("content insert --uri %s %s", shellQuote(uri), values)
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("content insert: %v", err)), nil
	}

	if out == "" {
		out = fmt.Sprintf("inserted into %s", uri)
	}

	return mcp.NewToolResultText(out), nil
}
