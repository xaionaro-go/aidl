//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const bluetoothManagerDescriptor = "android.bluetooth.IBluetoothManager"

// BluetoothState maps the int32 returned by IBluetoothManager.getState().
type BluetoothState int32

const (
	BluetoothStateOff          BluetoothState = 10
	BluetoothStateTurningOn    BluetoothState = 11
	BluetoothStateOn           BluetoothState = 12
	BluetoothStateTurningOff   BluetoothState = 13
	BluetoothStateBLETurningOn BluetoothState = 14
	BluetoothStateBLEOn        BluetoothState = 15
	BluetoothStateBLETurnOff   BluetoothState = 16
)

// BluetoothStateResult holds the get_bluetooth_state response.
type BluetoothStateResult struct {
	StateCode int32  `json:"state_code"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
}

func (ts *ToolSet) registerGetBluetoothState(s *server.MCPServer) {
	tool := mcp.NewTool("get_bluetooth_state",
		mcp.WithDescription(
			"Get the Bluetooth adapter state using IBluetoothManager.getState(). "+
				"Returns state_code and human-readable state (off/turning_on/on/turning_off).",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, ts.handleGetBluetoothState)
}

func (ts *ToolSet) handleGetBluetoothState(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleGetBluetoothState")
	defer func() { logger.Tracef(ctx, "/handleGetBluetoothState") }()

	svc, err := ts.sm.CheckService(ctx, servicemanager.ServiceName("bluetooth_manager"))
	if err != nil || svc == nil {
		return mcp.NewToolResultError("bluetooth_manager service unavailable"), nil
	}

	code, err := svc.ResolveCode(ctx, bluetoothManagerDescriptor, "getState")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolving getState: %v", err)), nil
	}

	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(bluetoothManagerDescriptor)

	reply, err := svc.Transact(ctx, code, 0, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getState: %v", err)), nil
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getState status: %v", err)), nil
	}

	stateCode, err := reply.ReadInt32()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading state: %v", err)), nil
	}

	result := BluetoothStateResult{
		StateCode: stateCode,
		State:     bluetoothStateString(BluetoothState(stateCode)),
	}

	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling bluetooth state: %w", err)
	}

	return mcp.NewToolResultText(string(out)), nil
}

func bluetoothStateString(state BluetoothState) string {
	switch state {
	case BluetoothStateOff:
		return "off"
	case BluetoothStateTurningOn:
		return "turning_on"
	case BluetoothStateOn:
		return "on"
	case BluetoothStateTurningOff:
		return "turning_off"
	case BluetoothStateBLETurningOn:
		return "ble_turning_on"
	case BluetoothStateBLEOn:
		return "ble_on"
	case BluetoothStateBLETurnOff:
		return "ble_turning_off"
	default:
		return fmt.Sprintf("unknown(%d)", state)
	}
}
