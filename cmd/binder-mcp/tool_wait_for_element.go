//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultWaitTimeoutSec = 10
	waitPollIntervalMS    = 500
)

func registerWaitForElement(s *server.MCPServer) {
	tool := mcp.NewTool("wait_for_element",
		mcp.WithDescription(
			"Wait until a UI element matching the given criteria appears on screen. "+
				"Polls the UI hierarchy periodically until the element is found or timeout expires. "+
				"Returns the matched element(s) or an error on timeout.",
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
		mcp.WithNumber("timeout_seconds",
			mcp.Description("Maximum wait time in seconds (default: 10)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleWaitForElement)
}

func handleWaitForElement(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleWaitForElement")
	defer func() { logger.Tracef(ctx, "/handleWaitForElement") }()

	textFilter := request.GetString("text", "")
	resourceIDFilter := request.GetString("resource_id", "")
	contentDescFilter := request.GetString("content_desc", "")
	timeout := request.GetInt("timeout_seconds", defaultWaitTimeoutSec)

	if textFilter == "" && resourceIDFilter == "" && contentDescFilter == "" {
		return mcp.NewToolResultError("at least one search filter is required (text, resource_id, or content_desc)"), nil
	}

	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		xmlData, err := dumpUIXML()
		if err == nil {
			elements := parseAndFilterUI(xmlData, textFilter, resourceIDFilter, contentDescFilter, "")
			if len(elements) > 0 {
				data, err := json.Marshal(elements)
				if err != nil {
					return nil, fmt.Errorf("marshaling UI elements: %w", err)
				}
				return mcp.NewToolResultText(string(data)), nil
			}
		}

		if time.Now().After(deadline) {
			return mcp.NewToolResultError(fmt.Sprintf(
				"timeout after %ds: element not found (text=%q, resource_id=%q, content_desc=%q)",
				timeout, textFilter, resourceIDFilter, contentDescFilter,
			)), nil
		}

		select {
		case <-ctx.Done():
			return mcp.NewToolResultError("cancelled"), nil
		case <-time.After(waitPollIntervalMS * time.Millisecond):
		}
	}
}
