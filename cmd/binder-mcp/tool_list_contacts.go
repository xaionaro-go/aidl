//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListContacts(s *server.MCPServer) {
	tool := mcp.NewTool("list_contacts",
		mcp.WithDescription(
			"List contacts from the device contacts provider using 'content query'. "+
				"Returns display names from the ContactsContract.Contacts URI.",
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of contacts to return (default: 50)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleListContacts)
}

func handleListContacts(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListContacts")
	defer func() { logger.Tracef(ctx, "/handleListContacts") }()

	limit := request.GetInt("limit", 50)

	cmd := fmt.Sprintf(
		"content query --uri content://com.android.contacts/contacts --projection display_name | head -n %d",
		limit,
	)
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list contacts: %v", err)), nil
	}

	if out == "" {
		out = "no contacts found"
	}

	return mcp.NewToolResultText(out), nil
}
