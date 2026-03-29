// Send randomized parcel data to services to test robustness.
//
// Sends junk transaction data to SurfaceFlinger's PING endpoint
// (a safe, read-only operation) to verify that the binder driver
// handles malformed data gracefully.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/binder_fuzzer ./examples/binder_fuzzer/
//	adb push build/binder_fuzzer /data/local/tmp/ && adb shell /data/local/tmp/binder_fuzzer
package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const fuzzIterations = 20

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

	// Use SurfaceFlinger as a safe target: PING is side-effect-free.
	sf, err := sm.GetService(ctx, servicemanager.ServiceName("SurfaceFlinger"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get SurfaceFlinger: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Fuzzing SurfaceFlinger PING with %d randomized parcels...\n\n", fuzzIterations)

	succeeded := 0
	failed := 0

	for i := 0; i < fuzzIterations; i++ {
		data := parcel.New()

		// Write random junk data (4-64 bytes).
		junkLen := 4 + rand.Intn(61)
		junk := make([]byte, junkLen)
		for j := range junk {
			junk[j] = byte(rand.Intn(256))
		}
		data.WriteRawBytes(junk)

		// Send as PING transaction (safe, won't modify state).
		reply, err := sf.Transact(ctx, binder.PingTransaction, 0, data)
		if err != nil {
			fmt.Printf("  [%2d] %d bytes -> error: %v\n", i+1, junkLen, err)
			failed++
		} else {
			fmt.Printf("  [%2d] %d bytes -> reply %d bytes\n", i+1, junkLen, reply.Len())
			succeeded++
			reply.Recycle()
		}

		data.Recycle()
	}

	fmt.Printf("\nResults: %d succeeded, %d failed (errors are expected for junk data)\n", succeeded, failed)
	fmt.Println("Service remained alive after fuzzing: ", sf.IsAlive(ctx))
}
