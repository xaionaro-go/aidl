// Query carrier configuration: default carrier service package,
// configuration for subscriptions.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/carrier_config ./examples/carrier_config/
//	adb push carrier_config /data/local/tmp/ && adb shell /data/local/tmp/carrier_config
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/com/android/internal_/telephony"
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

	svc, err := sm.GetService(ctx, servicemanager.CarrierConfigService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get carrier_config service: %v\n", err)
		os.Exit(1)
	}

	cc := telephony.NewCarrierConfigLoaderProxy(svc)

	fmt.Println("=== Carrier Configuration ===")

	// Get default carrier service package name.
	pkg, err := cc.GetDefaultCarrierServicePackageName(ctx)
	if err != nil {
		fmt.Printf("GetDefaultCarrierServicePackageName: %v\n", err)
	} else {
		fmt.Printf("Default carrier service package: %q\n", pkg)
	}

	// Get config for default subscription (subId=1).
	config, err := cc.GetConfigForSubId(ctx, 1)
	if err != nil {
		fmt.Printf("GetConfigForSubId(1): %v\n", err)
	} else {
		fmt.Printf("Carrier config for subId=1: %+v\n", config)
	}
}
