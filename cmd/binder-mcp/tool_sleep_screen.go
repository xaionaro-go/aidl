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

func (ts *ToolSet) registerSleepScreen(s *server.MCPServer) {
	tool := mcp.NewTool("sleep_screen",
		mcp.WithDescription(
			"Put the device screen to sleep using IPowerManager.goToSleep(). "+
				"Turns the screen off if it is currently on.",
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleSleepScreen)
}

func (ts *ToolSet) handleSleepScreen(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSleepScreen")
	defer func() { logger.Tracef(ctx, "/handleSleepScreen") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("power"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("power service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, powerManagerDescriptor, "goToSleep")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving goToSleep: %v", err)), nil
	}

	// goToSleep(long time, int reason, int flags)
	// reason=4 (GO_TO_SLEEP_REASON_POWER_BUTTON), flags=0
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(powerManagerDescriptor)
	data.WriteInt64(time.Now().UnixMilli())
	data.WriteInt32(4)
	data.WriteInt32(0)

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("goToSleep: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("goToSleep status: %v", err)), nil
	}

	return mcp.NewToolResultText("screen sleeping"), nil
}
