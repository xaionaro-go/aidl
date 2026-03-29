// List accounts on the device via AccountManager.
//
// Queries the "account" service for registered authenticator types
// (Google, Samsung, etc.) and displays their metadata.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/account_manager ./examples/account_manager/
//	adb push build/account_manager /data/local/tmp/ && adb shell /data/local/tmp/account_manager
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/accounts"
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

	am, err := accounts.GetAccountManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get account service: %v\n", err)
		os.Exit(1)
	}

	authTypes, err := am.GetAuthenticatorTypes(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAuthenticatorTypes: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d authenticator types:\n\n", len(authTypes))
	for i, a := range authTypes {
		fmt.Printf("  [%2d] type=%-40s package=%s\n", i+1, a.Type, a.PackageName)
	}
}
