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

func (ts *ToolSet) registerSetMediaVolume(s *server.MCPServer) {
	tool := mcp.NewTool("set_media_volume",
		mcp.WithDescription(
			"Set the volume for a given audio stream using "+
				"IAudioService.setStreamVolume(). "+
				"Stream types: 0=voice_call, 1=system, 2=ring, 3=music, "+
				"4=alarm, 5=notification.",
		),
		mcp.WithNumber("stream",
			mcp.Required(),
			mcp.Description("Stream type (0-5): 0=voice, 1=system, 2=ring, 3=music, 4=alarm, 5=notification"),
		),
		mcp.WithNumber("volume",
			mcp.Required(),
			mcp.Description("Volume index (0 to max for the stream)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleSetMediaVolume)
}

func (ts *ToolSet) handleSetMediaVolume(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleSetMediaVolume")
	defer func() { logger.Tracef(ctx, "/handleSetMediaVolume") }()

	stream := int32(request.GetInt("stream", -1))
	volume := int32(request.GetInt("volume", -1))

	if stream < 0 || stream > 5 {
		return mcp.NewToolResultError("stream must be 0-5"), nil
	}
	if volume < 0 {
		return mcp.NewToolResultError("volume must be >= 0"), nil
	}

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("audio"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("audio service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, audioServiceDescriptor, "setStreamVolume")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving setStreamVolume: %v", err)), nil
	}

	// setStreamVolume(int streamType, int index, int flags, String callingPackage)
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(audioServiceDescriptor)
	data.WriteInt32(stream)
	data.WriteInt32(volume)
	data.WriteInt32(0) // flags
	data.WriteString16("com.android.shell")

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("setStreamVolume: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("setStreamVolume status: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("stream %d volume set to %d", stream, volume)), nil
}
