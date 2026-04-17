package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed banner.txt
var banner string

var rootCmd = &cobra.Command{
	Use:     "jotter",
	Short:   "Append-only session log tool for Claude Code sessions",
	Long:    strings.TrimRight(banner, "\n") + "\n\nAppend-only session log tool for Claude Code sessions.",
	Version: versionString(),
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
