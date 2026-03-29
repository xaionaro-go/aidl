// Monitor IMS registration state via ITelephony proxy.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/ims_monitor ./examples/ims_monitor/
//	adb push ims_monitor /data/local/tmp/ && adb shell /data/local/tmp/ims_monitor
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

	svc, err := sm.GetService(ctx, servicemanager.TelephonyService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get phone service: %v\n", err)
		os.Exit(1)
	}

	phone := telephony.NewTelephonyProxy(svc)

	fmt.Println("=== IMS Registration Monitor ===")

	// Check IMS registration on default subscription (subId=1).
	registered, err := phone.IsImsRegistered(ctx, 1)
	if err != nil {
		fmt.Printf("IsImsRegistered(1): %v\n", err)
	} else {
		fmt.Printf("IMS registered (subId=1): %v\n", registered)
	}

	// Check WiFi calling availability.
	wifiCalling, err := phone.IsWifiCallingAvailable(ctx, 1)
	if err != nil {
		fmt.Printf("IsWifiCallingAvailable(1): %v\n", err)
	} else {
		fmt.Printf("WiFi calling available (subId=1): %v\n", wifiCalling)
	}

	// Check video telephony availability.
	videoAvail, err := phone.IsVideoTelephonyAvailable(ctx, 1)
	if err != nil {
		fmt.Printf("IsVideoTelephonyAvailable(1): %v\n", err)
	} else {
		fmt.Printf("Video telephony available (subId=1): %v\n", videoAvail)
	}

	// Check VoWiFi setting.
	vowifi, err := phone.IsVoWiFiSettingEnabled(ctx, 1)
	if err != nil {
		fmt.Printf("IsVoWiFiSettingEnabled(1): %v\n", err)
	} else {
		fmt.Printf("VoWiFi setting enabled (subId=1): %v\n", vowifi)
	}

	// Also check the telephony_ims service availability.
	imsSvc, err := sm.CheckService(ctx, servicemanager.TelephonyImsService)
	if err != nil {
		fmt.Printf("\nCheckService(telephony_ims): %v\n", err)
	} else if imsSvc == nil {
		fmt.Println("\ntelephony_ims service: NOT REGISTERED")
	} else {
		fmt.Printf("\ntelephony_ims service: FOUND (handle=%d)\n", imsSvc.Handle())
		fmt.Printf("telephony_ims alive: %v\n", imsSvc.IsAlive(ctx))
	}
}
