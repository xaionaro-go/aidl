// Acquire and release a system suspend wake lock to demonstrate SystemSuspend interaction.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/suspend_logger ./examples/suspend_logger/
//	adb push build/suspend_logger /data/local/tmp/ && adb shell /data/local/tmp/suspend_logger
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AndroidGoLab/binder/android/system/suspend"
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

	svc, err := sm.GetService(ctx, servicemanager.ServiceName(suspend.DescriptorISystemSuspend+"/default"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get SystemSuspend service: %v\n", err)
		os.Exit(1)
	}

	ss := suspend.NewSystemSuspendProxy(svc)

	fmt.Println("Acquiring partial wake lock...")
	wl, err := ss.AcquireWakeLock(ctx, suspend.WakeLockTypePARTIAL, "suspend_logger_example")
	if err != nil {
		fmt.Fprintf(os.Stderr, "AcquireWakeLock: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wake lock acquired. Device will not suspend.")

	fmt.Println("Holding for 3 seconds...")
	time.Sleep(3 * time.Second)

	fmt.Println("Releasing wake lock...")
	if err := wl.Release(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Release: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wake lock released. Device may suspend normally.")
}
