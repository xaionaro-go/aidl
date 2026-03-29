// Query GNSS hardware model name, year, and capabilities via LocationManager.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/gnss_diagnostics ./examples/gnss_diagnostics/
//	adb push build/gnss_diagnostics /data/local/tmp/ && adb shell /data/local/tmp/gnss_diagnostics
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

	model, err := lm.GetGnssHardwareModelName(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetGnssHardwareModelName: %v\n", err)
	} else {
		fmt.Printf("GNSS hardware model: %q\n", model)
	}

	year, err := lm.GetGnssYearOfHardware(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetGnssYearOfHardware: %v\n", err)
	} else {
		fmt.Printf("GNSS year of hardware: %d\n", year)
	}

	caps, err := lm.GetGnssCapabilities(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetGnssCapabilities: %v\n", err)
	} else {
		fmt.Printf("GNSS capabilities:\n")
		fmt.Printf("  TopFlags:                    0x%x\n", caps.TopFlags)
		fmt.Printf("  IsAdrCapabilityKnown:        %v\n", caps.IsAdrCapabilityKnown)
		fmt.Printf("  MeasurementCorrectionsFlags: 0x%x\n", caps.MeasurementCorrectionsFlags)
		fmt.Printf("  PowerFlags:                  0x%x\n", caps.PowerFlags)
	}

	batchSize, err := lm.GetGnssBatchSize(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetGnssBatchSize: %v\n", err)
	} else {
		fmt.Printf("GNSS batch size: %d\n", batchSize)
	}
}
