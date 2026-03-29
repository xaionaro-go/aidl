// Periodic timelapse camera capture via binder.
//
// Demonstrates:
//   - camera.Connect + ConfigureStream for capture pipeline
//   - Periodic frame capture with configurable interval
//   - Frame data validation and statistics
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/timelapse_capture ./examples/timelapse_capture/
//	adb push build/timelapse_capture /data/local/tmp/ && adb shell /data/local/tmp/timelapse_capture
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/camera"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func run(ctx context.Context) error {
	const (
		cameraID       = "0"
		width          = 640
		height         = 480
		captureCount   = 3
		captureTimeout = 10 * time.Second
		interval       = 2 * time.Second
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
		if strings.Contains(err.Error(), "ServiceSpecific") {
			fmt.Fprintf(os.Stderr, "Camera device %q not available (HAL reports disconnected).\n", cameraID)
			fmt.Fprintf(os.Stderr, "This is expected on emulators without a camera backend.\n")
			return nil
		}
		return fmt.Errorf("connect: %w", err)
	}
	defer cam.Close(ctx)

	if err := cam.ConfigureStream(ctx, width, height, camera.FormatYCbCr420); err != nil {
		return fmt.Errorf("configure stream: %w", err)
	}

	fmt.Printf("Timelapse: capturing %d frames at %v intervals (%dx%d)\n",
		captureCount, interval, width, height)

	for i := 0; i < captureCount; i++ {
		if i > 0 {
			fmt.Printf("  Waiting %v...\n", interval)
			time.Sleep(interval)
		}

		captureCtx, cancel := context.WithTimeout(ctx, captureTimeout)
		frame, err := cam.CaptureFrame(captureCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("capture frame %d: %w", i, err)
		}

		nonZero := 0
		for _, b := range frame {
			if b != 0 {
				nonZero++
			}
		}

		ts := time.Now().Format("15:04:05")
		fmt.Printf("  Frame %d [%s]: %d bytes, %.1f%% non-zero\n",
			i, ts, len(frame), float64(nonZero)/float64(len(frame))*100)
	}

	fmt.Println("Timelapse capture complete")
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}
