//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// PackageInfoResult holds the get_package_info response.
type PackageInfoResult struct {
	PackageName      string `json:"package_name"`
	VersionCode      int32  `json:"version_code"`
	VersionName      string `json:"version_name"`
	FirstInstallTime int64  `json:"first_install_time"`
	LastUpdateTime   int64  `json:"last_update_time"`
	Error            string `json:"error,omitempty"`
}

func (ts *ToolSet) registerGetPackageInfo(s *server.MCPServer) {
	tool := mcp.NewTool("get_package_info",
		mcp.WithDescription(
			"Get information about an installed package (version, install time) "+
				"using IPackageManager.getPackageInfo().",
		),
		mcp.WithString("package",
			mcp.Required(),
			mcp.Description("Package name (e.g. 'com.android.chrome')"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleGetPackageInfo)
}

func (ts *ToolSet) handleGetPackageInfo(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetPackageInfo")
	defer func() { logger.Tracef(ctx, "/handleGetPackageInfo") }()

	pkgName, err := request.RequireString("package")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("package"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("package service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, packageManagerDescriptor, "getPackageInfo")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getPackageInfo: %v", err)), nil
	}

	// getPackageInfo(String packageName, long flags, int userId)
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(packageManagerDescriptor)
	data.WriteString16(pkgName)
	data.WriteInt64(0) // flags
	data.WriteInt32(0) // userId

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getPackageInfo: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getPackageInfo status: %v", err)), nil
	}

	// Reply is a nullable PackageInfo parcelable.
	nullInd, err := reply.ReadInt32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading null indicator: %v", err)), nil
	}

	if nullInd == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("package %q not found", pkgName)), nil
	}

	// PackageInfo is a Java Parcelable with many fields. Parse the key fields
	// manually: packageName(String), splitNames(StringArray), versionCode(int32),
	// versionCodeMajor(int32), versionName(String), baseRevisionCode(int32),
	// splitRevisionCodes(int32[]), sharedUserId(String), sharedUserLabel(int32),
	// ApplicationInfo(Parcelable null marker + opaque), firstInstallTime(int64),
	// lastUpdateTime(int64).
	// The generated PackageInfo uses MarshalParcel/UnmarshalParcel with WriteString
	// (Java writeString = length-prefixed UTF-8), not WriteString16 (UTF-16).
	// Fall back to shell for reliable parsing.
	result := parsePackageInfoFromShell(pkgName)

	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling package info: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}

// parsePackageInfoFromShell uses 'dumpsys package' to extract package info
// reliably, since the binder PackageInfo parcelable is complex.
func parsePackageInfoFromShell(pkgName string) PackageInfoResult {
	cmd := fmt.Sprintf("dumpsys package %s", shellQuote(pkgName))
	out, err := shellExec(cmd)
	if err != nil {
		return PackageInfoResult{PackageName: pkgName, Error: fmt.Sprintf("dumpsys: %v", err)}
	}

	result := PackageInfoResult{PackageName: pkgName}

	// Parse versionCode and versionName from dumpsys output lines like:
	//   versionCode=33 minSdk=30 targetSdk=33
	//   versionName=13.0
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "versionCode="):
			fmt.Sscanf(trimmed, "versionCode=%d", &result.VersionCode)
		case strings.HasPrefix(trimmed, "versionName="):
			result.VersionName = trimmed[len("versionName="):]
		}
	}

	return result
}
