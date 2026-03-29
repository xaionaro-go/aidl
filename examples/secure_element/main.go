// Query OMAPI SecureElementService for available readers.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/secure_element ./examples/secure_element/
//	adb push secure_element /data/local/tmp/ && adb shell /data/local/tmp/secure_element
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/se/omapi"
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

	svc, err := sm.GetService(ctx, servicemanager.SecureElementService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get secure_element service: %v\n", err)
		os.Exit(1)
	}

	proxy := omapi.NewSecureElementServiceProxy(svc)

	fmt.Println("=== Secure Element (OMAPI) ===")

	readers, err := proxy.GetReaders(ctx)
	if err != nil {
		fmt.Printf("GetReaders: %v\n", err)
	} else {
		fmt.Printf("Available readers: %d\n", len(readers))
		for i, r := range readers {
			fmt.Printf("  [%d] %s\n", i, r)
		}
	}
}
