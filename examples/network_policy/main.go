// Query network policy settings via the INetworkPolicyManager system service.
//
// Uses INetworkPolicyManager via the "netpolicy" service to check background
// data restriction, per-UID policy, and metered network restrictions.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/network_policy ./examples/network_policy/
//	adb push build/network_policy /data/local/tmp/ && adb shell /data/local/tmp/network_policy
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

	policyMgr, err := net.GetNetworkPolicyManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get netpolicy service: %v\n", err)
		os.Exit(1)
	}

	// Check global background data restriction.
	restricted, err := policyMgr.GetRestrictBackground(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetRestrictBackground: %v\n", err)
	} else {
		fmt.Printf("Background data restricted: %v\n", restricted)
	}

	// Check background restriction from caller's perspective.
	callerStatus, err := policyMgr.GetRestrictBackgroundByCaller(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetRestrictBackgroundByCaller: %v\n", err)
	} else {
		fmt.Printf("Restrict background (caller): %d\n", callerStatus)
	}

	// Check per-UID policy for our process.
	uid := int32(os.Getuid())
	uidPolicy, err := policyMgr.GetUidPolicy(ctx, uid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetUidPolicy(%d): %v\n", uid, err)
	} else {
		fmt.Printf("UID %d policy: %d\n", uid, uidPolicy)
	}

	// Check if our UID is restricted on metered networks.
	meteredRestricted, err := policyMgr.IsUidRestrictedOnMeteredNetworks(ctx, uid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsUidRestrictedOnMeteredNetworks(%d): %v\n", uid, err)
	} else {
		fmt.Printf("UID %d restricted on metered: %v\n", uid, meteredRestricted)
	}

	// Check if our UID has networking blocked.
	blocked, err := policyMgr.IsUidNetworkingBlocked(ctx, uid, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsUidNetworkingBlocked(%d, non-metered): %v\n", uid, err)
	} else {
		fmt.Printf("UID %d networking blocked (non-metered): %v\n", uid, blocked)
	}

	blockedMetered, err := policyMgr.IsUidNetworkingBlocked(ctx, uid, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsUidNetworkingBlocked(%d, metered): %v\n", uid, err)
	} else {
		fmt.Printf("UID %d networking blocked (metered):     %v\n", uid, blockedMetered)
	}

	// List network policies.
	policies, err := policyMgr.GetNetworkPolicies(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetNetworkPolicies: %v\n", err)
	} else {
		fmt.Printf("\nNetwork policies (%d):\n", len(policies))
		for i, p := range policies {
			fmt.Printf("  [%d] %+v\n", i, p)
		}
	}
}
