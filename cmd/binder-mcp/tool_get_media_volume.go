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

const audioServiceDescriptor = "android.media.IAudioService"

// AudioStreamType enumerates the standard Android audio stream types.
type AudioStreamType int32

const (
	AudioStreamVoiceCall    AudioStreamType = 0
	AudioStreamSystem       AudioStreamType = 1
	AudioStreamRing         AudioStreamType = 2
	AudioStreamMusic        AudioStreamType = 3
	AudioStreamAlarm        AudioStreamType = 4
	AudioStreamNotification AudioStreamType = 5
)

// StreamVolumeInfo describes the volume state of one audio stream.
type StreamVolumeInfo struct {
	Stream  string `json:"stream"`
	Volume  int32  `json:"volume"`
	Max     int32  `json:"max"`
	Percent int32  `json:"percent"`
	Error   string `json:"error,omitempty"`
}

func (ts *ToolSet) registerGetMediaVolume(s *server.MCPServer) {
	tool := mcp.NewTool("get_media_volume",
		mcp.WithDescription(
			"Get the current volume level for all audio streams (voice, system, "+
				"ring, music, alarm, notification) using IAudioService. "+
				"Returns volume, max, and percentage for each stream.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleGetMediaVolume)
}

var streamNames = map[AudioStreamType]string{
	AudioStreamVoiceCall:    "voice_call",
	AudioStreamSystem:       "system",
	AudioStreamRing:         "ring",
	AudioStreamMusic:        "music",
	AudioStreamAlarm:        "alarm",
	AudioStreamNotification: "notification",
}

func (ts *ToolSet) handleGetMediaVolume(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetMediaVolume")
	defer func() { logger.Tracef(ctx, "/handleGetMediaVolume") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("audio"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("audio service unavailable"), nil
	}

	getVolCode, err := svc.ResolveCode(ctx, audioServiceDescriptor, "getStreamVolume")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getStreamVolume: %v", err)), nil
	}

	getMaxCode, err := svc.ResolveCode(ctx, audioServiceDescriptor, "getStreamMaxVolume")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getStreamMaxVolume: %v", err)), nil
	}

	streams := []AudioStreamType{
		AudioStreamVoiceCall,
		AudioStreamSystem,
		AudioStreamRing,
		AudioStreamMusic,
		AudioStreamAlarm,
		AudioStreamNotification,
	}

	results := make([]StreamVolumeInfo, 0, len(streams))
	for _, st := range streams {
		info := StreamVolumeInfo{Stream: streamNames[st]}

		vol, err := transactStreamInt32(ctx, svc, getVolCode, int32(st))
		if err != nil {
			info.Error = fmt.Sprintf("getStreamVolume: %v", err)
			results = append(results, info)
			continue
		}

		maxVol, err := transactStreamInt32(ctx, svc, getMaxCode, int32(st))
		if err != nil {
			info.Error = fmt.Sprintf("getStreamMaxVolume: %v", err)
			results = append(results, info)
			continue
		}

		info.Volume = vol
		info.Max = maxVol
		if maxVol > 0 {
			info.Percent = (vol * 100) / maxVol
		}
		results = append(results, info)
	}

	out, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("marshaling volume info: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}

// transactStreamInt32 sends a getStreamVolume/getStreamMaxVolume transaction
// for the given stream type and returns the int32 result.
func transactStreamInt32(
	ctx context.Context,
	svc binder.IBinder,
	code binder.TransactionCode,
	streamType int32,
) (int32, error) {
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(audioServiceDescriptor)
	data.WriteInt32(streamType)

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return 0, err
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return 0, err
	}

	return reply.ReadInt32()
}
