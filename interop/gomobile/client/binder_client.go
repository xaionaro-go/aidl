package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/logger"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	// defaultMapSize is the mmap size for the binder driver (128 KB).
	defaultMapSize = 128 * 1024
)

// BinderClient provides Java-friendly access to Android binder services.
// All exported methods use gomobile-safe types only.
type BinderClient struct {
	driver    *kernelbinder.Driver
	transport *versionaware.Transport
	sm        *servicemanager.ServiceManager
}

// NewBinderClient opens the binder driver, creates a version-aware
// transport, and initializes the service manager.
func NewBinderClient() (*BinderClient, error) {
	ctx := context.Background()
	logger.Debugf(ctx, "NewBinderClient")

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(defaultMapSize))
	if err != nil {
		return nil, fmt.Errorf("opening binder driver: %w", err)
	}

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		_ = drv.Close(ctx)
		return nil, fmt.Errorf("creating transport: %w", err)
	}

	sm := servicemanager.New(transport)

	return &BinderClient{
		driver:    drv,
		transport: transport,
		sm:        sm,
	}, nil
}

// Close releases the binder driver and all associated resources.
func (c *BinderClient) Close() error {
	ctx := context.Background()
	logger.Debugf(ctx, "BinderClient.Close")
	return c.driver.Close(ctx)
}

// ListServicesJSON returns all registered binder service names as a JSON array of strings.
func (c *BinderClient) ListServicesJSON() (string, error) {
	ctx := context.Background()
	logger.Debugf(ctx, "ListServicesJSON")

	services, err := c.sm.ListServices(ctx)
	if err != nil {
		return "", fmt.Errorf("listing services: %w", err)
	}

	// Convert to plain string slice for JSON serialization.
	names := make([]string, len(services))
	for i, s := range services {
		names[i] = string(s)
	}

	data, err := json.Marshal(names)
	if err != nil {
		return "", fmt.Errorf("marshaling services: %w", err)
	}

	return string(data), nil
}

// CheckServiceExists checks whether a service is registered in the service manager.
func (c *BinderClient) CheckServiceExists(serviceName string) (bool, error) {
	ctx := context.Background()

	svc, err := c.sm.CheckService(ctx, servicemanager.ServiceName(serviceName))
	if err != nil {
		return false, fmt.Errorf("checking service %q: %w", serviceName, err)
	}

	return svc != nil, nil
}
