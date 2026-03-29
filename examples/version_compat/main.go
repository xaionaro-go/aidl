// Validate proxy compatibility across API levels.
//
// Attempts to resolve transaction codes for methods across multiple
// services and reports which methods are available on the current
// device's API level.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/version_compat ./examples/version_compat/
//	adb push build/version_compat /data/local/tmp/ && adb shell /data/local/tmp/version_compat
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

type methodCheck struct {
	descriptor string
	method     string
}

var serviceChecks = map[string][]methodCheck{
	"activity": {
		{"android.app.IActivityManager", "getProcessLimit"},
		{"android.app.IActivityManager", "isUserAMonkey"},
		{"android.app.IActivityManager", "checkPermission"},
		{"android.app.IActivityManager", "isAppFreezerSupported"},
		{"android.app.IActivityManager", "getRunningAppProcesses"},
	},
	"power": {
		{"android.os.IPowerManager", "isInteractive"},
		{"android.os.IPowerManager", "isPowerSaveMode"},
		{"android.os.IPowerManager", "isDeviceIdleMode"},
		{"android.os.IPowerManager", "isLowPowerStandbyEnabled"},
		{"android.os.IPowerManager", "isBatterySaverSupported"},
	},
	"display": {
		{"android.hardware.display.IDisplayManager", "getDisplayIds"},
		{"android.hardware.display.IDisplayManager", "getBrightness"},
		{"android.hardware.display.IDisplayManager", "getBrightnessInfo"},
		{"android.hardware.display.IDisplayManager", "getStableDisplaySize"},
	},
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

	fmt.Println("=== API Version Compatibility Check ===")
	fmt.Println()

	for svcName, methods := range serviceChecks {
		svc, err := sm.GetService(ctx, servicemanager.ServiceName(svcName))
		if err != nil {
			fmt.Printf("[%s] SERVICE UNAVAILABLE: %v\n\n", svcName, err)
			continue
		}

		fmt.Printf("[%s] (handle=%d)\n", svcName, svc.Handle())
		available := 0
		for _, m := range methods {
			code, err := svc.ResolveCode(ctx, m.descriptor, m.method)
			if err != nil {
				fmt.Printf("  %-45s MISSING (%v)\n", m.method, err)
			} else {
				fmt.Printf("  %-45s OK (code=%d)\n", m.method, code)
				available++
			}
		}
		fmt.Printf("  Summary: %d/%d methods available\n\n", available, len(methods))
	}
}
