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

// TelephonyInfoResult holds the get_telephony_info response.
type TelephonyInfoResult struct {
	Operator    string `json:"operator,omitempty"`
	DataState   string `json:"data_state,omitempty"`
	SignalLevel string `json:"signal_level,omitempty"`
	PhoneType   string `json:"phone_type,omitempty"`
	Error       string `json:"error,omitempty"`
}

func registerGetTelephonyInfo(s *server.MCPServer) {
	tool := mcp.NewTool("get_telephony_info",
		mcp.WithDescription(
			"Get basic telephony info (operator, data state, signal) from "+
				"the device using dumpsys and getprop.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleGetTelephonyInfo)
}

func handleGetTelephonyInfo(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetTelephonyInfo")
	defer func() { logger.Tracef(ctx, "/handleGetTelephonyInfo") }()

	result := TelephonyInfoResult{}

	operator, _ := shellExec("getprop gsm.operator.alpha")
	result.Operator = strings.TrimSpace(operator)

	// Parse data state from dumpsys telephony.registry.
	out, _ := shellExec("dumpsys telephony.registry | grep -E 'mDataConnectionState|mSignalStrength' | head -2")
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, "mDataConnectionState"):
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				result.DataState = strings.TrimSpace(parts[1])
			}
		case strings.Contains(line, "mSignalStrength"):
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				result.SignalLevel = strings.TrimSpace(parts[1])
			}
		}
	}

	phoneType, _ := shellExec("getprop gsm.current.phone-type")
	result.PhoneType = strings.TrimSpace(phoneType)

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling telephony info: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}
