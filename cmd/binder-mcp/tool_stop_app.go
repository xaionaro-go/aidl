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

const activityManagerDescriptor = "android.app.IActivityManager"

func (ts *ToolSet) registerStopApp(s *server.MCPServer) {
	tool := mcp.NewTool("stop_app",
		mcp.WithDescription(
			"Force-stop an app by package name using "+
				"IActivityManager.forceStopPackage().",
		),
		mcp.WithString("package",
			mcp.Required(),
			mcp.Description("Package name to stop (e.g. 'com.android.chrome')"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleStopApp)
}

func (ts *ToolSet) handleStopApp(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleStopApp")
	defer func() { logger.Tracef(ctx, "/handleStopApp") }()

	pkg, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("activity"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("activity service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, activityManagerDescriptor, "forceStopPackage")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving forceStopPackage: %v", err)), nil
	}

	// forceStopPackage(String packageName, int userId)
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(activityManagerDescriptor)
	data.WriteString16(pkg)
	data.WriteInt32(0) // userId

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("forceStopPackage: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("forceStopPackage status: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("stopped %s", pkg)), nil
}
