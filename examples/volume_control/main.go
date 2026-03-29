// Get and set stream volumes via AudioService.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/volume_control ./examples/volume_control/
//	adb push volume_control /data/local/tmp/ && adb shell /data/local/tmp/volume_control
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

const (
	streamMusic = int32(3)
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

	// Read current volume for music stream.
	vol, err := audio.GetStreamVolume(ctx, streamMusic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetStreamVolume: %v\n", err)
		os.Exit(1)
	}
	maxVol, err := audio.GetStreamMaxVolume(ctx, streamMusic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetStreamMaxVolume: %v\n", err)
		os.Exit(1)
	}
	minVol, err := audio.GetStreamMinVolume(ctx, streamMusic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetStreamMinVolume: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Music stream volume: %d (min=%d, max=%d)\n", vol, minVol, maxVol)

	// Attempt to set volume to current value (safe no-op).
	err = audio.SetStreamVolume(ctx, streamMusic, vol, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SetStreamVolume: %v\n", err)
	} else {
		fmt.Printf("SetStreamVolume(%d) succeeded (no-op write-back)\n", vol)
	}

	// Check ringer mode.
	ringer, err := audio.GetRingerModeExternal(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetRingerModeExternal: %v\n", err)
	} else {
		ringerNames := map[int32]string{0: "silent", 1: "vibrate", 2: "normal"}
		name := ringerNames[ringer]
		if name == "" {
			name = "unknown"
		}
		fmt.Printf("Ringer mode: %s (%d)\n", name, ringer)
	}

	// Master mute status.
	masterMuted, err := audio.IsMasterMute(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsMasterMute: %v\n", err)
	} else {
		fmt.Printf("Master mute: %v\n", masterMuted)
	}
}
