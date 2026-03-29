// Introspect methods on binder services.
//
// Lists all registered services and pings each to check liveness.
// For well-known services, prints their interface descriptor and
// demonstrates method resolution using ResolveCode.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/aidl_explorer ./examples/aidl_explorer/
//	adb push build/aidl_explorer /data/local/tmp/ && adb shell /data/local/tmp/aidl_explorer
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

// knownInterfaces maps service names to their AIDL descriptors and
// methods for exploration.
var knownInterfaces = []struct {
	service    string
	descriptor string
	methods    []string
}{
	{
		service:    "SurfaceFlingerAIDL",
		descriptor: "android.gui.ISurfaceComposer",
		methods:    []string{"getPhysicalDisplayIds", "getBootDisplayModeSupport", "getStaticDisplayInfo"},
	},
	{
		service:    "activity",
		descriptor: "android.app.IActivityManager",
		methods:    []string{"getProcessLimit", "isUserAMonkey", "checkPermission", "isAppFreezerSupported"},
	},
	{
		service:    "power",
		descriptor: "android.os.IPowerManager",
		methods:    []string{"isInteractive", "isPowerSaveMode", "isDeviceIdleMode"},
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

	// List all services.
	services, err := sm.ListServices(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListServices: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== AIDL Explorer: %d services registered ===\n\n", len(services))

	// Show first 15 services with ping status.
	limit := 15
	if len(services) < limit {
		limit = len(services)
	}
	for i := 0; i < limit; i++ {
		svc, err := sm.CheckService(ctx, services[i])
		status := "unreachable"
		if err == nil && svc != nil {
			if svc.IsAlive(ctx) {
				status = fmt.Sprintf("alive (handle=%d)", svc.Handle())
			} else {
				status = "dead"
			}
		}
		fmt.Printf("  %-45s %s\n", services[i], status)
	}
	if len(services) > limit {
		fmt.Printf("  ... and %d more\n", len(services)-limit)
	}
	fmt.Println()

	// Explore known interfaces: resolve method names to transaction codes.
	fmt.Println("=== Method Resolution ===")
	fmt.Println()

	for _, iface := range knownInterfaces {
		svc, err := sm.GetService(ctx, servicemanager.ServiceName(iface.service))
		if err != nil {
			fmt.Printf("  %s: unavailable (%v)\n", iface.service, err)
			continue
		}

		fmt.Printf("  %s (%s):\n", iface.service, iface.descriptor)

		for _, method := range iface.methods {
			code, err := svc.ResolveCode(ctx, iface.descriptor, method)
			if err != nil {
				fmt.Printf("    %-40s -> ERROR: %v\n", method, err)
			} else {
				fmt.Printf("    %-40s -> code %d (0x%x)\n", method, code, code)
			}
		}
		fmt.Println()
	}
}
