// Query device rotation and display state via WindowManager and DisplayManager.
//
// The rotation resolver service is a bound Java service only available
// to the system server. Instead, this example queries the actual rotation
// state through WindowManager (GetDefaultDisplayRotation, IsRotationFrozen,
// HasNavigationBar, IsKeyguardLocked) and DisplayManager (display IDs,
// brightness).
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/rotation_resolver ./examples/rotation_resolver/
//	adb push build/rotation_resolver /data/local/tmp/ && adb shell /data/local/tmp/rotation_resolver
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/hardware/display"
	"github.com/AndroidGoLab/binder/android/view"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func rotationName(r int32) string {
	switch r {
	case 0:
		return "0 (portrait)"
	case 1:
		return "90 (landscape)"
	case 2:
		return "180 (reverse portrait)"
	case 3:
		return "270 (reverse landscape)"
	default:
		return fmt.Sprintf("%d (unknown)", r)
	}
}

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

	fmt.Println("=== Device Rotation & Display State ===")
	fmt.Println()

	// WindowManager — rotation and display properties
	wmSvc, err := sm.GetService(ctx, servicemanager.WindowService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get window service: %v\n", err)
		os.Exit(1)
	}

	wm := view.NewWindowManagerProxy(wmSvc)

	rotation, err := wm.GetDefaultDisplayRotation(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDefaultDisplayRotation: %v\n", err)
	} else {
		fmt.Printf("Display rotation:  %s\n", rotationName(rotation))
	}

	frozen, err := wm.IsRotationFrozen(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsRotationFrozen: %v\n", err)
	} else {
		fmt.Printf("Rotation frozen:   %v\n", frozen)
	}

	hasNav, err := wm.HasNavigationBar(ctx, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HasNavigationBar: %v\n", err)
	} else {
		fmt.Printf("Navigation bar:    %v\n", hasNav)
	}

	keyguard, err := wm.IsKeyguardLocked(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsKeyguardLocked: %v\n", err)
	} else {
		fmt.Printf("Keyguard locked:   %v\n", keyguard)
	}

	safeMode, err := wm.IsSafeModeEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsSafeModeEnabled: %v\n", err)
	} else {
		fmt.Printf("Safe mode:         %v\n", safeMode)
	}

	inTouch, err := wm.IsInTouchMode(ctx, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsInTouchMode: %v\n", err)
	} else {
		fmt.Printf("Touch mode:        %v\n", inTouch)
	}

	// DisplayManager — enumerate displays and query brightness
	fmt.Println()
	dmSvc, err := sm.GetService(ctx, servicemanager.DisplayService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get display service: %v\n", err)
		return
	}

	dm := display.NewDisplayManagerProxy(dmSvc)

	ids, err := dm.GetDisplayIds(ctx, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDisplayIds: %v\n", err)
	} else {
		fmt.Printf("Active displays: %d\n", len(ids))
		for _, id := range ids {
			brightness, err := dm.GetBrightness(ctx, id)
			if err != nil {
				fmt.Printf("  Display %d: brightness error (%v)\n", id, err)
			} else {
				fmt.Printf("  Display %d: brightness=%.2f\n", id, brightness)
			}
		}
	}
}
