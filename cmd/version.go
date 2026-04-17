package cmd

import (
	"fmt"
	"runtime"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func versionString() string {
	return fmt.Sprintf("jotter %s\ncommit: %s\nbuilt:  %s\ngo:     %s",
		version, commit, date, runtime.Version())
}
