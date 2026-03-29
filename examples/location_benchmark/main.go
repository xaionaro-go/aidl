// Compare location providers by querying all providers and their properties.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/location_benchmark ./examples/location_benchmark/
//	adb push build/location_benchmark /data/local/tmp/ && adb shell /data/local/tmp/location_benchmark
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

func powerUsageString(p int32) string {
	switch p {
	case 1:
		return "LOW"
	case 2:
		return "MEDIUM"
	case 3:
		return "HIGH"
	default:
		return fmt.Sprintf("%d", p)
	}
}

func accuracyString(a int32) string {
	switch a {
	case 1:
		return "FINE"
	case 2:
		return "COARSE"
	default:
		return fmt.Sprintf("%d", a)
	}
}

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

	fmt.Printf("%-15s %-8s %-8s %-8s %-8s %-8s %-8s\n",
		"PROVIDER", "POWER", "ACCURACY", "SAT", "NET", "CELL", "ALT")
	fmt.Println("-----------------------------------------------------------------------")

	for _, p := range providers {
		props, err := lm.GetProviderProperties(ctx, p)
		if err != nil {
			fmt.Printf("%-15s (error: %v)\n", p, err)
			continue
		}
		fmt.Printf("%-15s %-8s %-8s %-8v %-8v %-8v %-8v\n",
			p,
			powerUsageString(props.PowerUsage),
			accuracyString(props.Accuracy),
			props.HasSatelliteRequirement,
			props.HasNetworkRequirement,
			props.HasCellRequirement,
			props.HasAltitudeSupport,
		)
	}
}
