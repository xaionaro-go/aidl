package main

import (
	"github.com/AndroidGoLab/binder/cmd/bindercli/cliutil"
)

// Formatter is an alias for cliutil.Formatter for use within package main.
type Formatter = cliutil.Formatter

// NewFormatter is a convenience alias for cliutil.NewFormatter.
var NewFormatter = cliutil.NewFormatter

// resolveMode wraps cliutil.ResolveMode for backward compatibility
// within package main (used by tests).
func resolveMode(
	mode string,
	isTTY bool,
) string {
	return cliutil.ResolveMode(mode, isTTY)
}
