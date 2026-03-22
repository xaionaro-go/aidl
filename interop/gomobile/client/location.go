package client

import (
	"context"
	"fmt"

	"github.com/AndroidGoLab/binder/android/location"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/logger"
)

// LocationResult holds GPS location data in gomobile-safe types.
// All fields are exported primitives so gomobile can bridge them to Java.
type LocationResult struct {
	Provider string
	Lat      float64
	Lon      float64
	Alt      float64
	Speed    float32
	Bearing  float32
	Accuracy float32
	TimeMs   int64
}

// GetLastLocation returns the last known location for the given provider
// (e.g. "gps", "network", "fused"). Returns nil with no error if no
// location is cached for that provider.
func (c *BinderClient) GetLastLocation(
	provider string,
) (*LocationResult, error) {
	ctx := context.Background()
	logger.Debugf(ctx, "GetLastLocation(%q)", provider)

	locMgr, err := location.GetLocationManager(ctx, c.sm)
	if err != nil {
		return nil, fmt.Errorf("getting location manager: %w", err)
	}

	loc, err := locMgr.GetLastLocation(
		ctx,
		provider,
		location.LastLocationRequest{},
		binder.DefaultCallerIdentity().PackageName,
	)
	if err != nil {
		return nil, fmt.Errorf("getting last location: %w", err)
	}

	// A zero-value Location (no provider, zero lat/lon, zero time) indicates
	// the service returned null (no cached location).
	if loc.Provider == "" && loc.TimeMs == 0 {
		return nil, nil
	}

	return &LocationResult{
		Provider: loc.Provider,
		Lat:      loc.LatitudeDegrees,
		Lon:      loc.LongitudeDegrees,
		Alt:      loc.AltitudeMeters,
		Speed:    loc.SpeedMetersPerSecond,
		Bearing:  loc.BearingDegrees,
		Accuracy: loc.HorizontalAccuracyMeters,
		TimeMs:   loc.TimeMs,
	}, nil
}
