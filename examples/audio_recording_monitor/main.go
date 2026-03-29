// Detect which apps are currently recording audio.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/audio_recording_monitor ./examples/audio_recording_monitor/
//	adb push audio_recording_monitor /data/local/tmp/ && adb shell /data/local/tmp/audio_recording_monitor
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/media"
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

	svc, err := sm.GetService(ctx, servicemanager.AudioService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get audio service: %v\n", err)
		os.Exit(1)
	}

	audio := media.NewAudioServiceProxy(svc)

	// Query active recording configurations.
	recordings, err := audio.GetActiveRecordingConfigurations(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetActiveRecordingConfigurations: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Active audio recordings: %d\n", len(recordings))
	if len(recordings) == 0 {
		fmt.Println("No apps are currently recording audio.")
	}

	// Also query active playback.
	playbacks, err := audio.GetActivePlaybackConfigurations(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetActivePlaybackConfigurations: %v\n", err)
	} else {
		fmt.Printf("Active audio playbacks: %d\n", len(playbacks))
	}
}
