// List sound trigger modules via SoundTriggerMiddlewareService.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/sound_trigger ./examples/sound_trigger/
//	adb push sound_trigger /data/local/tmp/ && adb shell /data/local/tmp/sound_trigger
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/media/permission"
	"github.com/AndroidGoLab/binder/android/media/soundtrigger_middleware"
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

	svc, err := sm.GetService(ctx, servicemanager.SoundTriggerMiddlewareService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get soundtrigger_middleware service: %v\n", err)
		os.Exit(1)
	}

	stm := soundtrigger_middleware.NewSoundTriggerMiddlewareServiceProxy(svc)

	// Build an identity for the calling process.
	identity := permission.Identity{
		Uid:            int32(os.Getuid()),
		Pid:            int32(os.Getpid()),
		PackageName:    "",
		AttributionTag: "",
	}

	modules, err := stm.ListModulesAsOriginator(ctx, identity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListModulesAsOriginator: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sound trigger modules: %d\n", len(modules))
	for i, mod := range modules {
		fmt.Printf("  [%d] handle=%d\n", i, mod.Handle)
	}
}
