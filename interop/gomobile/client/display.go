package client

import (
	"context"
	"fmt"

	"github.com/AndroidGoLab/binder/android/app"
	androidOS "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/logger"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	// brightnessConstraintTypeDefault corresponds to
	// PowerManager.BRIGHTNESS_CONSTRAINT_TYPE_DEFAULT (2).
	brightnessConstraintTypeDefault = int32(2)

	// nightModeYes corresponds to UiModeManager.MODE_NIGHT_YES (2).
	nightModeYes = int32(2)
)

// DisplayInfo holds display state in gomobile-safe types.
type DisplayInfo struct {
	Brightness float32
	NightMode  bool
}

// GetDisplayInfo returns the current display brightness (default constraint)
// and night mode state.
func (c *BinderClient) GetDisplayInfo() (*DisplayInfo, error) {
	ctx := context.Background()
	logger.Debugf(ctx, "GetDisplayInfo")

	pm, err := androidOS.GetPowerManager(ctx, c.sm)
	if err != nil {
		return nil, fmt.Errorf("getting power manager: %w", err)
	}

	brightness, err := pm.GetBrightnessConstraint(ctx, brightnessConstraintTypeDefault)
	if err != nil {
		return nil, fmt.Errorf("querying brightness: %w", err)
	}

	nightMode, err := c.getNightMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying night mode: %w", err)
	}

	return &DisplayInfo{
		Brightness: brightness,
		NightMode:  nightMode,
	}, nil
}

// getNightMode queries the UiModeManager to determine if night mode is active.
func (c *BinderClient) getNightMode(
	ctx context.Context,
) (bool, error) {
	svc, err := c.sm.GetService(ctx, servicemanager.UiModeService)
	if err != nil {
		return false, fmt.Errorf("getting uimode service: %w", err)
	}

	uiMgr := app.NewUiModeManagerProxy(svc)
	mode, err := uiMgr.GetNightMode(ctx)
	if err != nil {
		return false, fmt.Errorf("querying night mode: %w", err)
	}

	return mode == nightModeYes, nil
}
