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

const packageManagerDescriptor = "android.content.pm.IPackageManager"

func (ts *ToolSet) registerListPackages(s *server.MCPServer) {
	tool := mcp.NewTool("list_packages",
		mcp.WithDescription(
			"List all installed package names on the device using "+
				"IPackageManager.getAllPackages(). Returns a JSON array of strings.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleListPackages)
}

func (ts *ToolSet) handleListPackages(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleListPackages")
	defer func() { logger.Tracef(ctx, "/handleListPackages") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("package"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("package service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, packageManagerDescriptor, "getAllPackages")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getAllPackages: %v", err)), nil
	}

	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(packageManagerDescriptor)

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getAllPackages: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getAllPackages status: %v", err)), nil
	}

	count, err := reply.ReadInt32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading count: %v", err)), nil
	}

	const maxPackages = 100000
	if count < 0 || count > maxPackages {
		return mcp.NewToolResultError(fmt.Sprintf("invalid package count: %d", count)), nil
	}

	packages := make([]string, 0, count)
	for i := int32(0); i < count; i++ {
		name, err := reply.ReadString16()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("reading package %d: %v", i, err)), nil
		}
		packages = append(packages, name)
	}

	out, err := json.Marshal(packages)
	if err != nil {
		return nil, fmt.Errorf("marshaling package list: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}
