// Access the clipboard service to check clipboard state.
//
// Queries the "clipboard" service to check whether the clipboard
// has a primary clip and whether it has text content.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/clipboard_monitor ./examples/clipboard_monitor/
//	adb push build/clipboard_monitor /data/local/tmp/ && adb shell /data/local/tmp/clipboard_monitor
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/content"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func main() {
	ctx := context.Background()

	driver, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, driver, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)

	svc, err := sm.GetService(ctx, servicemanager.ClipboardService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get clipboard service: %v\n", err)
		os.Exit(1)
	}

	cb := content.NewClipboardProxy(svc)

	// Default device ID (0) for the primary display.
	const defaultDeviceID int32 = 0

	hasPrimary, err := cb.HasPrimaryClip(ctx, defaultDeviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HasPrimaryClip: %v\n", err)
	} else {
		fmt.Printf("Has primary clip: %v\n", hasPrimary)
	}

	hasText, err := cb.HasClipboardText(ctx, defaultDeviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HasClipboardText: %v\n", err)
	} else {
		fmt.Printf("Has clipboard text: %v\n", hasText)
	}

	source, err := cb.GetPrimaryClipSource(ctx, defaultDeviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetPrimaryClipSource: %v\n", err)
	} else {
		if source == "" {
			source = "(none)"
		}
		fmt.Printf("Primary clip source: %s\n", source)
	}
}
