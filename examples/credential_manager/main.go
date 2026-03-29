// Query the CredentialManager service for availability.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/credential_manager ./examples/credential_manager/
//	adb push credential_manager /data/local/tmp/ && adb shell /data/local/tmp/credential_manager
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/credentials"
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

	proxy, err := credentials.GetCredentialManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get credential service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Credential Manager ===")

	enabled, err := proxy.IsServiceEnabled(ctx)
	if err != nil {
		fmt.Printf("IsServiceEnabled: %v\n", err)
	} else {
		fmt.Printf("Credential service enabled: %v\n", enabled)
	}

	// Query available credential providers.
	providers, err := proxy.GetCredentialProviderServices(ctx, 0)
	if err != nil {
		fmt.Printf("GetCredentialProviderServices: %v\n", err)
	} else {
		fmt.Printf("Credential providers: %d\n", len(providers))
		for i, p := range providers {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(providers)-10)
				break
			}
			fmt.Printf("  [%d] %+v\n", i, p)
		}
	}
}
