// Measure battery current draw over time via the Health HAL.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/power_profiling ./examples/power_profiling/
//	adb push build/power_profiling /data/local/tmp/ && adb shell /data/local/tmp/power_profiling
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AndroidGoLab/binder/android/hardware/health"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const samples = 5
const interval = 2 * time.Second

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

	svc, err := sm.GetService(ctx, servicemanager.ServiceName(health.DescriptorIHealth+"/default"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get health service: %v\n", err)
		os.Exit(1)
	}

	h := health.NewHealthProxy(svc)

	fmt.Printf("Sampling battery current %d times at %s intervals...\n", samples, interval)
	for i := 0; i < samples; i++ {
		now, err := h.GetCurrentNowMicroamps(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  sample %d: GetCurrentNowMicroamps: %v\n", i, err)
		} else {
			fmt.Printf("  sample %d: %d uA (%.1f mA)\n", i, now, float64(now)/1000.0)
		}

		avg, err := h.GetCurrentAverageMicroamps(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  sample %d: GetCurrentAverageMicroamps: %v\n", i, err)
		} else {
			fmt.Printf("  sample %d avg: %d uA (%.1f mA)\n", i, avg, float64(avg)/1000.0)
		}

		if i < samples-1 {
			time.Sleep(interval)
		}
	}
}
