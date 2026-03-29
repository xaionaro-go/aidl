// Query update engine for OTA update status.
//
// Uses SystemUpdateManager to retrieve pending update information
// and RecoverySystem to check recovery state.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/ota_status ./examples/ota_status/
//	adb push build/ota_status /data/local/tmp/ && adb shell /data/local/tmp/ota_status
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
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

	fmt.Println("=== OTA Update Status ===")
	fmt.Println()

	// SystemUpdateManager
	updateMgr, err := genOs.GetSystemUpdateManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SystemUpdateManager unavailable: %v\n", err)
	} else {
		info, err := updateMgr.RetrieveSystemUpdateInfo(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "RetrieveSystemUpdateInfo: %v\n", err)
		} else {
			// The result is a Bundle containing update status fields.
			_ = info
			fmt.Println("System update info: bundle retrieved successfully")
		}
	}
	fmt.Println()

	// RecoverySystem — check if LSKF is captured (pre-OTA readiness).
	recovery, err := genOs.GetRecoverySystem(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RecoverySystem unavailable: %v\n", err)
	} else {
		captured, err := recovery.IsLskfCaptured(ctx, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "IsLskfCaptured: %v\n", err)
		} else {
			fmt.Printf("LSKF captured (OTA readiness): %v\n", captured)
		}
	}

	fmt.Println()
	fmt.Println("=== End OTA Status ===")
}
