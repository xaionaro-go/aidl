//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func (ts *ToolSet) registerSetBrightness(s *server.MCPServer) {
	tool := mcp.NewTool("set_brightness",
		mcp.WithDescription(
			"Set the display brightness level using IDisplayManager.setBrightness(). "+
				"Value is a float from 0.0 (off) to 1.0 (max).",
		),
		mcp.WithNumber("brightness",
			mcp.Required(),
			mcp.Description("Brightness level from 0.0 to 1.0"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleSetBrightness)
}

func (ts *ToolSet) handleSetBrightness(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetBrightness")
	defer func() { logger.Tracef(ctx, "/handleSetBrightness") }()

	brightness := float32(request.GetFloat("brightness", -1))
	if brightness < 0 || brightness > 1 {
		return mcp.NewToolResultError("brightness must be between 0.0 and 1.0"), nil
	}

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("display"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("display service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, displayManagerDescriptor, "setBrightness")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving setBrightness: %v", err)), nil
	}

	// setBrightness(int displayId, float brightness)
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(displayManagerDescriptor)
	data.WriteInt32(0)
	data.WriteFloat32(brightness)

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("setBrightness: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("setBrightness status: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("brightness set to %.2f", brightness)), nil
}
