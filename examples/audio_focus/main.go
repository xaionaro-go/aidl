// Query current audio focus state via AudioService.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/audio_focus ./examples/audio_focus/
//	adb push audio_focus /data/local/tmp/ && adb shell /data/local/tmp/audio_focus
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

	// Query current audio focus state.
	// Returns: 0 = AUDIOFOCUS_NONE, 1 = AUDIOFOCUS_GAIN, etc.
	focus, err := audio.GetCurrentAudioFocus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCurrentAudioFocus: %v\n", err)
		os.Exit(1)
	}

	focusNames := map[int32]string{
		0: "AUDIOFOCUS_NONE",
		1: "AUDIOFOCUS_GAIN",
	}
	name := focusNames[focus]
	if name == "" {
		name = fmt.Sprintf("unknown(%d)", focus)
	}
	fmt.Printf("Current audio focus: %s\n", name)

	// Query audio mode (normal, ringtone, in-call, etc.)
	mode, err := audio.GetMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMode: %v\n", err)
	} else {
		modeNames := map[int32]string{
			0: "NORMAL",
			1: "RINGTONE",
			2: "IN_CALL",
			3: "IN_COMMUNICATION",
		}
		modeName := modeNames[mode]
		if modeName == "" {
			modeName = fmt.Sprintf("unknown(%d)", mode)
		}
		fmt.Printf("Audio mode: %s\n", modeName)
	}

	// Check hotword stream support (lookbackAudio=false).
	hotword, err := audio.IsHotwordStreamSupported(ctx, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsHotwordStreamSupported: %v\n", err)
	} else {
		fmt.Printf("Hotword stream supported: %v\n", hotword)
	}

	// Check speakerphone state.
	speaker, err := audio.IsSpeakerphoneOn(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsSpeakerphoneOn: %v\n", err)
	} else {
		fmt.Printf("Speakerphone on: %v\n", speaker)
	}
}
