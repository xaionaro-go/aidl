package client

import (
	"context"
	"fmt"

	androidOS "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/logger"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// DeviceInfo holds aggregate device state in gomobile-safe types.
type DeviceInfo struct {
	ScreenOn      bool
	PowerSave     bool
	Brightness    float32
	ThermalStatus int32
	ServiceCount  int32
}

// GetDeviceInfo returns an aggregate snapshot of device state by querying
// the power manager, thermal service, and service manager.
func (c *BinderClient) GetDeviceInfo() (*DeviceInfo, error) {
	ctx := context.Background()
	logger.Debugf(ctx, "GetDeviceInfo")

	pm, err := androidOS.GetPowerManager(ctx, c.sm)
	if err != nil {
		return nil, fmt.Errorf("getting power manager: %w", err)
	}

	screenOn, err := pm.IsInteractive(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying IsInteractive: %w", err)
	}

	powerSave, err := pm.IsPowerSaveMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying IsPowerSaveMode: %w", err)
	}

	brightness, err := pm.GetBrightnessConstraint(ctx, brightnessConstraintTypeDefault)
	if err != nil {
		return nil, fmt.Errorf("querying brightness: %w", err)
	}

	thermalStatus, err := c.getCurrentThermalStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying thermal status: %w", err)
	}

	services, err := c.sm.ListServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	return &DeviceInfo{
		ScreenOn:      screenOn,
		PowerSave:     powerSave,
		Brightness:    brightness,
		ThermalStatus: thermalStatus,
		ServiceCount:  int32(len(services)),
	}, nil
}

// getCurrentThermalStatus queries the thermal service for the current thermal status.
func (c *BinderClient) getCurrentThermalStatus(
	ctx context.Context,
) (int32, error) {
	svc, err := c.sm.GetService(ctx, servicemanager.ThermalService)
	if err != nil {
		return 0, fmt.Errorf("getting thermal service: %w", err)
	}

	thermalProxy := androidOS.NewThermalServiceProxy(svc)
	return thermalProxy.GetCurrentThermalStatus(ctx)
}
