// Query SMS service: preferred subscription, IMS SMS support.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/sms_monitor ./examples/sms_monitor/
//	adb push sms_monitor /data/local/tmp/ && adb shell /data/local/tmp/sms_monitor
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

	svc, err := sm.GetService(ctx, servicemanager.SmsService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get isms service: %v\n", err)
		os.Exit(1)
	}

	sms := telephony.NewSmsProxy(svc)

	fmt.Println("=== SMS Service Monitor ===")

	// Get preferred SMS subscription.
	prefSub, err := sms.GetPreferredSmsSubscription(ctx)
	if err != nil {
		fmt.Printf("GetPreferredSmsSubscription: %v\n", err)
	} else {
		fmt.Printf("Preferred SMS subscription: %d\n", prefSub)
	}

	// Check if SMS prompt is enabled (for multi-SIM devices).
	promptEnabled, err := sms.IsSMSPromptEnabled(ctx)
	if err != nil {
		fmt.Printf("IsSMSPromptEnabled: %v\n", err)
	} else {
		fmt.Printf("SMS prompt enabled: %v\n", promptEnabled)
	}

	// Check IMS SMS support for default subscription.
	imsSupported, err := sms.IsImsSmsSupportedForSubscriber(ctx, 1)
	if err != nil {
		fmt.Printf("IsImsSmsSupportedForSubscriber(1): %v\n", err)
	} else {
		fmt.Printf("IMS SMS supported (subId=1): %v\n", imsSupported)
	}
}
