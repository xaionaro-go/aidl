// Query activity/window manager for kiosk lockdown information.
//
// Checks lock task packages, screen density, display size, and
// process limits to assess kiosk mode readiness.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/kiosk_lockdown ./examples/kiosk_lockdown/
//	adb push build/kiosk_lockdown /data/local/tmp/ && adb shell /data/local/tmp/kiosk_lockdown
package main

import (
	"context"
	"fmt"
	"os"

	genAdmin "github.com/AndroidGoLab/binder/android/app/admin"
	"github.com/AndroidGoLab/binder/android/content"
	"github.com/AndroidGoLab/binder/android/view"
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
	callerPkg := binder.DefaultCallerIdentity().PackageName

	fmt.Println("=== Kiosk Lockdown Info ===")
	fmt.Println()

	// DevicePolicyManager: lock task packages
	dpm, err := genAdmin.GetDevicePolicyManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DevicePolicyManager: %v\n", err)
	} else {
		// Query lock task packages (requires admin component, use empty for query).
		emptyAdmin := content.ComponentName{}
		pkgs, err := dpm.GetLockTaskPackages(ctx, emptyAdmin, callerPkg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetLockTaskPackages: %v\n", err)
		} else {
			fmt.Printf("Lock task packages: %d\n", len(pkgs))
			for _, p := range pkgs {
				fmt.Printf("  - %s\n", p)
			}
		}

		features, err := dpm.GetLockTaskFeatures(ctx, emptyAdmin, callerPkg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetLockTaskFeatures: %v\n", err)
		} else {
			fmt.Printf("Lock task features: 0x%x\n", features)
		}

		scDisabled, err := dpm.GetScreenCaptureDisabled(ctx, emptyAdmin, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetScreenCaptureDisabled: %v\n", err)
		} else {
			fmt.Printf("Screen capture disabled: %v\n", scDisabled)
		}
	}
	fmt.Println()

	// WindowManager: display dimensions and density
	fmt.Println("--- Window Manager ---")
	wmSvc, err := sm.GetService(ctx, servicemanager.WindowService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WindowManager: %v\n", err)
	} else {
		wm := view.NewWindowManagerProxy(wmSvc)

		// Default display ID is 0.
		density, err := wm.GetBaseDisplayDensity(ctx, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetBaseDisplayDensity: %v\n", err)
		} else {
			fmt.Printf("Base display density: %d dpi\n", density)
		}

		// Check if view server is running (debug tool).
		vsRunning, err := wm.IsViewServerRunning(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsViewServerRunning: %v\n", err)
		} else {
			fmt.Printf("View server running:  %v\n", vsRunning)
		}
	}

	fmt.Println()
	fmt.Println("=== End Kiosk Info ===")
}
