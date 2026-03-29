// Retrieve the last known fused location without registering a listener.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/last_location ./examples/last_location/
//	adb push build/last_location /data/local/tmp/ && adb shell /data/local/tmp/last_location
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

	packageName := binder.DefaultCallerIdentity().PackageName

	providers := []string{
		string(location.FusedProvider),
		string(location.GpsProvider),
		string(location.NetworkProvider),
	}

	for _, provider := range providers {
		loc, err := lm.GetLastLocation(ctx, provider, location.LastLocationRequest{}, packageName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetLastLocation(%s): %v\n", provider, err)
			continue
		}
		if loc.LatitudeDegrees == 0 && loc.LongitudeDegrees == 0 && loc.TimeMs == 0 {
			fmt.Printf("[%s] No cached location available.\n", provider)
		} else {
			fmt.Printf("[%s] Lat: %.6f  Lon: %.6f  Alt: %.1f m  Accuracy: %.1f m\n",
				provider,
				loc.LatitudeDegrees,
				loc.LongitudeDegrees,
				loc.AltitudeMeters,
				loc.HorizontalAccuracyMeters,
			)
		}
	}
}
