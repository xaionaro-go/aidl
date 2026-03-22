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

// AppInstalledResult holds the is_app_installed response.
type AppInstalledResult struct {
	Installed bool   `json:"installed"`
	Package   string `json:"package"`
}

func registerIsAppInstalled(s *server.MCPServer) {
	tool := mcp.NewTool("is_app_installed",
		mcp.WithDescription(
			"Check if a specific package is installed on the device using 'pm list packages'.",
		),
		mcp.WithString("package",
			mcp.Required(),
			mcp.Description("Package name (e.g. com.example.app)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleIsAppInstalled)
}

func handleIsAppInstalled(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleIsAppInstalled")
	defer func() { logger.Tracef(ctx, "/handleIsAppInstalled") }()

	pkg, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("pm list packages %s", shellQuote(pkg))
	out, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pm list packages: %v", err)), nil
	}

	// pm list packages outputs "package:com.example.app" for exact matches.
	installed := strings.Contains(out, "package:"+pkg)

	result := AppInstalledResult{
		Installed: installed,
		Package:   pkg,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}
