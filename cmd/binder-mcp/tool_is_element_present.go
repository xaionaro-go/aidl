//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ElementPresenceResult holds the is_element_present response.
type ElementPresenceResult struct {
	Present bool `json:"present"`
	Count   int  `json:"count"`
}

func registerIsElementPresent(s *server.MCPServer) {
	tool := mcp.NewTool("is_element_present",
		mcp.WithDescription(
			"Check whether a UI element matching the given criteria exists on the current screen. "+
				"Returns a boolean 'present' field and the count of matches.",
		),
		mcp.WithString("text",
			mcp.Description("Match elements whose text contains this substring (case-insensitive)"),
		),
		mcp.WithString("resource_id",
			mcp.Description("Match elements whose resource-id contains this substring"),
		),
		mcp.WithString("content_desc",
			mcp.Description("Match elements whose content-desc contains this substring (case-insensitive)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleIsElementPresent)
}

func handleIsElementPresent(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleIsElementPresent")
	defer func() { logger.Tracef(ctx, "/handleIsElementPresent") }()

	textFilter := request.GetString("text", "")
	resourceIDFilter := request.GetString("resource_id", "")
	contentDescFilter := request.GetString("content_desc", "")

	if textFilter == "" && resourceIDFilter == "" && contentDescFilter == "" {
		return mcp.NewToolResultError("at least one search filter is required (text, resource_id, or content_desc)"), nil
	}

	xmlData, err := dumpUIXML()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("dumping UI: %v", err)), nil
	}

	elements := parseAndFilterUI(xmlData, textFilter, resourceIDFilter, contentDescFilter, "")

	result := ElementPresenceResult{
		Present: len(elements) > 0,
		Count:   len(elements),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling presence result: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}
