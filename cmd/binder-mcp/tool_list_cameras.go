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

const cameraServiceDescriptor = "android.hardware.ICameraService"

// ListCamerasResult holds the list_cameras response.
type ListCamerasResult struct {
	Count int32  `json:"count"`
	Error string `json:"error,omitempty"`
}

func (ts *ToolSet) registerListCameras(s *server.MCPServer) {
	tool := mcp.NewTool("list_cameras",
		mcp.WithDescription(
			"Get the number of cameras on the device using "+
				"ICameraService.getNumberOfCameras().",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleListCameras)
}

func (ts *ToolSet) handleListCameras(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListCameras")
	defer func() { logger.Tracef(ctx, "/handleListCameras") }()

	// The camera service is registered as "media.camera".
	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("media.camera"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("camera service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, cameraServiceDescriptor, "getNumberOfCameras")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getNumberOfCameras: %v", err)), nil
	}

	// getNumberOfCameras(int type) -- type=0 for CAMERA_TYPE_BACKWARD_COMPATIBLE
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(cameraServiceDescriptor)
	data.WriteInt32(0)

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getNumberOfCameras: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getNumberOfCameras status: %v", err)), nil
	}

	count, err := reply.ReadInt32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading count: %v", err)), nil
	}

	out, err := json.Marshal(ListCamerasResult{Count: count})
	if err != nil {
		return nil, fmt.Errorf("marshaling camera count: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}
