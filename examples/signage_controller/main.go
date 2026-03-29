// Display brightness and power control for digital signage.
//
// Queries display state, brightness levels, and power mode for
// controlling signage displays.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/signage_controller ./examples/signage_controller/
//	adb push build/signage_controller /data/local/tmp/ && adb shell /data/local/tmp/signage_controller
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/hardware/display"
	genOs "github.com/AndroidGoLab/binder/android/os"
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

	fmt.Println("=== Signage Controller ===")
	fmt.Println()

	// Power state — is the screen on?
	power, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PowerManager: %v\n", err)
	} else {
		interactive, err := power.IsInteractive(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsInteractive: %v\n", err)
		} else {
			fmt.Printf("Screen power: %v\n", map[bool]string{true: "ON", false: "OFF"}[interactive])
		}
	}
	fmt.Println()

	// Display manager — enumerate displays and read brightness.
	fmt.Println("--- Displays ---")
	dispSvc, err := sm.GetService(ctx, servicemanager.DisplayService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DisplayManager: %v\n", err)
		os.Exit(1)
	}

	dm := display.NewDisplayManagerProxy(dispSvc)
	ids, err := dm.GetDisplayIds(ctx, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDisplayIds: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected displays: %d\n", len(ids))
	for _, id := range ids {
		fmt.Printf("\n  Display %d:\n", id)

		brightness, err := dm.GetBrightness(ctx, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    Brightness: %v\n", err)
		} else {
			fmt.Printf("    Brightness: %.2f\n", brightness)
		}

		brightnessInfo, err := dm.GetBrightnessInfo(ctx, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    BrightnessInfo: %v\n", err)
		} else {
			fmt.Printf("    BrightnessInfo: %+v\n", brightnessInfo)
		}
	}
	fmt.Println()

	// Color display — night mode status.
	colorSvc, err := sm.GetService(ctx, servicemanager.ColorDisplayService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ColorDisplayManager: %v\n", err)
	} else {
		cdm := display.NewColorDisplayManagerProxy(colorSvc)

		nightMode, err := cdm.IsNightDisplayActivated(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsNightDisplayActivated: %v\n", err)
		} else {
			fmt.Printf("Night mode:    %v\n", nightMode)
		}

		colorMode, err := cdm.GetColorMode(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetColorMode: %v\n", err)
		} else {
			fmt.Printf("Color mode:    %d\n", colorMode)
		}
	}

	fmt.Println()
	fmt.Println("=== End Signage Controller ===")
}
