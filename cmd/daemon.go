package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// daemonManager adapts the platform daemon functions to setup.DaemonManager,
// letting the setup wizard offer to install the push timer without importing
// the launchd specifics.
type daemonManager struct{}

func (daemonManager) Installed() bool { return daemonInstalled() }

func (daemonManager) Install(out io.Writer) error { return installDaemon(out, defaultInterval) }

// defaultInterval is how often, in seconds, the background job runs `sync --all`.
const defaultInterval = 300

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the background push timer (launchd)",
	Long: "Install, remove, or inspect the launchd timer that periodically runs " +
		"`jotter sync --all`, pushing every registered data repo to its remote in " +
		"the background so writes never block on the network. macOS only.",
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install (or update) the background push timer",
	RunE: func(cmd *cobra.Command, _ []string) error {
		interval, _ := cmd.Flags().GetInt("interval")
		if interval < 1 {
			return fmt.Errorf("--interval must be at least 1 second")
		}
		return installDaemon(cmd.OutOrStdout(), interval)
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the background push timer",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return uninstallDaemon(cmd.OutOrStdout())
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the background push timer is installed and loaded",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return statusDaemon(cmd.OutOrStdout())
	},
}

func init() {
	daemonInstallCmd.Flags().Int("interval", defaultInterval, "Seconds between background pushes")
	daemonCmd.AddCommand(daemonInstallCmd, daemonUninstallCmd, daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)
}
