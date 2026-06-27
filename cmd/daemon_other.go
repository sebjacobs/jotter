//go:build !darwin

package cmd

import (
	"fmt"
	"io"
)

var errUnsupported = fmt.Errorf("the jotter daemon is only supported on macOS (launchd); on other platforms schedule `jotter sync --all` with cron or a systemd timer")

func installDaemon(_ io.Writer, _ int) error { return errUnsupported }

func uninstallDaemon(_ io.Writer) error { return errUnsupported }

func statusDaemon(_ io.Writer) error { return errUnsupported }
