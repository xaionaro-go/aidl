// Query OemLockService for bootloader lock state and OEM unlock status.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/oem_lock_status ./examples/oem_lock_status/
//	adb push oem_lock_status /data/local/tmp/ && adb shell /data/local/tmp/oem_lock_status
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/service/oemlock"
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

	proxy, err := oemlock.GetOemLockService(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get oem_lock service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== OEM Lock Status ===")

	lockName, err := proxy.GetLockName(ctx)
	if err != nil {
		fmt.Printf("GetLockName: %v\n", err)
	} else {
		fmt.Printf("Lock name: %q\n", lockName)
	}

	allowed, err := proxy.IsOemUnlockAllowed(ctx)
	if err != nil {
		fmt.Printf("IsOemUnlockAllowed: %v\n", err)
	} else {
		fmt.Printf("OEM unlock allowed: %v\n", allowed)
	}

	carrierAllowed, err := proxy.IsOemUnlockAllowedByCarrier(ctx)
	if err != nil {
		fmt.Printf("IsOemUnlockAllowedByCarrier: %v\n", err)
	} else {
		fmt.Printf("OEM unlock allowed by carrier: %v\n", carrierAllowed)
	}

	userAllowed, err := proxy.IsOemUnlockAllowedByUser(ctx)
	if err != nil {
		fmt.Printf("IsOemUnlockAllowedByUser: %v\n", err)
	} else {
		fmt.Printf("OEM unlock allowed by user: %v\n", userAllowed)
	}

	unlocked, err := proxy.IsDeviceOemUnlocked(ctx)
	if err != nil {
		fmt.Printf("IsDeviceOemUnlocked: %v\n", err)
	} else {
		fmt.Printf("Device OEM unlocked: %v\n", unlocked)
	}
}
