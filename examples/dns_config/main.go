// Query network configuration via the netd system service.
//
// Uses INetd via the "netd" service to check IP forwarding status
// and OEM network management. The netd service provides low-level
// network configuration including routing and interface management.
//
// Note: Most netd operations require root or AID_SYSTEM.
// There is no DNS resolver AIDL proxy in the generated code; DNS
// configuration is queried indirectly through netd's network management.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/dns_config ./examples/dns_config/
//	adb push build/dns_config /data/local/tmp/ && adb shell /data/local/tmp/dns_config
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/android/system/net/netd"
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

	// Use NetworkManagementService to list interfaces.
	nmSvc, err := sm.GetService(ctx, servicemanager.NetworkmanagementService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get network_management service: %v\n", err)
	} else {
		netMgr := genOs.NewNetworkManagementServiceProxy(nmSvc)

		ifaces, err := netMgr.ListInterfaces(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ListInterfaces: %v\n", err)
		} else {
			fmt.Printf("Network interfaces (%d):\n", len(ifaces))
			for _, iface := range ifaces {
				fmt.Printf("  %s\n", iface)
			}
		}

		bwCtrl, err := netMgr.IsBandwidthControlEnabled(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "IsBandwidthControlEnabled: %v\n", err)
		} else {
			fmt.Printf("\nBandwidth control enabled: %v\n", bwCtrl)
		}
	}

	// Use netd (android.system.net.netd.INetd) for low-level network ops.
	netdSvc, err := sm.GetService(ctx, servicemanager.NetdService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nget netd service: %v\n", err)
		fmt.Fprintf(os.Stderr, "(netd service not available or access denied)\n")
		return
	}

	netdProxy := netd.NewNetdProxy(netdSvc)

	// Create and immediately destroy an OEM network to verify netd access.
	oemNet, err := netdProxy.CreateOemNetwork(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCreateOemNetwork: %v (requires root)\n", err)
	} else {
		fmt.Printf("\nOEM network created: handle=%d, packetMark=%d\n",
			oemNet.NetworkHandle, oemNet.PacketMark)

		err = netdProxy.DestroyOemNetwork(ctx, oemNet.NetworkHandle)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DestroyOemNetwork: %v\n", err)
		} else {
			fmt.Println("OEM network destroyed successfully")
		}
	}
}
