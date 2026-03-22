//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// BrightnessResult holds the get_brightness response.
type BrightnessResult struct {
	Brightness float32 `json:"brightness"`
	Error      string  `json:"error,omitempty"`
}

func (ts *ToolSet) registerGetBrightness(s *server.MCPServer) {
	tool := mcp.NewTool("get_brightness",
		mcp.WithDescription(
			"Get the current display brightness level (0.0 to 1.0) from "+
				"IDisplayManager.getBrightness().",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleGetBrightness)
}

func (ts *ToolSet) handleGetBrightness(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetBrightness")
	defer func() { logger.Tracef(ctx, "/handleGetBrightness") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("display"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("display service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, displayManagerDescriptor, "getBrightness")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getBrightness: %v", err)), nil
	}

	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(displayManagerDescriptor)
	// getBrightness(int displayId) -- use display 0 (default).
	data.WriteInt32(0)

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getBrightness: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getBrightness status: %v", err)), nil
	}

	brightness, err := reply.ReadFloat32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading brightness: %v", err)), nil
	}

	out, err := json.Marshal(BrightnessResult{Brightness: brightness})
	if err != nil {
		return nil, fmt.Errorf("marshaling brightness: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}
