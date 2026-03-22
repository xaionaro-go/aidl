//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NotificationEntry describes one active notification parsed from dumpsys.
type NotificationEntry struct {
	Key     string `json:"key"`
	Package string `json:"package,omitempty"`
	Title   string `json:"title,omitempty"`
	Text    string `json:"text,omitempty"`
}

func registerListNotifications(s *server.MCPServer) {
	tool := mcp.NewTool("list_notifications",
		mcp.WithDescription(
			"List active notifications on the device. Parses output from "+
				"'dumpsys notification --noredact'. Returns package and key for each.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListNotifications)
}

func handleListNotifications(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListNotifications")
	defer func() { logger.Tracef(ctx, "/handleListNotifications") }()

	out, err := shellExec("dumpsys notification --noredact 2>/dev/null | grep -A 2 'NotificationRecord' | head -200")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("dumpsys notification: %v", err)), nil
	}

	entries := parseNotifications(out)

	data, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("marshaling notifications: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

// parseNotifications extracts notification entries from dumpsys output.
// Lines look like: "  NotificationRecord(0x... | ... pkg=com.foo key=0|com.foo|...)"
func parseNotifications(output string) []NotificationEntry {
	var entries []NotificationEntry

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "NotificationRecord") {
			continue
		}

		entry := NotificationEntry{}

		// Extract "key=..." and "pkg=..." fields.
		if idx := strings.Index(line, "pkg="); idx >= 0 {
			rest := line[idx+4:]
			if sp := strings.IndexAny(rest, " )"); sp > 0 {
				entry.Package = rest[:sp]
			} else {
				entry.Package = rest
			}
		}

		if idx := strings.Index(line, "key="); idx >= 0 {
			rest := line[idx+4:]
			if sp := strings.IndexAny(rest, " )"); sp > 0 {
				entry.Key = rest[:sp]
			} else {
				entry.Key = rest
			}
		}

		if entry.Key != "" || entry.Package != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}
