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
	defaultScrollMaxAttempts = 10
	scrollPauseMS            = 300
)

func registerScrollToElement(s *server.MCPServer) {
	tool := mcp.NewTool("scroll_to_element",
		mcp.WithDescription(
			"Scroll the screen until a target UI element is found. "+
				"Repeatedly swipes in the given direction and checks the UI hierarchy. "+
				"Returns the found element or an error after max_attempts.",
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
		mcp.WithString("direction",
			mcp.Description("Scroll direction: up, down, left, right (default: down)"),
			mcp.Enum("up", "down", "left", "right"),
		),
		mcp.WithNumber("max_attempts",
			mcp.Description("Maximum number of scroll attempts (default: 10)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleScrollToElement)
}

func handleScrollToElement(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleScrollToElement")
	defer func() { logger.Tracef(ctx, "/handleScrollToElement") }()

	textFilter := request.GetString("text", "")
	resourceIDFilter := request.GetString("resource_id", "")
	contentDescFilter := request.GetString("content_desc", "")
	direction := request.GetString("direction", "down")
	maxAttempts := request.GetInt("max_attempts", defaultScrollMaxAttempts)

	if textFilter == "" && resourceIDFilter == "" && contentDescFilter == "" {
		return mcp.NewToolResultError("at least one search filter is required (text, resource_id, or content_desc)"), nil
	}

	// Get display size for scroll coordinates.
	sizeOut, err := shellExec("wm size")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("wm size: %v", err)), nil
	}

	cx, cy := 540, 960 // Default fallback.
	fmt.Sscanf(sizeOut, "Physical size: %dx%d", &cx, &cy)
	cx /= 2
	cy /= 2

	// Compute swipe coordinates for the scroll direction.
	var x1, y1, x2, y2 int
	switch direction {
	case "down":
		x1, y1, x2, y2 = cx, cy+cy/2, cx, cy-cy/2
	case "up":
		x1, y1, x2, y2 = cx, cy-cy/2, cx, cy+cy/2
	case "left":
		x1, y1, x2, y2 = cx+cx/2, cy, cx-cx/2, cy
	case "right":
		x1, y1, x2, y2 = cx-cx/2, cy, cx+cx/2, cy
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported direction: %s", direction)), nil
	}

	for attempt := range maxAttempts {
		xmlData, xmlErr := dumpUIXML()
		if xmlErr == nil {
			elements := parseAndFilterUI(xmlData, textFilter, resourceIDFilter, contentDescFilter, "")
			if len(elements) > 0 {
				data, err := json.Marshal(elements)
				if err != nil {
					return nil, fmt.Errorf("marshaling UI elements: %w", err)
				}
				return mcp.NewToolResultText(string(data)), nil
			}
		}

		if attempt == maxAttempts-1 {
			break
		}

		// Scroll.
		swipeCmd := fmt.Sprintf("input swipe %d %d %d %d %d", x1, y1, x2, y2, defaultSwipeDurationMS)
		if _, err := shellExec(swipeCmd); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("swipe: %v", err)), nil
		}

		select {
		case <-ctx.Done():
			return mcp.NewToolResultError("cancelled"), nil
		case <-time.After(scrollPauseMS * time.Millisecond):
		}
	}

	return mcp.NewToolResultError(fmt.Sprintf(
		"element not found after %d scroll attempts (text=%q, resource_id=%q, content_desc=%q)",
		maxAttempts, textFilter, resourceIDFilter, contentDescFilter,
	)), nil
}
