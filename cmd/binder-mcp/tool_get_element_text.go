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

func registerGetElementText(s *server.MCPServer) {
	tool := mcp.NewTool("get_element_text",
		mcp.WithDescription(
			"Extract the text content from a specific UI element found by "+
				"resource-id or content-desc. Returns the text attribute of all matching elements.",
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

	s.AddTool(tool, handleGetElementText)
}

func handleGetElementText(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetElementText")
	defer func() { logger.Tracef(ctx, "/handleGetElementText") }()

	resourceIDFilter := request.GetString("resource_id", "")
	contentDescFilter := request.GetString("content_desc", "")

	if resourceIDFilter == "" && contentDescFilter == "" {
		return mcp.NewToolResultError("at least one filter is required (resource_id or content_desc)"), nil
	}

	xmlData, err := dumpUIXML()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("dumping UI: %v", err)), nil
	}

	elements := parseAndFilterUI(xmlData, "", resourceIDFilter, contentDescFilter, "")

	if len(elements) == 0 {
		return mcp.NewToolResultError("no matching elements found"), nil
	}

	// Return just the text values.
	type TextResult struct {
		ResourceID string `json:"resource_id,omitempty"`
		Text       string `json:"text"`
	}

	results := make([]TextResult, 0, len(elements))
	for _, e := range elements {
		results = append(results, TextResult{
			ResourceID: e.ResourceID,
			Text:       e.Text,
		})
	}

	data, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("marshaling text results: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}
