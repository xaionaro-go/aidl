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
	registerGetDeviceProperties(s)
	registerGetDeviceSpecs(s)
	registerTakeScreenshot(s)
	registerTap(s)
	registerLongPress(s)
	registerSwipe(s)
	registerInputText(s)
	registerPressKey(s)
	registerDumpUIHierarchy(s)
	registerFindUIElement(s)
	registerClickUIElement(s)
	registerGetFocusedWindow(s)
	registerInstallApp(s)
	registerUninstallApp(s)
	registerGetSetting(s)
	registerSetSetting(s)
	registerGetLogcat(s)
	registerShellExec(s)
	registerDumpService(s)
	registerOpenURL(s)
	registerStartActivity(s)

	// Clipboard (shell-based due to complex ClipData parcelable).
	registerGetClipboard(s)
	registerSetClipboard(s)

	// App management (shell-based).
	registerLaunchApp(s)
	registerGetCurrentApp(s)
	registerGetFocusedActivity(s)

	// Network & telephony (shell-based).
	registerGetWifiState(s)
	registerGetTelephonyInfo(s)

	// Notifications (shell-based, StatusBarNotification parcelable incomplete).
	registerListNotifications(s)

	// Battery (sysfs-based, more reliable than binder for shell UID).
	registerGetBatteryInfo(s)
}
