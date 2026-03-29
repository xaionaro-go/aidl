//go:build e2e_root

package e2e

// Root-tier E2E tests. These tests require:
//   - UID 0 (adb root)
//   - Permissive SELinux (setenforce 0)
//
// They exercise HAL services and privileged system services that are
// blocked by SELinux for shell (UID 2000) but work with root access.
//
// Build: go test -tags e2e_root -c -o build/e2e_root_test ./tests/e2e/
// Run:   adb root && adb shell setenforce 0
//        adb push build/e2e_root_test /data/local/tmp/
//        adb shell /data/local/tmp/e2e_root_test -test.timeout 300s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genApp "github.com/AndroidGoLab/binder/android/app"
	genAdmin "github.com/AndroidGoLab/binder/android/app/admin"
	"github.com/AndroidGoLab/binder/android/content"
	fwkService "github.com/AndroidGoLab/binder/android/frameworks/cameraservice/service"
	"github.com/AndroidGoLab/binder/android/hardware"
	genMedia "github.com/AndroidGoLab/binder/android/media"
	"github.com/AndroidGoLab/binder/android/net/wifi/nl80211"
	"github.com/AndroidGoLab/binder/android/system/net/netd"
	genOs "github.com/AndroidGoLab/binder/android/os"
	genKeystore2 "github.com/AndroidGoLab/binder/android/system/keystore2"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// ---------------------------------------------------------------------------
// #22: SetTorchMode (camera SELinux: kernel status -61 as shell)
// ---------------------------------------------------------------------------

func TestUseCase22_SetTorchMode_Root(t *testing.T) {
	ctx := context.Background()

	driver, transport := openBinderDirect(ctx, t)
	defer func() { _ = driver.Close(ctx) }()
	defer func() { _ = transport.Close(ctx) }()

	sm := servicemanager.New(transport)

	// Enumerate cameras via framework camera service.
	fwkSvc, err := sm.GetService(ctx, "android.frameworks.cameraservice.service.ICameraService/default")
	requireOrSkip(t, err)
	fwkCam := fwkService.NewCameraServiceProxy(fwkSvc)

	listener := fwkService.NewCameraServiceListenerStub(&noopCameraServiceListener{})
	cameras, err := fwkCam.AddListener(ctx, listener)
	requireOrSkip(t, err)
	defer func() { _ = fwkCam.RemoveListener(ctx, listener) }()
	require.Greater(t, len(cameras), 0, "expected at least one camera")

	// Torch control via media.camera (requires root to bypass SELinux).
	mediaSvc, err := sm.GetService(ctx, servicemanager.MediaCameraService)
	requireOrSkip(t, err)
	cam := hardware.NewCameraServiceProxy(mediaSvc)

	token := binder.NewStubBinder(&torchClientToken{})
	token.RegisterWithTransport(ctx, transport)

	err = cam.SetTorchMode(ctx, cameras[0].CameraId, true, token)
	requireOrSkip(t, err)
	t.Log("Torch ON")

	err = cam.SetTorchMode(ctx, cameras[0].CameraId, false, token)
	requireOrSkip(t, err)
	t.Log("Torch OFF")
}

// ---------------------------------------------------------------------------
// #23: GetCameraCharacteristics (HAL ServiceSpecific error as shell)
// ---------------------------------------------------------------------------

func TestUseCase23_GetCameraCharacteristics_Root(t *testing.T) {
	ctx := context.Background()

	driver, transport := openBinderDirect(ctx, t)
	defer func() { _ = driver.Close(ctx) }()
	defer func() { _ = transport.Close(ctx) }()

	sm := servicemanager.New(transport)

	fwkSvc, err := sm.GetService(ctx, "android.frameworks.cameraservice.service.ICameraService/default")
	requireOrSkip(t, err)
	fwkCam := fwkService.NewCameraServiceProxy(fwkSvc)

	listener := fwkService.NewCameraServiceListenerStub(&noopCameraServiceListener{})
	cameras, err := fwkCam.AddListener(ctx, listener)
	requireOrSkip(t, err)
	defer func() { _ = fwkCam.RemoveListener(ctx, listener) }()

	if len(cameras) == 0 {
		t.Skip("no cameras available")
	}

	chars, err := fwkCam.GetCameraCharacteristics(ctx, cameras[0].CameraId)
	requireOrSkip(t, err)
	t.Logf("Camera %q characteristics: %d bytes of metadata",
		cameras[0].CameraId, len(chars.Metadata))
	assert.Greater(t, len(chars.Metadata), 0, "expected non-empty camera metadata")
}

// ---------------------------------------------------------------------------
// #33: WiFi scanner (wificond SELinux denial as shell)
// ---------------------------------------------------------------------------

func TestUseCase33_WifiScanner_GetClientInterfaces_Root(t *testing.T) {
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

func TestUseCase33_WifiScanner_GetScanResults_Root(t *testing.T) {
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

func TestUseCase33_WifiScanner_AvailableChannels_Root(t *testing.T) {
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

// ---------------------------------------------------------------------------
// #35: WiFi HAL diagnostics (wificond SELinux denial as shell)
// ---------------------------------------------------------------------------

func TestUseCase35_WifiHAL_PhyCapabilities_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "wifinl80211")
	wificond := nl80211.NewWificondProxy(svc)

	caps, err := wificond.GetDeviceWiphyCapabilities(ctx, "wlan0")
	requireOrSkip(t, err)
	t.Logf("wlan0 PHY capabilities: %+v", caps)
}

func TestUseCase35_WifiHAL_ApInterfaces_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "wifinl80211")
	wificond := nl80211.NewWificondProxy(svc)

	apIfaces, err := wificond.GetApInterfaces(ctx)
	requireOrSkip(t, err)
	t.Logf("AP interfaces: %d", len(apIfaces))
}

// ---------------------------------------------------------------------------
// #47: Keystore operations (keystore2 access denied as shell)
// ---------------------------------------------------------------------------

func TestUsecase_KeystoreOps_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	const svcName = "android.system.keystore2.IKeystoreService/default"
	svc, err := sm.CheckService(ctx, servicemanager.ServiceName(svcName))
	requireOrSkip(t, err)
	if svc == nil {
		t.Skip("keystore2 service not registered")
	}
	require.True(t, svc.IsAlive(ctx), "keystore2 should be alive")

	ks := genKeystore2.NewKeystoreServiceProxy(svc)

	// Query number of entries in SELINUX domain (namespace 0).
	count, err := ks.GetNumberOfEntries(ctx, genKeystore2.DomainSELINUX, 0)
	requireOrSkip(t, err)
	t.Logf("GetNumberOfEntries(SELINUX, ns=0): %d", count)
	require.GreaterOrEqual(t, count, int32(0))

	// Query number of entries in APP domain.
	appCount, err := ks.GetNumberOfEntries(ctx, genKeystore2.DomainAPP, -1)
	requireOrSkip(t, err)
	t.Logf("GetNumberOfEntries(APP, ns=-1): %d", appCount)
	require.GreaterOrEqual(t, appCount, int32(0))
}

// ---------------------------------------------------------------------------
// #61: IsUltrasoundSupported (ACCESS_ULTRASOUND permission as shell)
// ---------------------------------------------------------------------------

func TestUseCase61_AudioFocus_IsUltrasoundSupported_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	supported, err := audio.IsUltrasoundSupported(ctx)
	requireOrSkip(t, err)
	t.Logf("Ultrasound supported: %v", supported)
}

// ---------------------------------------------------------------------------
// #66: GetActiveNotifications (ACCESS_NOTIFICATIONS permission as shell)
// ---------------------------------------------------------------------------

func TestUseCase66_NotificationListener_GetActiveNotifications_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	notifs, err := nm.GetActiveNotifications(ctx, "com.android.shell")
	requireOrSkip(t, err)
	t.Logf("Active notifications: %d", len(notifs))
}

// ---------------------------------------------------------------------------
// #68: ShouldHideSilentStatusIcons (notification listener access as shell)
// ---------------------------------------------------------------------------

func TestUseCase68_DNDController_ShouldHideSilentStatusIcons_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	hidden, err := nm.ShouldHideSilentStatusIcons(ctx, "com.android.shell")
	requireOrSkip(t, err)
	t.Logf("Hide silent status icons: %v", hidden)
}

// ---------------------------------------------------------------------------
// #84: DeviceProvisioned (system permission as shell)
// ---------------------------------------------------------------------------

func TestUseCase84_DeviceProvisioned_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	dpm, err := genAdmin.GetDevicePolicyManager(ctx, sm)
	requireOrSkip(t, err)

	provisioned, err := dpm.IsDeviceProvisioned(ctx)
	requireOrSkip(t, err)
	assert.True(t, provisioned, "device should be provisioned")
	t.Logf("Device provisioned: %v", provisioned)
}

// ---------------------------------------------------------------------------
// #85: SystemUpdate (READ_SYSTEM_UPDATE_INFO permission as shell)
// ---------------------------------------------------------------------------

func TestUseCase85_SystemUpdate_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	updateMgr, err := genOs.GetSystemUpdateManager(ctx, sm)
	requireOrSkip(t, err)

	info, err := updateMgr.RetrieveSystemUpdateInfo(ctx)
	requireOrSkip(t, err)
	_ = info
	t.Logf("System update info retrieved successfully")
}

// ---------------------------------------------------------------------------
// #88: OTA SystemUpdate (READ_SYSTEM_UPDATE_INFO permission as shell)
// ---------------------------------------------------------------------------

func TestUseCase88_SystemUpdate_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	updateMgr, err := genOs.GetSystemUpdateManager(ctx, sm)
	requireOrSkip(t, err)

	info, err := updateMgr.RetrieveSystemUpdateInfo(ctx)
	requireOrSkip(t, err)
	_ = info
	t.Logf("System update info bundle retrieved")
}

// ---------------------------------------------------------------------------
// #38: Netd CreateOemNetwork (SELinux denied as shell)
// ---------------------------------------------------------------------------

func TestUseCase38_DnsConfig_NetdCreateDestroyOemNetwork_Root(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "android.system.net.netd.INetd/default")
	netdProxy := netd.NewNetdProxy(svc)

	oemNet, err := netdProxy.CreateOemNetwork(ctx)
	requireOrSkip(t, err)
	assert.NotZero(t, oemNet.NetworkHandle, "OEM network handle should be non-zero")
	t.Logf("OEM network: handle=%d, packetMark=%d", oemNet.NetworkHandle, oemNet.PacketMark)

	err = netdProxy.DestroyOemNetwork(ctx, oemNet.NetworkHandle)
	requireOrSkip(t, err)
	t.Log("OEM network destroyed")
}

// Ensure imports are used.
var (
	_ = parcel.New
	_ = content.ComponentName{}
)
