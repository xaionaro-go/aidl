//go:build linux

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func (ts *ToolSet) registerWakeScreen(s *server.MCPServer) {
	tool := mcp.NewTool("wake_screen",
		mcp.WithDescription(
			"Wake the device screen using IPowerManager.wakeUp(). "+
				"Turns the screen on if it is currently off.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleWakeScreen)
}

func (ts *ToolSet) handleWakeScreen(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleWakeScreen")
	defer func() { logger.Tracef(ctx, "/handleWakeScreen") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("power"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("power service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, powerManagerDescriptor, "wakeUp")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving wakeUp: %v", err)), nil
	}

	// wakeUp(long time, int reason, String details, String opPackageName)
	// reason=0 (WAKE_REASON_UNKNOWN), details="binder-mcp"
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(powerManagerDescriptor)
	data.WriteInt64(time.Now().UnixMilli())
	data.WriteInt32(0)
	data.WriteString16("binder-mcp")
	data.WriteString16("com.android.shell")

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("wakeUp: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("wakeUp status: %v", err)), nil
	}

	return mcp.NewToolResultText("screen woken"), nil
}
