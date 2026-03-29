// Factory reset demonstration via DevicePolicyManager.
//
// WARNING: This example shows the API but does NOT actually perform
// a factory reset. The WipeDataWithReason call is commented out to
// prevent accidental data loss. Uncomment it only on emulators or
// devices you intend to wipe.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/factory_reset ./examples/factory_reset/
//	adb push build/factory_reset /data/local/tmp/ && adb shell /data/local/tmp/factory_reset
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	genAdmin "github.com/AndroidGoLab/binder/android/app/admin"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// isEmulator checks build properties for emulator indicators.
func isEmulator() bool {
	data, err := os.ReadFile("/sys/class/dmi/id/product_name")
	if err == nil {
		product := strings.TrimSpace(string(data))
		if strings.Contains(strings.ToLower(product), "emulator") ||
			strings.Contains(strings.ToLower(product), "sdk") {
			return true
		}
	}

	// Check for goldfish/ranchu (emulator kernel).
	data, err = os.ReadFile("/proc/cpuinfo")
	if err == nil {
		cpuinfo := strings.ToLower(string(data))
		if strings.Contains(cpuinfo, "goldfish") ||
			strings.Contains(cpuinfo, "ranchu") {
			return true
		}
	}

	// Check ro.hardware via getprop (if available).
	data, err = os.ReadFile("/sys/class/dmi/id/board_name")
	if err == nil {
		board := strings.TrimSpace(strings.ToLower(string(data)))
		if strings.Contains(board, "goldfish") ||
			strings.Contains(board, "ranchu") {
			return true
		}
	}

	return false
}

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
	callerPkg := binder.DefaultCallerIdentity().PackageName

	dpm, err := genAdmin.GetDevicePolicyManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get device_policy service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Factory Reset Example ===")
	fmt.Println()

	// Show device state before reset.
	provisioned, err := dpm.IsDeviceProvisioned(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsDeviceProvisioned: %v\n", err)
	} else {
		fmt.Printf("Device provisioned: %v\n", provisioned)
	}

	encStatus, err := dpm.GetStorageEncryptionStatus(ctx, callerPkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetStorageEncryptionStatus: %v\n", err)
	} else {
		fmt.Printf("Encryption status:  %d\n", encStatus)
	}

	fmt.Println()

	if !isEmulator() {
		fmt.Println("WARNING: Not running on an emulator.")
		fmt.Println("Factory reset will NOT be performed.")
		fmt.Println("To perform a factory reset, run on an emulator.")
		fmt.Println()
		fmt.Println("The API call would be:")
		fmt.Println("  dpm.WipeDataWithReason(ctx, callerPkg, 0, \"remote-wipe\", false)")
		return
	}

	fmt.Println("Running on emulator. Factory reset API is available.")
	fmt.Println("The factory reset call is commented out for safety.")
	fmt.Println()
	fmt.Println("To actually perform factory reset, uncomment the call below:")
	fmt.Println("  dpm.WipeDataWithReason(ctx, callerPkg, 0, \"remote-wipe\", false)")

	// DANGER: Uncomment the following line ONLY on emulators you intend to wipe.
	// err = dpm.WipeDataWithReason(ctx, callerPkg, 0, "remote-wipe", false)
	// if err != nil {
	//     fmt.Fprintf(os.Stderr, "WipeDataWithReason: %v\n", err)
	// } else {
	//     fmt.Println("Factory reset initiated.")
	// }

	_ = dpm // Suppress unused warning if uncommented above.
}
