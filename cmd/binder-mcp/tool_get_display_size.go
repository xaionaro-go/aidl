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

const windowManagerDescriptor = "android.view.IWindowManager"

// DisplaySizeResult holds the get_display_size response.
type DisplaySizeResult struct {
	Width  int32  `json:"width"`
	Height int32  `json:"height"`
	Error  string `json:"error,omitempty"`
}

func (ts *ToolSet) registerGetDisplaySize(s *server.MCPServer) {
	tool := mcp.NewTool("get_display_size",
		mcp.WithDescription(
			"Get the initial display size (width x height in pixels) from "+
				"IWindowManager.getInitialDisplaySize().",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleGetDisplaySize)
}

func (ts *ToolSet) handleGetDisplaySize(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetDisplaySize")
	defer func() { logger.Tracef(ctx, "/handleGetDisplaySize") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("window"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("window service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, windowManagerDescriptor, "getInitialDisplaySize")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getInitialDisplaySize: %v", err)), nil
	}

	// getInitialDisplaySize(int displayId, out Point size)
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(windowManagerDescriptor)
	data.WriteInt32(0) // default display

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getInitialDisplaySize: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getInitialDisplaySize status: %v", err)), nil
	}

	// Reply contains a nullable Point parcel.
	nullInd, err := reply.ReadInt32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading null indicator: %v", err)), nil
	}

	if nullInd == 0 {
		return mcp.NewToolResultError("display size not available (null Point)"), nil
	}

	// Point is a flat parcelable: x(int32), y(int32).
	x, err := reply.ReadInt32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading width: %v", err)), nil
	}

	y, err := reply.ReadInt32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading height: %v", err)), nil
	}

	out, err := json.Marshal(DisplaySizeResult{Width: x, Height: y})
	if err != nil {
		return nil, fmt.Errorf("marshaling display size: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}
