//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerQueryContent(s *server.MCPServer) {
	tool := mcp.NewTool("query_content",
		mcp.WithDescription(
			"Query a content provider URI using 'content query'. "+
				"Returns rows from the specified content URI, optionally filtered by a WHERE clause.",
		),
		mcp.WithString("uri",
			mcp.Required(),
			mcp.Description("Content provider URI (e.g. content://settings/system)"),
		),
		mcp.WithString("projection",
			mcp.Description("Columns to select (comma-separated, e.g. 'name:value')"),
		),
		mcp.WithString("where",
			mcp.Description("WHERE clause (e.g. \"name='screen_brightness'\")"),
		),
		mcp.WithString("sort",
			mcp.Description("ORDER BY clause"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleQueryContent)
}

func handleQueryContent(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleQueryContent")
	defer func() { logger.Tracef(ctx, "/handleQueryContent") }()

	uri, err := request.RequireString("uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projection := request.GetString("projection", "")
	where := request.GetString("where", "")
	sort := request.GetString("sort", "")

	cmd := fmt.Sprintf("content query --uri %s", shellQuote(uri))
	if projection != "" {
		cmd += fmt.Sprintf(" --projection %s", shellQuote(projection))
	}
	if where != "" {
		cmd += fmt.Sprintf(" --where %s", shellQuote(where))
	}
	if sort != "" {
		cmd += fmt.Sprintf(" --sort %s", shellQuote(sort))
	}

	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("content query: %v", err)), nil
	}

	if out == "" {
		out = "no results"
	}

	return mcp.NewToolResultText(out), nil
}
