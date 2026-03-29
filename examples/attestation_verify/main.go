// Query attestation verification and related security services.
//
// The attestation verification service's methods (VerifyAttestation,
// VerifyToken) require complex callback parameters, so this example
// demonstrates the service alongside related security queries:
// SecurityStateManager.GetGlobalSecurityState and
// FileIntegrityService.IsApkVeritySupported.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/attestation_verify ./examples/attestation_verify/
//	adb push attestation_verify /data/local/tmp/ && adb shell /data/local/tmp/attestation_verify
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	genSecurity "github.com/AndroidGoLab/binder/android/security"
	genAV "github.com/AndroidGoLab/binder/android/security/attestationverification"
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

	fmt.Println("=== Attestation & Security Services ===")
	fmt.Println()

	// 1. Attestation Verification Manager Service
	fmt.Print("Attestation verification service: ")
	avProxy, err := genAV.GetAttestationVerificationManagerService(ctx, sm)
	if err != nil {
		fmt.Printf("unavailable (%v)\n", err)
	} else {
		alive := avProxy.AsBinder().IsAlive(ctx)
		fmt.Printf("FOUND (alive=%v)\n", alive)
		fmt.Println("  Methods: VerifyAttestation, VerifyToken (require callback binders)")
	}

	// 2. File Integrity Service — check APK verity support
	fmt.Print("APK verity supported:             ")
	fiProxy, err := genSecurity.GetFileIntegrityService(ctx, sm)
	if err != nil {
		fmt.Printf("service unavailable (%v)\n", err)
	} else {
		supported, err := fiProxy.IsApkVeritySupported(ctx)
		if err != nil {
			fmt.Printf("error (%v)\n", err)
		} else {
			fmt.Printf("%v\n", supported)
		}
	}

	// 3. Security State Manager — query global security patch level
	fmt.Print("Global security state:            ")
	secMgr, err := genOs.GetSecurityStateManager(ctx, sm)
	if err != nil {
		fmt.Printf("service unavailable (%v)\n", err)
	} else {
		state, err := secMgr.GetGlobalSecurityState(ctx)
		if err != nil {
			fmt.Printf("error (%v)\n", err)
		} else {
			_ = state
			fmt.Println("retrieved (Bundle)")
		}
	}

	// 4. System Update Manager — OTA update info
	fmt.Print("System update info:               ")
	updateMgr, err := genOs.GetSystemUpdateManager(ctx, sm)
	if err != nil {
		fmt.Printf("service unavailable (%v)\n", err)
	} else {
		info, err := updateMgr.RetrieveSystemUpdateInfo(ctx)
		if err != nil {
			fmt.Printf("error (%v)\n", err)
		} else {
			_ = info
			fmt.Println("retrieved (Bundle)")
		}
	}
}
