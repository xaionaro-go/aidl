// Check VPN status via the IVpnManager system service.
//
// Uses IVpnManager via the "vpn_management" service to query always-on
// VPN configuration, lockdown status, and legacy VPN info.
//
// Note: Most VPN queries require system-level permissions.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/vpn_monitor ./examples/vpn_monitor/
//	adb push build/vpn_monitor /data/local/tmp/ && adb shell /data/local/tmp/vpn_monitor
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/net"
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

	vpnMgr, err := net.GetVpnManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get vpn_management service: %v\n", err)
		os.Exit(1)
	}

	// Check always-on VPN package.
	alwaysOnPkg, err := vpnMgr.GetAlwaysOnVpnPackage(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAlwaysOnVpnPackage: %v\n", err)
	} else if alwaysOnPkg == "" {
		fmt.Println("Always-on VPN: (none configured)")
	} else {
		fmt.Printf("Always-on VPN package: %s\n", alwaysOnPkg)
	}

	// Check lockdown mode.
	lockdown, err := vpnMgr.IsVpnLockdownEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsVpnLockdownEnabled: %v\n", err)
	} else {
		fmt.Printf("VPN lockdown enabled: %v\n", lockdown)
	}

	// Check if caller is the always-on VPN app.
	isAlwaysOn, err := vpnMgr.IsCallerCurrentAlwaysOnVpnApp(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsCallerCurrentAlwaysOnVpnApp: %v\n", err)
	} else {
		fmt.Printf("Caller is always-on VPN app: %v\n", isAlwaysOn)
	}

	// Get lockdown allowlist.
	allowlist, err := vpnMgr.GetVpnLockdownAllowlist(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetVpnLockdownAllowlist: %v\n", err)
	} else if len(allowlist) == 0 {
		fmt.Println("VPN lockdown allowlist: (empty)")
	} else {
		fmt.Printf("VPN lockdown allowlist (%d apps):\n", len(allowlist))
		for _, pkg := range allowlist {
			fmt.Printf("  %s\n", pkg)
		}
	}

	// Get legacy VPN info.
	legacyInfo, err := vpnMgr.GetLegacyVpnInfo(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetLegacyVpnInfo: %v\n", err)
	} else {
		fmt.Printf("Legacy VPN info: %+v\n", legacyInfo)
	}
}
