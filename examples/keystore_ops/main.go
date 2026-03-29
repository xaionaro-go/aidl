// Query Keystore2 service for key entries and counts (read-only).
// WARNING: This example only performs read-only queries. No keys are
// generated, deleted, or modified.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/keystore_ops ./examples/keystore_ops/
//	adb push keystore_ops /data/local/tmp/ && adb shell /data/local/tmp/keystore_ops
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/system/keystore2"
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

	// Keystore2 is registered as "android.system.keystore2.IKeystoreService/default"
	// on the binder bus (HAL-style name).
	const svcName = "android.system.keystore2.IKeystoreService/default"
	svc, err := sm.GetService(ctx, servicemanager.ServiceName(svcName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get keystore2 service: %v\n", err)
		os.Exit(1)
	}

	ks := keystore2.NewKeystoreServiceProxy(svc)

	fmt.Println("=== Keystore2 Read-Only Queries ===")

	// Query number of entries in the APP domain (our own namespace).
	count, err := ks.GetNumberOfEntries(ctx, keystore2.DomainAPP, -1)
	if err != nil {
		fmt.Printf("GetNumberOfEntries(APP): %v\n", err)
	} else {
		fmt.Printf("Key entries in APP domain: %d\n", count)
	}

	// Query number of entries in the SELINUX domain (namespace 0).
	count, err = ks.GetNumberOfEntries(ctx, keystore2.DomainSELINUX, 0)
	if err != nil {
		fmt.Printf("GetNumberOfEntries(SELINUX): %v\n", err)
	} else {
		fmt.Printf("Key entries in SELINUX domain (ns=0): %d\n", count)
	}

	// List entries in APP domain.
	entries, err := ks.ListEntries(ctx, keystore2.DomainAPP, -1)
	if err != nil {
		fmt.Printf("ListEntries(APP): %v\n", err)
	} else {
		fmt.Printf("Listed %d entries in APP domain\n", len(entries))
		for i, e := range entries {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(entries)-10)
				break
			}
			fmt.Printf("  [%d] domain=%d nspace=%d alias=%q\n",
				i, e.Domain, e.Nspace, e.Alias)
		}
	}
}
