// Measure round-trip binder transaction times.
//
// Sends PING transactions to multiple services and measures the
// latency of each round-trip through the binder driver.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/binder_latency ./examples/binder_latency/
//	adb push build/binder_latency /data/local/tmp/ && adb shell /data/local/tmp/binder_latency
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	warmupRounds = 5
	benchRounds  = 50
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

	targets := []string{"SurfaceFlinger", "activity", "SurfaceFlingerAIDL"}

	fmt.Printf("=== Binder Latency Benchmark (PING x %d) ===\n\n", benchRounds)
	fmt.Printf("%-25s %10s %10s %10s\n", "SERVICE", "MIN", "AVG", "MAX")
	fmt.Println("-----------------------------------------------------------")

	for _, name := range targets {
		svc, err := sm.GetService(ctx, servicemanager.ServiceName(name))
		if err != nil {
			fmt.Printf("%-25s  unavailable: %v\n", name, err)
			continue
		}

		// Warmup: discard first few to stabilize caches.
		for i := 0; i < warmupRounds; i++ {
			_, _ = svc.Transact(ctx, binder.PingTransaction, 0, parcel.New())
		}

		var minDur, maxDur, totalDur time.Duration

		for i := 0; i < benchRounds; i++ {
			data := parcel.New()
			start := time.Now()
			reply, err := svc.Transact(ctx, binder.PingTransaction, 0, data)
			elapsed := time.Since(start)
			data.Recycle()

			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s round %d: %v\n", name, i, err)
				continue
			}
			reply.Recycle()

			totalDur += elapsed
			if i == 0 || elapsed < minDur {
				minDur = elapsed
			}
			if elapsed > maxDur {
				maxDur = elapsed
			}
		}

		avg := totalDur / time.Duration(benchRounds)
		fmt.Printf("%-25s %10s %10s %10s\n", name, minDur, avg, maxDur)
	}
}
