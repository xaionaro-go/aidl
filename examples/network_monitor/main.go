// Check network connectivity status via NetworkManagementService.
//
// IConnectivityManager is a Java-only AIDL interface, so transaction codes
// cannot be resolved for it. Instead, we use INetworkManagementService
// (service name "network_management") which has full AIDL support and
// provides network interface listing, firewall state, tethering status,
// and bandwidth control information.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/network_monitor ./examples/network_monitor/
//	adb push build/network_monitor /data/local/tmp/ && adb shell /data/local/tmp/network_monitor
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

	// Connectivity service (Java-only AIDL, can only ping)
	connSvc, err := sm.CheckService(ctx, servicemanager.ConnectivityService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check connectivity service: %v\n", err)
	} else if connSvc == nil {
		fmt.Println("Connectivity service: not registered")
	} else {
		alive := connSvc.IsAlive(ctx)
		fmt.Printf("Connectivity service: alive=%v handle=%d\n", alive, connSvc.Handle())
	}

	// Network Management Service (full AIDL support)
	netSvc, err := sm.GetService(ctx, servicemanager.NetworkmanagementService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get network_management service: %v\n", err)
		os.Exit(1)
	}

	net := genOs.NewNetworkManagementServiceProxy(netSvc)

	// List network interfaces
	ifaces, err := net.ListInterfaces(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListInterfaces: %v\n", err)
	} else {
		fmt.Printf("\nNetwork interfaces: %d\n", len(ifaces))
		for _, iface := range ifaces {
			fmt.Printf("  %s\n", iface)
		}
	}

	// IP forwarding status
	fwdEnabled, err := net.GetIpForwardingEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetIpForwardingEnabled: %v\n", err)
	} else {
		fmt.Printf("\nIP forwarding:     %v\n", fwdEnabled)
	}

	// Bandwidth control
	bwControl, err := net.IsBandwidthControlEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsBandwidthControlEnabled: %v\n", err)
	} else {
		fmt.Printf("Bandwidth control: %v\n", bwControl)
	}

	// Firewall state
	fwEnabled, err := net.IsFirewallEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsFirewallEnabled: %v\n", err)
	} else {
		fmt.Printf("Firewall enabled:  %v\n", fwEnabled)
	}

	// Tethering status
	tethering, err := net.IsTetheringStarted(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsTetheringStarted: %v\n", err)
	} else {
		fmt.Printf("Tethering active:  %v\n", tethering)
	}

	// Tethered interfaces
	tethered, err := net.ListTetheredInterfaces(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListTetheredInterfaces: %v\n", err)
	} else if len(tethered) > 0 {
		fmt.Printf("Tethered ifaces:   %v\n", tethered)
	}

	// Network restriction check for our UID
	uid := int32(os.Getuid())
	restricted, err := net.IsNetworkRestricted(ctx, uid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsNetworkRestricted(%d): %v\n", uid, err)
	} else {
		fmt.Printf("UID %d restricted: %v\n", uid, restricted)
	}
}
