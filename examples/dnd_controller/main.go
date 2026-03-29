// Query and display Do Not Disturb (Zen) mode via NotificationManager.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/dnd_controller ./examples/dnd_controller/
//	adb push dnd_controller /data/local/tmp/ && adb shell /data/local/tmp/dnd_controller
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/app"
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

	svc, err := sm.GetService(ctx, servicemanager.NotificationService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get notification service: %v\n", err)
		os.Exit(1)
	}

	nm := app.NewNotificationManagerProxy(svc)

	// Read current Zen mode.
	zenMode, err := nm.GetZenMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetZenMode: %v\n", err)
		os.Exit(1)
	}

	zenNames := map[int32]string{
		0: "ZEN_MODE_OFF",
		1: "ZEN_MODE_IMPORTANT_INTERRUPTIONS",
		2: "ZEN_MODE_NO_INTERRUPTIONS",
		3: "ZEN_MODE_ALARMS",
	}
	name := zenNames[zenMode]
	if name == "" {
		name = fmt.Sprintf("unknown(%d)", zenMode)
	}
	fmt.Printf("Current DND (Zen) mode: %s\n", name)

	// Check if channels bypass DND.
	bypassing, err := nm.AreChannelsBypassingDnd(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AreChannelsBypassingDnd: %v\n", err)
	} else {
		fmt.Printf("Channels bypassing DND: %v\n", bypassing)
	}

	// Query consolidated notification policy.
	policy, err := nm.GetConsolidatedNotificationPolicy(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetConsolidatedNotificationPolicy: %v\n", err)
	} else {
		fmt.Printf("Consolidated notification policy: %+v\n", policy)
	}

	// Check if silent status icons should be hidden.
	hidden, err := nm.ShouldHideSilentStatusIcons(ctx, "com.android.shell")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ShouldHideSilentStatusIcons: %v\n", err)
	} else {
		fmt.Printf("Hide silent status icons: %v\n", hidden)
	}
}
