package cmd

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed banner.txt
var banner string

// skillsFS holds the embedded skills tree, passed in from main.
var skillsFS embed.FS

var rootCmd = &cobra.Command{
	Use:   "jotter",
	Short: "Append-only, git-backed log for Claude Code sessions",
	Long: strings.TrimRight(banner, "\n") + "\n\n" +
		"Jot it, keep it, find it later.\n\n" +
		"An append-only log of your Claude Code sessions, stored as JSONL in a git repo " +
		"outside your project tree. Entries are human-curated checkpoints — start, note, " +
		"checkpoint, break, finish — committed one per write and pushed on finish. Use " +
		"`tail` to replay a branch, `search` to grep across projects, and `ls` to browse " +
		"what's been logged.",
	Version: versionString(),
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "print version information and exit")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Println(versionString())
			os.Exit(0)
		}
	}
}

func Execute(fs embed.FS) {
	skillsFS = fs
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
