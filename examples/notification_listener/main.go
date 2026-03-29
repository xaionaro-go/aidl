// Query notification state via NotificationManager: zen mode, active notifications.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/notification_listener ./examples/notification_listener/
//	adb push notification_listener /data/local/tmp/ && adb shell /data/local/tmp/notification_listener
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

	// Query Zen (Do Not Disturb) mode.
	zenMode, err := nm.GetZenMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetZenMode: %v\n", err)
	} else {
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
		fmt.Printf("Zen mode: %s\n", name)
	}

	// Check if notifications are enabled for a well-known package.
	enabled, err := nm.AreNotificationsEnabled(ctx, "com.android.systemui")
	if err != nil {
		fmt.Fprintf(os.Stderr, "AreNotificationsEnabled: %v\n", err)
	} else {
		fmt.Printf("Notifications enabled for systemui: %v\n", enabled)
	}

	// Check effects suppressor.
	suppressor, err := nm.GetEffectsSuppressor(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetEffectsSuppressor: %v\n", err)
	} else {
		fmt.Printf("Effects suppressor: %+v\n", suppressor)
	}

	// Check if channels are bypassing DND.
	bypassing, err := nm.AreChannelsBypassingDnd(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AreChannelsBypassingDnd: %v\n", err)
	} else {
		fmt.Printf("Channels bypassing DND: %v\n", bypassing)
	}

	// Try to read active notifications (requires system privileges).
	notifs, err := nm.GetActiveNotifications(ctx, "com.android.shell")
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetActiveNotifications: %v (may require system permission)\n", err)
	} else {
		fmt.Printf("Active notifications: %d\n", len(notifs))
	}
}
