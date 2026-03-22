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
)

// BatteryInfoResult holds the get_battery_info response.
type BatteryInfoResult struct {
	Level       string `json:"level"`
	Status      string `json:"status"`
	Temperature string `json:"temperature,omitempty"`
	Voltage     string `json:"voltage,omitempty"`
	Technology  string `json:"technology,omitempty"`
	Health      string `json:"health,omitempty"`
	Error       string `json:"error,omitempty"`
}

func registerGetBatteryInfo(s *server.MCPServer) {
	tool := mcp.NewTool("get_battery_info",
		mcp.WithDescription(
			"Get battery information (level, status, temperature, voltage) "+
				"from /sys/class/power_supply/ sysfs entries. More reliable "+
				"than binder for shell UID.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetBatteryInfo)
}

func handleGetBatteryInfo(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetBatteryInfo")
	defer func() { logger.Tracef(ctx, "/handleGetBatteryInfo") }()

	result := BatteryInfoResult{}

	// Try dumpsys battery first (most reliable on Android).
	out, err := shellExec("dumpsys battery")
	if err == nil {
		result = parseBatteryDumpsys(out)
	} else {
		// Fall back to sysfs.
		result = readBatterySysfs()
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling battery info: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

func parseBatteryDumpsys(output string) BatteryInfoResult {
	result := BatteryInfoResult{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "level":
			result.Level = val
		case "status":
			result.Status = batteryStatusName(val)
		case "temperature":
			result.Temperature = val
		case "voltage":
			result.Voltage = val
		case "technology":
			result.Technology = val
		case "health":
			result.Health = batteryHealthName(val)
		}
	}
	return result
}

func readBatterySysfs() BatteryInfoResult {
	result := BatteryInfoResult{}
	readSysfs := func(path string) string {
		out, err := shellExec("cat " + path + " 2>/dev/null")
		if err != nil {
			return ""
		}
		return strings.TrimSpace(out)
	}

	result.Level = readSysfs("/sys/class/power_supply/battery/capacity")
	result.Status = readSysfs("/sys/class/power_supply/battery/status")
	result.Temperature = readSysfs("/sys/class/power_supply/battery/temp")
	result.Voltage = readSysfs("/sys/class/power_supply/battery/voltage_now")
	result.Technology = readSysfs("/sys/class/power_supply/battery/technology")
	result.Health = readSysfs("/sys/class/power_supply/battery/health")

	return result
}

func batteryStatusName(code string) string {
	switch code {
	case "1":
		return "unknown"
	case "2":
		return "charging"
	case "3":
		return "discharging"
	case "4":
		return "not_charging"
	case "5":
		return "full"
	default:
		return code
	}
}

func batteryHealthName(code string) string {
	switch code {
	case "1":
		return "unknown"
	case "2":
		return "good"
	case "3":
		return "overheat"
	case "4":
		return "dead"
	case "5":
		return "over_voltage"
	case "6":
		return "failure"
	case "7":
		return "cold"
	default:
		return code
	}
}
