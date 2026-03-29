// Scan available WiFi networks via the wificond system service.
//
// Uses IWificond via the "wifinl80211" service to get client interfaces,
// then uses the IWifiScannerImpl from a client interface to retrieve
// scan results. Also queries available channels and PHY capabilities.
//
// Note: Requires root or appropriate SELinux permissions to interact
// with wificond. The scan results structure is empty in generated code
// (NativeScanResult fields not yet mapped), so only the count is shown.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/wifi_scanner ./examples/wifi_scanner/
//	adb push build/wifi_scanner /data/local/tmp/ && adb shell /data/local/tmp/wifi_scanner
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/net/wifi/nl80211"
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

	svc, err := sm.GetService(ctx, servicemanager.WifiNl80211Service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get wifinl80211 service: %v\n", err)
		fmt.Fprintf(os.Stderr, "(wificond not available or access denied)\n")
		os.Exit(1)
	}

	wificond := nl80211.NewWificondProxy(svc)

	// List client (STA) interfaces and try to scan from each.
	clientIfaces, err := wificond.GetClientInterfaces(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetClientInterfaces: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Client interfaces found: %d\n", len(clientIfaces))

	for i, ifaceBinder := range clientIfaces {
		client := nl80211.NewClientInterfaceProxy(ifaceBinder)

		ifName, err := client.GetInterfaceName(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [%d] GetInterfaceName: %v\n", i, err)
			continue
		}
		fmt.Printf("\n  Interface %d: %s\n", i, ifName)

		mac, err := client.GetMacAddress(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    GetMacAddress: %v\n", err)
		} else {
			fmt.Printf("    MAC: %x\n", mac)
		}

		scanner, err := client.GetWifiScannerImpl(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    GetWifiScannerImpl: %v\n", err)
			continue
		}

		maxSSIDs, err := scanner.GetMaxSsidsPerScan(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    GetMaxSsidsPerScan: %v\n", err)
		} else {
			fmt.Printf("    Max SSIDs per scan: %d\n", maxSSIDs)
		}

		results, err := scanner.GetScanResults(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    GetScanResults: %v\n", err)
			continue
		}
		fmt.Printf("    Cached scan results: %d networks\n", len(results))
	}

	// Show available channels per band.
	fmt.Println("\nAvailable channels by band:")
	bands := []struct {
		name string
		fn   func(context.Context) ([]int32, error)
	}{
		{"2.4 GHz", wificond.GetAvailable2gChannels},
		{"5 GHz (non-DFS)", wificond.GetAvailable5gNonDFSChannels},
		{"DFS", wificond.GetAvailableDFSChannels},
		{"6 GHz", wificond.GetAvailable6gChannels},
		{"60 GHz", wificond.GetAvailable60gChannels},
	}
	for _, band := range bands {
		channels, err := band.fn(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", band.name, err)
			continue
		}
		fmt.Printf("  %-20s %v\n", band.name+":", channels)
	}
}
