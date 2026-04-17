package cmd

import (
	"fmt"

	"github.com/sebjacobs/jotter/internal"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the resolved config for the current directory",
	RunE:  runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	path, cfg, err := internal.ResolveConfig()
	if err != nil {
		return err
	}
	fmt.Printf("%s %s\n", internal.Dim("config:"), path)
	fmt.Printf("%s %s\n", internal.Dim("data_dir:"), cfg.DataDir)
	return nil
}
