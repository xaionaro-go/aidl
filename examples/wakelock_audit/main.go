// Enumerate supported wake lock levels via PowerManager.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/wakelock_audit ./examples/wakelock_audit/
//	adb push build/wakelock_audit /data/local/tmp/ && adb shell /data/local/tmp/wakelock_audit
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// Standard Android wake lock levels from PowerManager.java.
var wakeLockLevels = []struct {
	Name  string
	Level int32
}{
	{"PARTIAL_WAKE_LOCK", 1},
	{"SCREEN_DIM_WAKE_LOCK", 6},
	{"SCREEN_BRIGHT_WAKE_LOCK", 10},
	{"FULL_WAKE_LOCK", 26},
	{"PROXIMITY_SCREEN_OFF_WAKE_LOCK", 32},
	{"DOZE_WAKE_LOCK", 64},
	{"DRAW_WAKE_LOCK", 128},
}

func main() {
	ctx := context.Background()

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer drv.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)

	power, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get power service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Wake lock level support:")
	for _, wl := range wakeLockLevels {
		supported, err := power.IsWakeLockLevelSupported(ctx, wl.Level)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-35s error: %v\n", wl.Name, err)
		} else {
			fmt.Printf("  %-35s supported: %v\n", wl.Name, supported)
		}
	}

	fmt.Println("\nScreen brightness boosted:", func() string {
		boosted, err := power.IsScreenBrightnessBoosted(ctx)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return fmt.Sprintf("%v", boosted)
	}())
}
