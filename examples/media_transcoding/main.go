// Query media transcoding service status and media metrics session IDs.
//
// Uses the MediaTranscodingService proxy to query active client count,
// and the MediaMetricsManager proxy to obtain session IDs for
// transcoding, playback, recording, editing, and bundle operations.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/media_transcoding ./examples/media_transcoding/
//	adb push media_transcoding /data/local/tmp/ && adb shell /data/local/tmp/media_transcoding
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/media"
	"github.com/AndroidGoLab/binder/android/media/metrics"
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

	fmt.Println("=== Media Transcoding & Metrics ===")
	fmt.Println()

	// 1. MediaTranscodingService — active client count
	fmt.Println("-- Transcoding Service --")
	svc, err := sm.GetService(ctx, servicemanager.MediaTranscodingService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get media_transcoding service: %v\n", err)
		os.Exit(1)
	}

	tc := media.NewMediaTranscodingServiceProxy(svc)

	numClients, err := tc.GetNumOfClients(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetNumOfClients: %v\n", err)
	} else {
		fmt.Printf("Active transcoding clients: %d\n", numClients)
	}

	// 2. MediaMetricsManager — obtain session IDs for various media operations
	fmt.Println()
	fmt.Println("-- Media Metrics Session IDs --")

	mm, err := metrics.GetMediaMetricsManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get media_metrics service: %v\n", err)
		return
	}

	type sessionQuery struct {
		name string
		fn   func(context.Context) (string, error)
	}

	queries := []sessionQuery{
		{"Transcoding", mm.GetTranscodingSessionId},
		{"Playback", mm.GetPlaybackSessionId},
		{"Recording", mm.GetRecordingSessionId},
		{"Editing", mm.GetEditingSessionId},
		{"Bundle", mm.GetBundleSessionId},
	}

	for _, q := range queries {
		sid, err := q.fn(ctx)
		if err != nil {
			fmt.Printf("  %-14s error: %v\n", q.name+":", err)
		} else {
			if sid == "" {
				sid = "(empty)"
			}
			fmt.Printf("  %-14s %s\n", q.name+":", sid)
		}
	}

	// Release one of the sessions to demonstrate cleanup
	if transID, err := mm.GetTranscodingSessionId(ctx); err == nil && transID != "" {
		err = mm.ReleaseSessionId(ctx, transID)
		if err != nil {
			fmt.Printf("\nReleaseSessionId(%s): %v\n", transID, err)
		} else {
			fmt.Printf("\nReleased transcoding session: %s\n", transID)
		}
	}
}
