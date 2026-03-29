// Monitor remote key provisioning (RKP) and device security state.
//
// Queries SecurityStateManager.GetGlobalSecurityState for patch-level
// and key provisioning information, SystemUpdateManager for pending
// OTA updates, and checks remote_provisioning service availability.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/rkp_monitor ./examples/rkp_monitor/
//	adb push rkp_monitor /data/local/tmp/ && adb shell /data/local/tmp/rkp_monitor
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	genSecurity "github.com/AndroidGoLab/binder/android/security"
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

	fmt.Println("=== Remote Key Provisioning & Security State ===")
	fmt.Println()

	// 1. Remote Provisioning service availability
	fmt.Print("Remote provisioning service: ")
	rpSvc, err := sm.CheckService(ctx, servicemanager.RemoteProvisioningService)
	if err != nil {
		fmt.Printf("error (%v)\n", err)
	} else if rpSvc == nil {
		fmt.Println("NOT REGISTERED")
	} else {
		fmt.Printf("FOUND (handle=%d, alive=%v)\n", rpSvc.Handle(), rpSvc.IsAlive(ctx))
	}

	// 2. Global security state (patch levels, provisioning info)
	fmt.Print("Global security state:      ")
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

	// 3. System update status — relevant because RKP keys
	//    may need reprovisioning after OTA updates.
	fmt.Print("System update info:         ")
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

	// 4. File integrity — APK verity support indicates
	//    hardware-backed integrity verification capability.
	fmt.Print("APK verity supported:       ")
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

	// 5. Hardware properties — device thermal state can affect
	//    key provisioning operations.
	fmt.Print("Hardware properties:        ")
	hwSvc, err := sm.CheckService(ctx, servicemanager.HardwarePropertiesService)
	if err != nil {
		fmt.Printf("error (%v)\n", err)
	} else if hwSvc == nil {
		fmt.Println("NOT REGISTERED")
	} else {
		fmt.Printf("FOUND (handle=%d, alive=%v)\n", hwSvc.Handle(), hwSvc.IsAlive(ctx))
	}
}
