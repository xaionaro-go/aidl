// Query screensaver/daydream state via DreamManager.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/dream_manager ./examples/dream_manager/
//	adb push dream_manager /data/local/tmp/ && adb shell /data/local/tmp/dream_manager
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/service/dreams"
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

	svc, err := sm.GetService(ctx, servicemanager.DreamService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get dream service: %v\n", err)
		os.Exit(1)
	}

	dm := dreams.NewDreamManagerProxy(svc)

	// Check if the device is currently dreaming.
	dreaming, err := dm.IsDreaming(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsDreaming: %v\n", err)
	} else {
		fmt.Printf("Currently dreaming: %v\n", dreaming)
	}

	// Check dreaming or in preview.
	dreamingOrPreview, err := dm.IsDreamingOrInPreview(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsDreamingOrInPreview: %v\n", err)
	} else {
		fmt.Printf("Dreaming or in preview: %v\n", dreamingOrPreview)
	}

	// Query configured dream components.
	components, err := dm.GetDreamComponents(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDreamComponents: %v\n", err)
	} else {
		fmt.Printf("Dream components: %d configured\n", len(components))
	}

	// Query dream components for current user.
	userComponents, err := dm.GetDreamComponentsForUser(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDreamComponentsForUser: %v\n", err)
	} else {
		fmt.Printf("Dream components (user): %d configured\n", len(userComponents))
	}
}
