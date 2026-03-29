//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AndroidGoLab/binder/android/app"
	"github.com/AndroidGoLab/binder/android/apphibernation"
	"github.com/AndroidGoLab/binder/android/content/pm"
	"github.com/AndroidGoLab/binder/android/net"
	"github.com/AndroidGoLab/binder/android/net/wifi/nl80211"
	"github.com/AndroidGoLab/binder/android/system/net/netd"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/android/app/usage"
	internalNet "github.com/AndroidGoLab/binder/com/android/internal_/net"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// --- Use case #32: WiFi hotspot (softap_manage) ---

func TestUseCase32_SoftapManage_ListInterfaces(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "network_management")
	netMgr := genOs.NewNetworkManagementServiceProxy(svc)

	ifaces, err := netMgr.ListInterfaces(ctx)
	requireOrSkip(t, err)
	assert.NotEmpty(t, ifaces, "expected at least one network interface")
	t.Logf("network interfaces: %d", len(ifaces))
}

func TestUseCase32_SoftapManage_BandwidthControl(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "network_management")
	netMgr := genOs.NewNetworkManagementServiceProxy(svc)

	bwCtrl, err := netMgr.IsBandwidthControlEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("bandwidth control enabled: %v", bwCtrl)
}

// --- Use case #33: WiFi scanner ---

func TestUseCase33_WifiScanner_GetClientInterfaces(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "wifinl80211")
	wificond := nl80211.NewWificondProxy(svc)

	clientIfaces, err := wificond.GetClientInterfaces(ctx)
	requireOrSkip(t, err)
	t.Logf("client interfaces: %d", len(clientIfaces))

	for i, ifaceBinder := range clientIfaces {
		client := nl80211.NewClientInterfaceProxy(ifaceBinder)
		ifName, err := client.GetInterfaceName(ctx)
		requireOrSkip(t, err)
		t.Logf("  [%d] interface name: %s", i, ifName)

		mac, err := client.GetMacAddress(ctx)
		requireOrSkip(t, err)
		t.Logf("  [%d] MAC: %x", i, mac)
	}
}

func TestUseCase33_WifiScanner_GetScanResults(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "wifinl80211")
	wificond := nl80211.NewWificondProxy(svc)

	clientIfaces, err := wificond.GetClientInterfaces(ctx)
	requireOrSkip(t, err)
	if len(clientIfaces) == 0 {
		t.Skip("no WiFi client interfaces available")
	}

	client := nl80211.NewClientInterfaceProxy(clientIfaces[0])
	scanner, err := client.GetWifiScannerImpl(ctx)
	requireOrSkip(t, err)

	maxSSIDs, err := scanner.GetMaxSsidsPerScan(ctx)
	requireOrSkip(t, err)
	assert.Greater(t, maxSSIDs, int32(0), "max SSIDs per scan should be > 0")
	t.Logf("max SSIDs per scan: %d", maxSSIDs)

	results, err := scanner.GetScanResults(ctx)
	requireOrSkip(t, err)
	t.Logf("cached scan results: %d", len(results))
}

func TestUseCase33_WifiScanner_AvailableChannels(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "wifinl80211")
	wificond := nl80211.NewWificondProxy(svc)

	ch2g, err := wificond.GetAvailable2gChannels(ctx)
	requireOrSkip(t, err)
	t.Logf("2.4 GHz channels: %v", ch2g)

	ch5g, err := wificond.GetAvailable5gNonDFSChannels(ctx)
	requireOrSkip(t, err)
	t.Logf("5 GHz (non-DFS) channels: %v", ch5g)
}

// --- Use case #34: Tethering offload (softap_tether_offload) ---

func TestUseCase34_TetheringOffload_IsTetheringStarted(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "network_management")
	netMgr := genOs.NewNetworkManagementServiceProxy(svc)

	tethering, err := netMgr.IsTetheringStarted(ctx)
	requireOrSkip(t, err)
	t.Logf("tethering started: %v", tethering)
}

// --- Use case #35: WiFi HAL diagnostics (softap_wifi_hal) ---

func TestUseCase35_WifiHAL_PhyCapabilities(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "wifinl80211")
	wificond := nl80211.NewWificondProxy(svc)

	caps, err := wificond.GetDeviceWiphyCapabilities(ctx, "wlan0")
	requireOrSkip(t, err)
	t.Logf("wlan0 PHY capabilities: %+v", caps)
}

func TestUseCase35_WifiHAL_ApInterfaces(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "wifinl80211")
	wificond := nl80211.NewWificondProxy(svc)

	apIfaces, err := wificond.GetApInterfaces(ctx)
	requireOrSkip(t, err)
	t.Logf("AP interfaces: %d", len(apIfaces))
}

// --- Use case #36: Network policy ---

func TestUseCase36_NetworkPolicy_GetRestrictBackground(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	policyMgr, err := net.GetNetworkPolicyManager(ctx, sm)
	requireOrSkip(t, err)

	restricted, err := policyMgr.GetRestrictBackground(ctx)
	requireOrSkip(t, err)
	t.Logf("background data restricted: %v", restricted)
}

func TestUseCase36_NetworkPolicy_GetUidPolicy(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	policyMgr, err := net.GetNetworkPolicyManager(ctx, sm)
	requireOrSkip(t, err)

	uid := int32(os.Getuid())
	policy, err := policyMgr.GetUidPolicy(ctx, uid)
	requireOrSkip(t, err)
	t.Logf("UID %d policy: %d", uid, policy)
}

func TestUseCase36_NetworkPolicy_IsUidNetworkingBlocked(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	policyMgr, err := net.GetNetworkPolicyManager(ctx, sm)
	requireOrSkip(t, err)

	uid := int32(os.Getuid())
	blocked, err := policyMgr.IsUidNetworkingBlocked(ctx, uid, false)
	requireOrSkip(t, err)
	t.Logf("UID %d networking blocked (non-metered): %v", uid, blocked)
}

func TestUseCase36_NetworkPolicy_IsUidRestrictedOnMetered(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	policyMgr, err := net.GetNetworkPolicyManager(ctx, sm)
	requireOrSkip(t, err)

	uid := int32(os.Getuid())
	restricted, err := policyMgr.IsUidRestrictedOnMeteredNetworks(ctx, uid)
	requireOrSkip(t, err)
	t.Logf("UID %d restricted on metered: %v", uid, restricted)
}

func TestUseCase36_NetworkPolicy_GetNetworkPolicies(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	policyMgr, err := net.GetNetworkPolicyManager(ctx, sm)
	requireOrSkip(t, err)

	policies, err := policyMgr.GetNetworkPolicies(ctx)
	requireOrSkip(t, err)
	t.Logf("network policies: %d", len(policies))
	for i, p := range policies {
		t.Logf("  [%d] warning=%d limit=%d metered=%v", i, p.WarningBytes, p.LimitBytes, p.Metered)
	}
}

func TestUseCase36_NetworkPolicy_GetRestrictBackgroundByCaller(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	policyMgr, err := net.GetNetworkPolicyManager(ctx, sm)
	requireOrSkip(t, err)

	status, err := policyMgr.GetRestrictBackgroundByCaller(ctx)
	requireOrSkip(t, err)
	t.Logf("restrict background by caller: %d", status)
}

// --- Use case #37: VPN monitor ---

func TestUseCase37_VpnMonitor_GetAlwaysOnVpnPackage(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	vpnMgr, err := net.GetVpnManager(ctx, sm)
	requireOrSkip(t, err)

	pkg, err := vpnMgr.GetAlwaysOnVpnPackage(ctx)
	requireOrSkip(t, err)
	t.Logf("always-on VPN package: %q", pkg)
}

func TestUseCase37_VpnMonitor_IsVpnLockdownEnabled(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	vpnMgr, err := net.GetVpnManager(ctx, sm)
	requireOrSkip(t, err)

	lockdown, err := vpnMgr.IsVpnLockdownEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("VPN lockdown enabled: %v", lockdown)
}

func TestUseCase37_VpnMonitor_IsCallerCurrentAlwaysOnVpnApp(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	vpnMgr, err := net.GetVpnManager(ctx, sm)
	requireOrSkip(t, err)

	isAlwaysOn, err := vpnMgr.IsCallerCurrentAlwaysOnVpnApp(ctx)
	requireOrSkip(t, err)
	assert.False(t, isAlwaysOn, "test process should not be the always-on VPN app")
	t.Logf("caller is always-on VPN app: %v", isAlwaysOn)
}

func TestUseCase37_VpnMonitor_GetVpnLockdownAllowlist(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	vpnMgr, err := net.GetVpnManager(ctx, sm)
	requireOrSkip(t, err)

	allowlist, err := vpnMgr.GetVpnLockdownAllowlist(ctx)
	requireOrSkip(t, err)
	t.Logf("VPN lockdown allowlist: %v", allowlist)
}

// --- Use case #38: DNS config ---

func TestUseCase38_DnsConfig_ListInterfaces(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "network_management")
	netMgr := genOs.NewNetworkManagementServiceProxy(svc)

	ifaces, err := netMgr.ListInterfaces(ctx)
	requireOrSkip(t, err)
	assert.NotEmpty(t, ifaces)
	t.Logf("interfaces: %v", ifaces)
}

func TestUseCase38_DnsConfig_NetdPing(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "netd")

	alive := svc.IsAlive(ctx)
	assert.True(t, alive, "netd service should be alive")
	t.Logf("netd alive: %v", alive)
}

func TestUseCase38_DnsConfig_NetdCreateDestroyOemNetwork(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "netd")
	netdProxy := netd.NewNetdProxy(svc)

	oemNet, err := netdProxy.CreateOemNetwork(ctx)
	requireOrSkip(t, err)
	assert.NotZero(t, oemNet.NetworkHandle, "OEM network handle should be non-zero")
	t.Logf("OEM network: handle=%d, packetMark=%d", oemNet.NetworkHandle, oemNet.PacketMark)

	err = netdProxy.DestroyOemNetwork(ctx, oemNet.NetworkHandle)
	requireOrSkip(t, err)
	t.Log("OEM network destroyed")
}

func TestUseCase38_DnsConfig_NetworkWatchlistHash(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "network_watchlist")
	watchlistMgr := internalNet.NewNetworkWatchlistManagerProxy(svc)

	hash, err := watchlistMgr.GetWatchlistConfigHash(ctx)
	requireOrSkip(t, err)
	t.Logf("watchlist config hash: %x (%d bytes)", hash, len(hash))
}

// --- Use case #39: Installed packages (list_packages) ---

func TestUseCase39_ListPackages_GetAllPackages(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	packages, err := pkgMgr.GetAllPackages(ctx)
	requireOrSkip(t, err)
	require.NotEmpty(t, packages, "expected at least one package")
	t.Logf("installed packages: %d", len(packages))

	assert.Contains(t, packages, "com.android.settings",
		"com.android.settings should be installed")
}

func TestUseCase39_ListPackages_GetPackagesForUid(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	uid := int32(os.Getuid())
	packages, err := pkgMgr.GetPackagesForUid(ctx, uid)
	requireOrSkip(t, err)
	t.Logf("packages for UID %d: %v", uid, packages)
}

// --- Use case #40: Package monitor ---

func TestUseCase40_PackageMonitor_SnapshotAndCompare(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	// Take two snapshots and verify they are consistent.
	snap1, err := pkgMgr.GetAllPackages(ctx)
	requireOrSkip(t, err)

	snap2, err := pkgMgr.GetAllPackages(ctx)
	requireOrSkip(t, err)

	// Package lists should be identical in rapid succession.
	assert.Equal(t, len(snap1), len(snap2),
		"two rapid snapshots should have the same package count")
	t.Logf("snapshot 1: %d packages, snapshot 2: %d packages", len(snap1), len(snap2))
}

func TestUseCase40_PackageMonitor_PackageAvailability(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	// Check availability of a known package vs a nonexistent one.
	available, err := pkgMgr.IsPackageAvailable(ctx, "com.android.settings")
	requireOrSkip(t, err)
	assert.True(t, available, "com.android.settings should be available")

	notAvailable, err := pkgMgr.IsPackageAvailable(ctx, "com.example.definitely.does.not.exist")
	requireOrSkip(t, err)
	assert.False(t, notAvailable, "nonexistent package should not be available")
}

// --- Use case #41: Permission audit ---

func TestUseCase41_PermissionAudit_CheckPermission_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "activity")
	am := app.NewActivityManagerProxy(svc)

	// Root (uid=0) should have INTERNET permission.
	result, err := am.CheckPermission(ctx, "android.permission.INTERNET", int32(os.Getpid()), 0)
	requireOrSkip(t, err)
	assert.Equal(t, int32(0), result, "root should have INTERNET permission (0=GRANTED)")
	t.Logf("checkPermission(INTERNET, uid=0): %d", result)
}

func TestUseCase41_PermissionAudit_CheckPermission_CurrentProcess(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "activity")
	am := app.NewActivityManagerProxy(svc)

	permissions := []string{
		"android.permission.INTERNET",
		"android.permission.CAMERA",
		"android.permission.READ_PHONE_STATE",
		"android.permission.ACCESS_FINE_LOCATION",
	}

	pid := int32(os.Getpid())
	uid := int32(os.Getuid())
	for _, perm := range permissions {
		result, err := am.CheckPermission(ctx, perm, pid, uid)
		requireOrSkip(t, err)
		status := "DENIED"
		if result == 0 {
			status = "GRANTED"
		}
		t.Logf("  %s: %s (%d)", perm, status, result)
	}
}

func TestUseCase41_PermissionAudit_GetProcessLimit(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "activity")
	am := app.NewActivityManagerProxy(svc)

	limit, err := am.GetProcessLimit(ctx)
	requireOrSkip(t, err)
	t.Logf("process limit: %d", limit)
}

func TestUseCase41_PermissionAudit_GetRunningAppProcesses(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "activity")
	am := app.NewActivityManagerProxy(svc)

	procs, err := am.GetRunningAppProcesses(ctx)
	requireOrSkip(t, err)
	assert.NotEmpty(t, procs, "expected at least one running process")
	t.Logf("running processes: %d", len(procs))
}

// --- Use case #42: Version checking (package_query) ---

func TestUseCase42_PackageQuery_IsPackageAvailable(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	available, err := pkgMgr.IsPackageAvailable(ctx, "com.android.shell")
	requireOrSkip(t, err)
	assert.True(t, available, "com.android.shell should be available")
}

func TestUseCase42_PackageQuery_GetTargetSdkVersion(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	sdk, err := pkgMgr.GetTargetSdkVersion(ctx, "com.android.settings")
	requireOrSkip(t, err)
	assert.Greater(t, sdk, int32(0), "target SDK should be > 0")
	t.Logf("com.android.settings target SDK: %d", sdk)
}

func TestUseCase42_PackageQuery_GetInstallerPackageName(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	installer, err := pkgMgr.GetInstallerPackageName(ctx, "com.android.settings")
	requireOrSkip(t, err)
	t.Logf("com.android.settings installer: %q", installer)
}

func TestUseCase42_PackageQuery_GetPackageInfo(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	info, err := pkgMgr.GetPackageInfo(ctx, "com.android.settings", 0)
	requireOrSkip(t, err)
	assert.Equal(t, "com.android.settings", info.PackageName)
	t.Logf("com.android.settings: version=%q, versionCode=%d, firstInstall=%d",
		info.VersionName, info.VersionCode, info.FirstInstallTime)
}

// --- Use case #43: System app classifier ---

func TestUseCase43_SystemAppClassifier_FlagCheck(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	// ApplicationInfo is an opaque Java Parcelable — its fields (Flags, Uid,
	// TargetSdkVersion) are always zero in our generated code. Instead, detect
	// system apps by checking the installer: system apps have no installer
	// (empty string), while user-installed apps have an installer like
	// "com.android.vending".
	installer, err := pkgMgr.GetInstallerPackageName(ctx, "com.android.settings")
	requireOrSkip(t, err)
	isSystem := installer == ""
	assert.True(t, isSystem,
		"com.android.settings should be a system app (no installer), got installer=%q", installer)
	t.Logf("com.android.settings: installer=%q, system=%v", installer, isSystem)
}

func TestUseCase43_SystemAppClassifier_ClassifyMultiple(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	// Use GetInstallerPackageName instead of ApplicationInfo.Flags, since
	// ApplicationInfo is an opaque Java Parcelable (all fields decode as zero).
	packages := []string{
		"com.android.settings",
		"com.android.systemui",
		"com.android.shell",
	}

	for _, pkg := range packages {
		installer, err := pkgMgr.GetInstallerPackageName(ctx, pkg)
		requireOrSkip(t, err)

		isSystem := installer == ""
		t.Logf("  %-40s system=%v installer=%q", pkg, isSystem, installer)
		assert.True(t, isSystem, "%s should be a system app (no installer)", pkg)
	}
}

func TestUseCase43_SystemAppClassifier_CountSystemVsUser(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "package")
	pkgMgr := pm.NewPackageManagerProxy(svc)

	// Use GetInstallerPackageName instead of ApplicationInfo.Flags, since
	// ApplicationInfo is an opaque Java Parcelable (all fields decode as zero).
	allPkgs, err := pkgMgr.GetAllPackages(ctx)
	requireOrSkip(t, err)

	var systemCount, userCount, errorCount int
	// Check first 20 packages to avoid excessive calls.
	limit := 20
	if len(allPkgs) < limit {
		limit = len(allPkgs)
	}
	for _, pkg := range allPkgs[:limit] {
		installer, err := pkgMgr.GetInstallerPackageName(ctx, pkg)
		if err != nil {
			errorCount++
			continue
		}
		if installer == "" {
			systemCount++
		} else {
			userCount++
		}
	}

	t.Logf("of %d checked: system=%d, user=%d, errors=%d",
		limit, systemCount, userCount, errorCount)
	assert.Greater(t, systemCount, 0, "expected at least one system app")
}

// --- Use case #44: App hibernation ---

func TestUseCase44_AppHibernation_IsOatArtifactDeletionEnabled(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "app_hibernation")
	hibMgr := apphibernation.NewAppHibernationServiceProxy(svc)

	enabled, err := hibMgr.IsOatArtifactDeletionEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("OAT artifact deletion enabled: %v", enabled)
}

func TestUseCase44_AppHibernation_GetHibernatingPackagesForUser(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "app_hibernation")
	hibMgr := apphibernation.NewAppHibernationServiceProxy(svc)

	pkgs, err := hibMgr.GetHibernatingPackagesForUser(ctx)
	requireOrSkip(t, err)
	t.Logf("hibernating packages: %d", len(pkgs))
	for i, pkg := range pkgs {
		if i < 10 {
			t.Logf("  %s", pkg)
		}
	}
}

func TestUseCase44_AppHibernation_IsHibernatingGlobally(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "app_hibernation")
	hibMgr := apphibernation.NewAppHibernationServiceProxy(svc)

	hibernating, err := hibMgr.IsHibernatingGlobally(ctx, "com.android.settings")
	requireOrSkip(t, err)
	assert.False(t, hibernating, "com.android.settings should not be globally hibernating")
	t.Logf("com.android.settings globally hibernating: %v", hibernating)
}

func TestUseCase44_AppHibernation_IsHibernatingForUser(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "app_hibernation")
	hibMgr := apphibernation.NewAppHibernationServiceProxy(svc)

	hibernating, err := hibMgr.IsHibernatingForUser(ctx, "com.android.settings")
	requireOrSkip(t, err)
	assert.False(t, hibernating, "com.android.settings should not be hibernating for user")
	t.Logf("com.android.settings user-hibernating: %v", hibernating)
}

// --- Use case #45: Usage stats ---

func TestUseCase45_UsageStats_IsAppStandbyEnabled(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	usageMgr, err := usage.GetUsageStatsManager(ctx, sm)
	requireOrSkip(t, err)

	enabled, err := usageMgr.IsAppStandbyEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("app standby enabled: %v", enabled)
}

func TestUseCase45_UsageStats_GetAppStandbyBucket(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	usageMgr, err := usage.GetUsageStatsManager(ctx, sm)
	requireOrSkip(t, err)

	bucket, err := usageMgr.GetAppStandbyBucket(ctx, "com.android.settings")
	requireOrSkip(t, err)
	t.Logf("com.android.settings standby bucket: %d", bucket)
}

func TestUseCase45_UsageStats_IsAppInactive(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	usageMgr, err := usage.GetUsageStatsManager(ctx, sm)
	requireOrSkip(t, err)

	inactive, err := usageMgr.IsAppInactive(ctx, "com.android.settings")
	requireOrSkip(t, err)
	t.Logf("com.android.settings inactive: %v", inactive)
}

func TestUseCase45_UsageStats_GetUsageSource(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	usageMgr, err := usage.GetUsageStatsManager(ctx, sm)
	requireOrSkip(t, err)

	source, err := usageMgr.GetUsageSource(ctx)
	requireOrSkip(t, err)
	t.Logf("usage source: %d", source)
}

func TestUseCase45_UsageStats_MultiplePackageBuckets(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	usageMgr, err := usage.GetUsageStatsManager(ctx, sm)
	requireOrSkip(t, err)

	packages := []string{
		"com.android.settings",
		"com.android.systemui",
		"com.android.shell",
	}

	for _, pkg := range packages {
		bucket, err := usageMgr.GetAppStandbyBucket(ctx, pkg)
		if err != nil {
			t.Logf("  %-40s bucket: error (%v)", pkg, err)
			continue
		}
		inactive, err := usageMgr.IsAppInactive(ctx, pkg)
		if err != nil {
			t.Logf("  %-40s bucket=%d, inactive: error", pkg, bucket)
			continue
		}
		t.Logf("  %-40s bucket=%d, inactive=%v", pkg, bucket, inactive)
	}
}
