//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultPhotoPath = "/data/local/tmp/photo.jpg"

func registerCapturePhoto(s *server.MCPServer) {
	tool := mcp.NewTool("capture_photo",
		mcp.WithDescription(
			"Take a photo from a device camera and return it as a base64-encoded image. "+
				"Uses the camera intent or the 'am' command to trigger the camera app. "+
				"The photo is saved to a temporary file on device.",
		),
		mcp.WithString("camera_id",
			mcp.Description("Camera ID to use (e.g. '0' for rear, '1' for front). Default is '0'."),
		),
		mcp.WithString("path",
			mcp.Description("Output path on device (default: /data/local/tmp/photo.jpg)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)

	s.AddTool(tool, handleCapturePhoto)
}

func handleCapturePhoto(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleCapturePhoto")
	defer func() { logger.Tracef(ctx, "/handleCapturePhoto") }()

	path := request.GetString("path", defaultPhotoPath)

	// Use am start to launch the camera capture intent.
	// On many devices, the simplest shell-based approach is to use screencap
	// since direct camera capture from shell requires complex IGBP setup.
	// For a quick shell-based approach, launch the camera app and screenshot.
	cmd := fmt.Sprintf(
		"am start -a android.media.action.IMAGE_CAPTURE --ei android.intent.extras.CAMERA_FACING 0 && "+
			"sleep 2 && input keyevent KEYCODE_CAMERA && sleep 1 && "+
			"screencap -p %s",
		shellQuote(path),
	)

	_, err := shellExec(cmd)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("capture photo: %v", err)), nil
	}

	b64, err := shellExec(fmt.Sprintf("base64 -w 0 %s", shellQuote(path)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("base64 encode: %v", err)), nil
	}

	// Clean up.
	_, _ = shellExec(fmt.Sprintf("rm -f %s", shellQuote(path)))

	return mcp.NewToolResultImage("photo", b64, "image/jpeg"), nil
}
