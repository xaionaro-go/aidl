//go:build linux

package main

import (
	"github.com/AndroidGoLab/binder/cmd/bindercli/cliutil"
)

// Conn is an alias for cliutil.Conn for use within package main.
type Conn = cliutil.Conn

// OpenConn is a convenience alias for cliutil.OpenConn.
var OpenConn = cliutil.OpenConn
