package client

import (
	"context"
	"fmt"

	androidOS "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/logger"
)

// PowerStatus holds power state in gomobile-safe types.
type PowerStatus struct {
	IsInteractive       bool
	IsPowerSaveMode     bool
	IsBatterySaverSupported bool
}

// GetPowerStatus returns the current power state by querying the PowerManager service.
func (c *BinderClient) GetPowerStatus() (*PowerStatus, error) {
	ctx := context.Background()
	logger.Debugf(ctx, "GetPowerStatus")

	pm, err := androidOS.GetPowerManager(ctx, c.sm)
	if err != nil {
		return nil, fmt.Errorf("getting power manager: %w", err)
	}

	interactive, err := pm.IsInteractive(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying IsInteractive: %w", err)
	}

	powerSave, err := pm.IsPowerSaveMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying IsPowerSaveMode: %w", err)
	}

	batterySaver, err := pm.IsBatterySaverSupported(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying IsBatterySaverSupported: %w", err)
	}

	return &PowerStatus{
		IsInteractive:       interactive,
		IsPowerSaveMode:     powerSave,
		IsBatterySaverSupported: batterySaver,
	}, nil
}
