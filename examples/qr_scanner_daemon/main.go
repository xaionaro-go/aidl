// QR/barcode scanner daemon that captures camera frames for processing.
//
// Demonstrates:
//   - camera.Connect + ConfigureStream + CaptureFrame pipeline
//   - Continuous frame capture loop with timeout
//   - Frame statistics for barcode detection readiness
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/qr_scanner_daemon ./examples/qr_scanner_daemon/
//	adb push build/qr_scanner_daemon /data/local/tmp/ && adb shell /data/local/tmp/qr_scanner_daemon
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/camera"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func run(ctx context.Context) error {
	const (
		cameraID   = "0"
		width      = 640
		height     = 480
		scanFrames = 5
	)

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(4*1024*1024))
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer drv.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		return fmt.Errorf("transport: %w", err)
	}
	sm := servicemanager.New(transport)

	cam, err := camera.Connect(ctx, sm, transport, cameraID)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cam.Close(ctx)

	if err := cam.ConfigureStream(ctx, width, height, camera.FormatYCbCr420); err != nil {
		return fmt.Errorf("configure stream: %w", err)
	}

	fmt.Printf("QR scanner daemon: capturing %d frames at %dx%d\n", scanFrames, width, height)

	for i := 0; i < scanFrames; i++ {
		captureCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		frame, err := cam.CaptureFrame(captureCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("capture frame %d: %w", i, err)
		}

		// Compute frame statistics that would feed into barcode detection.
		nonZero := 0
		var sum uint64
		for _, b := range frame {
			if b != 0 {
				nonZero++
			}
			sum += uint64(b)
		}

		avgLuma := float64(0)
		if len(frame) > 0 {
			avgLuma = float64(sum) / float64(len(frame))
		}

		fmt.Printf("  Frame %d: %d bytes, %.1f%% non-zero, avg luma=%.1f\n",
			i, len(frame), float64(nonZero)/float64(len(frame))*100, avgLuma)
	}

	fmt.Println("QR scanner daemon: scan complete")
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}
