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

// CurrentAppResult holds the get_current_app response.
type CurrentAppResult struct {
	Package  string `json:"package"`
	Activity string `json:"activity"`
	Error    string `json:"error,omitempty"`
}

func registerGetCurrentApp(s *server.MCPServer) {
	tool := mcp.NewTool("get_current_app",
		mcp.WithDescription(
			"Get the currently focused app package and activity name. "+
				"Uses dumpsys to parse the focused activity.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetCurrentApp)
}

func handleGetCurrentApp(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetCurrentApp")
	defer func() { logger.Tracef(ctx, "/handleGetCurrentApp") }()

	// dumpsys activity activities prints the focused activity in a line like:
	//   topResumedActivity=ActivityRecord{abc u0 com.android.chrome/.Main t5}
	//   ResumedActivity: ActivityRecord{abc u0 com.android.chrome/.Main t5}
	out, err := shellExec("dumpsys activity activities | grep -E 'topResumedActivity=|ResumedActivity:|mResumedActivity:' | head -1")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("dumpsys: %v", err)), nil
	}

	result := parseFocusedActivity(out)

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling current app: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

// parseFocusedActivity extracts package and activity from a dumpsys line like:
// mResumedActivity: ActivityRecord{abc123 u0 com.foo/.BarActivity t5}
func parseFocusedActivity(line string) CurrentAppResult {
	line = strings.TrimSpace(line)
	if line == "" {
		return CurrentAppResult{Error: "no focused activity found"}
	}

	// Find the component name in the ActivityRecord.
	// Format: ... u0 com.foo/.BarActivity t5}
	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.Contains(p, "/") && !strings.HasPrefix(p, "{") {
			// Strip trailing "}" or "t123}"
			component := strings.TrimRight(p, "}")
			slashIdx := strings.Index(component, "/")
			if slashIdx < 0 {
				continue
			}
			pkg := component[:slashIdx]
			activity := component
			return CurrentAppResult{Package: pkg, Activity: activity}
		}
	}

	return CurrentAppResult{Error: fmt.Sprintf("cannot parse focused activity from: %s", line)}
}
