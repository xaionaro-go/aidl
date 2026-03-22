//go:build linux

package main

import (
	"context"

	"github.com/mark3labs/mcp-go/server"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// ServiceLookup is the subset of servicemanager.ServiceManager used by the
// MCP tool handlers. Both the real ServiceManager (device mode) and the
// remote proxy implementation satisfy this interface.
type ServiceLookup interface {
	ListServices(ctx context.Context) ([]servicemanager.ServiceName, error)
	CheckService(ctx context.Context, name servicemanager.ServiceName) (binder.IBinder, error)
}

// ToolSet holds the binder connection state shared by all MCP tool handlers.
type ToolSet struct {
	sm ServiceLookup
}

// Register adds all binder MCP tools to the given server.
func (ts *ToolSet) Register(s *server.MCPServer) {
	ts.registerListServices(s)
	ts.registerGetServiceInfo(s)
	ts.registerCallMethod(s)
	ts.registerGetDeviceInfo(s)
	ts.registerGetLocation(s)
	ts.registerCheckPermissions(s)

	// Power & display.
	ts.registerIsScreenOn(s)
	ts.registerWakeScreen(s)
	ts.registerSleepScreen(s)
	ts.registerGetBrightness(s)
	ts.registerSetBrightness(s)
	ts.registerGetDisplaySize(s)

	// Package management.
	ts.registerListPackages(s)
	ts.registerGetPackageInfo(s)

	// App management (binder-based).
	ts.registerStopApp(s)

	// Camera.
	ts.registerListCameras(s)

	// Audio.
	ts.registerGetMediaVolume(s)
	ts.registerSetMediaVolume(s)

	// Bluetooth.
	ts.registerGetBluetoothState(s)
}

// RegisterShellTools adds all shell-based MCP tools to the given server.
// These tools use direct shell command execution and do not require a binder
// connection. They are available in both device and remote modes.
func RegisterShellTools(s *server.MCPServer) {
	// P0: Device info & diagnostics.
	registerGetDeviceProperties(s)
	registerGetDeviceSpecs(s)

	// P0: Screen capture.
	registerTakeScreenshot(s)

	// P0: Input & gestures.
	registerTap(s)
	registerLongPress(s)
	registerSwipe(s)
	registerInputText(s)
	registerPressKey(s)

	// P0: UI automation.
	registerDumpUIHierarchy(s)
	registerFindUIElement(s)
	registerClickUIElement(s)
	registerGetFocusedWindow(s)

	// P0: App management.
	registerInstallApp(s)
	registerUninstallApp(s)

	// P0: Settings.
	registerGetSetting(s)
	registerSetSetting(s)

	// P0: Dev tools.
	registerGetLogcat(s)
	registerShellExec(s)
	registerDumpService(s)

	// P0: Activity & intent.
	registerOpenURL(s)
	registerStartActivity(s)

	// P0: Clipboard (shell-based due to complex ClipData parcelable).
	registerGetClipboard(s)
	registerSetClipboard(s)

	// P0: App management (shell-based).
	registerLaunchApp(s)
	registerGetCurrentApp(s)
	registerGetFocusedActivity(s)

	// P0: Network & telephony (shell-based).
	registerGetWifiState(s)
	registerGetTelephonyInfo(s)

	// P0: Notifications (shell-based, StatusBarNotification parcelable incomplete).
	registerListNotifications(s)

	// P0: Battery (sysfs-based, more reliable than binder for shell UID).
	registerGetBatteryInfo(s)

	// P1: Device info.
	registerGetMemoryInfo(s)
	registerGetStorageInfo(s)
	registerGetUptime(s)
	registerGetSystemFeatures(s)

	// P1: Power.
	registerSetStayOn(s)
	registerRebootDevice(s)

	// P1: Display.
	registerSetDisplaySize(s)
	registerGetDisplayDensity(s)
	registerSetDisplayDensity(s)
	registerGetScreenOrientation(s)
	registerSetScreenOrientation(s)

	// P1: Screen recording.
	registerStartScreenRecording(s)
	registerStopScreenRecording(s)

	// P1: Input.
	registerDragDrop(s)
	registerScroll(s)

	// P1: UI automation.
	registerWaitForElement(s)
	registerScrollToElement(s)
	registerGetElementText(s)
	registerIsElementPresent(s)

	// P1: App management.
	registerClearAppData(s)
	registerGrantPermission(s)
	registerRevokePermission(s)
	registerListAppPermissions(s)
	registerIsAppInstalled(s)

	// P1: Activity.
	registerSendBroadcast(s)

	// P1: Network.
	registerGetMobileDataState(s)
	registerSetMobileDataEnabled(s)
	registerGetAirplaneMode(s)
	registerSetAirplaneMode(s)
	registerGetIPAddress(s)
	registerGetNetworkInfo(s)
	registerSetWifiEnabled(s)

	// P1: Location.
	registerSetMockLocation(s)
	registerGetLocationProviders(s)

	// P1: Sensors.
	registerListSensors(s)

	// P1: Camera & media.
	registerCapturePhoto(s)
	registerListAudioDevices(s)

	// P1: Storage & files.
	registerPushFile(s)
	registerPullFile(s)
	registerListDirectory(s)
	registerReadFile(s)

	// P1: Notifications.
	registerPostNotification(s)
	registerCancelNotification(s)
	registerExpandNotifications(s)
	registerCollapsePanels(s)

	// P1: Telephony.
	registerMakeCall(s)
	registerEndCall(s)
	registerSendSMS(s)
	registerGetCallState(s)

	// P1: Bluetooth.
	registerSetBluetoothEnabled(s)
	registerListPairedDevices(s)
	registerListConnectedBTDevices(s)

	// P1: NFC.
	registerGetNFCState(s)

	// P1: Settings.
	registerListSettings(s)
	registerGetLocale(s)
	registerGetTimezone(s)
	registerGetDateTime(s)

	// P1: Dev tools.
	registerClearLogcat(s)
	registerGetRunningServices(s)

	// P1: Window management.
	registerListRecentTasks(s)
	registerGetWindowStack(s)

	// P1: Vibrator.
	registerVibrate(s)

	// P1: Content.
	registerQueryContent(s)

	// P2: Device info.
	registerGetCPUInfo(s)

	// P2: Power.
	registerShutdownDevice(s)

	// P2: Input.
	registerPinch(s)

	// P2: App management.
	registerGetAppProcesses(s)

	// P2: Activity.
	registerStartService(s)

	// P2: Sensors.
	registerReadSensor(s)

	// P2: Battery.
	registerGetBatteryStats(s)

	// P2: Storage.
	registerWriteFile(s)
	registerDeleteFile(s)

	// P2: NFC.
	registerSetNFCEnabled(s)

	// P2: Dev tools.
	registerBugreport(s)

	// P2: Window management.
	registerMoveTaskToFront(s)
	registerRemoveTask(s)

	// P2: Vibrator.
	registerCancelVibration(s)

	// P2: Alarms.
	registerListAlarms(s)
	registerSetAlarm(s)

	// P2: Accounts.
	registerListAccounts(s)

	// P2: Content.
	registerInsertContent(s)

	// P2: Telephony.
	registerListContacts(s)
}
