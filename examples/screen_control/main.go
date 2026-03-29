// Check screen on/off state and display interactivity via PowerManager.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/screen_control ./examples/screen_control/
//	adb push build/screen_control /data/local/tmp/ && adb shell /data/local/tmp/screen_control
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func main() {
	ctx := context.Background()

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer drv.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)

	power, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get power service: %v\n", err)
		os.Exit(1)
	}

	interactive, err := power.IsInteractive(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsInteractive: %v\n", err)
	} else {
		fmt.Printf("Screen on (interactive): %v\n", interactive)
	}

	// Check primary display (displayId = 0).
	displayInteractive, err := power.IsDisplayInteractive(ctx, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsDisplayInteractive(0): %v\n", err)
	} else {
		fmt.Printf("Display 0 interactive:   %v\n", displayInteractive)
	}

	ambient, err := power.IsAmbientDisplayAvailable(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsAmbientDisplayAvailable: %v\n", err)
	} else {
		fmt.Printf("Ambient display avail:   %v\n", ambient)
	}

	suppressed, err := power.IsAmbientDisplaySuppressed(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsAmbientDisplaySuppressed: %v\n", err)
	} else {
		fmt.Printf("Ambient suppressed:      %v\n", suppressed)
	}

	lastSleep, err := power.GetLastSleepReason(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetLastSleepReason: %v\n", err)
	} else {
		fmt.Printf("Last sleep reason:       %d\n", lastSleep)
	}
}
