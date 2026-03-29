// Query location provider availability for geofencing use cases.
// Note: RequestGeofence requires a PendingIntent which cannot be fully
// serialized from a non-app context. This example demonstrates the
// prerequisite checks: listing providers and verifying GPS availability.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/geofence ./examples/geofence/
//	adb push build/geofence /data/local/tmp/ && adb shell /data/local/tmp/geofence
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/location"
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

	lm, err := location.GetLocationManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get location manager: %v\n", err)
		os.Exit(1)
	}

	providers, err := lm.GetAllProviders(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAllProviders: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Available providers (%d):\n", len(providers))
	for _, p := range providers {
		fmt.Printf("  - %s\n", p)
	}

	// Check if GPS is available (required for geofencing).
	hasGPS, err := lm.HasProvider(ctx, location.GpsProvider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HasProvider(gps): %v\n", err)
	} else {
		fmt.Printf("\nGPS provider available: %v\n", hasGPS)
	}

	hasFused, err := lm.HasProvider(ctx, location.FusedProvider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HasProvider(fused): %v\n", err)
	} else {
		fmt.Printf("Fused provider available: %v\n", hasFused)
	}

	// Check if location is enabled for the current user.
	enabled, err := lm.IsLocationEnabledForUser(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsLocationEnabledForUser: %v\n", err)
	} else {
		fmt.Printf("Location enabled: %v\n", enabled)
	}

	if !hasGPS {
		fmt.Println("\nGeofencing requires GPS. Provider not available.")
	} else {
		fmt.Println("\nGeofencing prerequisites met.")
	}
}
