// Query input devices from InputManager.
//
// Uses the "input" service to enumerate all input device IDs and
// query details for each device (name, vendor, product info).
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/input_injector ./examples/input_injector/
//	adb push build/input_injector /data/local/tmp/ && adb shell /data/local/tmp/input_injector
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/hardware/input"
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

	svc, err := sm.GetService(ctx, servicemanager.InputService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get input service: %v\n", err)
		os.Exit(1)
	}

	im := input.NewInputManagerProxy(svc)

	// List all input device IDs.
	ids, err := im.GetInputDeviceIds(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetInputDeviceIds: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d input devices:\n\n", len(ids))

	for _, id := range ids {
		dev, err := im.GetInputDevice(ctx, id)
		if err != nil {
			fmt.Printf("  [id=%d] error: %v\n", id, err)
			continue
		}
		fmt.Printf("  [id=%d] %s\n", id, dev.Name)
	}

	// Query mouse pointer speed.
	speed, err := im.GetMousePointerSpeed(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGetMousePointerSpeed: %v\n", err)
	} else {
		fmt.Printf("\nMouse pointer speed: %d\n", speed)
	}
}
