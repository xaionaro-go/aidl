// Resolve AIDL method names to transaction codes for binder services.
//
// Uses the version-aware transport's ResolveCode to map method names
// to their corresponding binder transaction codes for the detected
// device API level.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/transaction_resolver ./examples/transaction_resolver/
//	adb push build/transaction_resolver /data/local/tmp/ && adb shell /data/local/tmp/transaction_resolver
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

type methodQuery struct {
	service    string
	descriptor string
	method     string
}

var queries = []methodQuery{
	{"activity", "android.app.IActivityManager", "getProcessLimit"},
	{"activity", "android.app.IActivityManager", "isUserAMonkey"},
	{"activity", "android.app.IActivityManager", "checkPermission"},
	{"activity", "android.app.IActivityManager", "isAppFreezerSupported"},
	{"power", "android.os.IPowerManager", "isInteractive"},
	{"power", "android.os.IPowerManager", "isPowerSaveMode"},
	{"power", "android.os.IPowerManager", "isDeviceIdleMode"},
	{"SurfaceFlingerAIDL", "android.gui.ISurfaceComposer", "getPhysicalDisplayIds"},
	{"SurfaceFlingerAIDL", "android.gui.ISurfaceComposer", "getBootDisplayModeSupport"},
	{"display", "android.hardware.display.IDisplayManager", "getDisplayIds"},
	{"display", "android.hardware.display.IDisplayManager", "getBrightness"},
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

	fmt.Println("=== Transaction Code Resolver ===")
	fmt.Println()
	fmt.Printf("%-25s %-45s %-35s %s\n", "SERVICE", "DESCRIPTOR", "METHOD", "CODE")
	fmt.Println("-----------------------------------------------------------------------------------------------------------")

	svcCache := map[string]binder.IBinder{}

	for _, q := range queries {
		svc, ok := svcCache[q.service]
		if !ok {
			svc, err = sm.GetService(ctx, servicemanager.ServiceName(q.service))
			if err != nil {
				fmt.Printf("%-25s %-45s %-35s ERROR: %v\n", q.service, q.descriptor, q.method, err)
				continue
			}
			svcCache[q.service] = svc
		}

		code, err := svc.ResolveCode(ctx, q.descriptor, q.method)
		if err != nil {
			fmt.Printf("%-25s %-45s %-35s ERROR: %v\n", q.service, q.descriptor, q.method, err)
		} else {
			fmt.Printf("%-25s %-45s %-35s %d (0x%x)\n", q.service, q.descriptor, q.method, code, code)
		}
	}
}
