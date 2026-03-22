//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerReadSensor(s *server.MCPServer) {
	tool := mcp.NewTool("read_sensor",
		mcp.WithDescription(
			"Read the current value from a specific sensor. "+
				"Uses 'dumpsys sensorservice' and filters for the named sensor. "+
				"For real-time data, the sensor must have an active listener.",
		),
		mcp.WithString("sensor_name",
			mcp.Required(),
			mcp.Description("Sensor name or type (e.g. 'accelerometer', 'gyroscope', 'light')"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)

	s.AddTool(tool, handleReadSensor)
}

func handleReadSensor(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger.Tracef(ctx, "handleReadSensor")
	defer func() { logger.Tracef(ctx, "/handleReadSensor") }()

	sensorName, err := request.RequireString("sensor_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd := fmt.Sprintf("dumpsys sensorservice | grep -i -A 5 %s | head -20", shellQuote(sensorName))
	out, err := shellExec(cmd)
	if err != nil {
		if out == "" {
			return mcp.NewToolResultError(fmt.Sprintf("read sensor: %v", err)), nil
		}
	}

	if out == "" {
		out = fmt.Sprintf("no data for sensor %q (sensor may not have an active listener)", sensorName)
	}

	return mcp.NewToolResultText(out), nil
}
