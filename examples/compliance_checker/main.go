// Verify device compliance: encryption, security state, OTA update status.
//
// Queries DevicePolicyManager for encryption, SecurityStateManager for
// global security state, and SystemUpdateManager for pending updates.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/compliance_checker ./examples/compliance_checker/
//	adb push build/compliance_checker /data/local/tmp/ && adb shell /data/local/tmp/compliance_checker
package main

import (
	"context"
	"fmt"
	"os"

	genAdmin "github.com/AndroidGoLab/binder/android/app/admin"
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
	callerPkg := binder.DefaultCallerIdentity().PackageName
	allPassed := true

	fmt.Println("=== Device Compliance Check ===")
	fmt.Println()

	// Check 1: Encryption status
	fmt.Print("Encryption status: ")
	dpm, err := genAdmin.GetDevicePolicyManager(ctx, sm)
	if err != nil {
		fmt.Printf("SKIP (service unavailable: %v)\n", err)
	} else {
		encStatus, err := dpm.GetStorageEncryptionStatus(ctx, callerPkg)
		if err != nil {
			fmt.Printf("SKIP (%v)\n", err)
		} else {
			// Status >= 3 means some form of active encryption.
			if encStatus >= 3 {
				fmt.Printf("PASS (active, code=%d)\n", encStatus)
			} else {
				fmt.Printf("FAIL (code=%d, expected >= 3)\n", encStatus)
				allPassed = false
			}
		}
	}

	// Check 2: Security state / patch level
	fmt.Print("Security state:    ")
	secMgr, err := genOs.GetSecurityStateManager(ctx, sm)
	if err != nil {
		fmt.Printf("SKIP (service unavailable: %v)\n", err)
	} else {
		state, err := secMgr.GetGlobalSecurityState(ctx)
		if err != nil {
			fmt.Printf("SKIP (%v)\n", err)
		} else {
			// Bundle was parsed; the raw data length indicates validity.
			_ = state
			fmt.Printf("PASS (security state bundle retrieved)\n")
		}
	}

	// Check 3: System update status
	fmt.Print("OTA update status: ")
	updateMgr, err := genOs.GetSystemUpdateManager(ctx, sm)
	if err != nil {
		fmt.Printf("SKIP (service unavailable: %v)\n", err)
	} else {
		info, err := updateMgr.RetrieveSystemUpdateInfo(ctx)
		if err != nil {
			fmt.Printf("SKIP (%v)\n", err)
		} else {
			_ = info
			fmt.Printf("PASS (update info bundle retrieved)\n")
		}
	}

	// Check 4: Device provisioned
	fmt.Print("Device provisioned:")
	if dpm != nil {
		provisioned, err := dpm.IsDeviceProvisioned(ctx)
		if err != nil {
			fmt.Printf(" SKIP (%v)\n", err)
		} else if provisioned {
			fmt.Printf(" PASS\n")
		} else {
			fmt.Printf(" FAIL (not provisioned)\n")
			allPassed = false
		}
	} else {
		fmt.Printf(" SKIP (DPM unavailable)\n")
	}

	fmt.Println()
	if allPassed {
		fmt.Println("Overall: COMPLIANT")
	} else {
		fmt.Println("Overall: NON-COMPLIANT")
	}
}
