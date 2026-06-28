package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The README is the CLI's hand-written documentation; the cobra command tree is
// the source of truth for what commands exist. They drift when a command ships
// without a matching README mention — so this asserts every user-facing command
// (and subcommand) appears in the README as a `jotter <command>` invocation.
//
// cobra's own generated `help` and `completion` commands are boilerplate, not
// part of the product's documented surface, so they're skipped — as is any
// command explicitly marked Hidden.
func TestREADMEDocumentsEveryCommand(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join("..", "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	doc := string(readme)

	skip := map[string]bool{"help": true, "completion": true}

	var check func(cmd *cobra.Command, path string)
	check = func(cmd *cobra.Command, path string) {
		for _, sub := range cmd.Commands() {
			if skip[sub.Name()] || sub.Hidden {
				continue
			}
			full := strings.TrimSpace(path + " " + sub.Name())
			if invocation := "jotter " + full; !strings.Contains(doc, invocation) {
				t.Errorf("README.md does not document `%s` — add it to the CLI reference or mark the command Hidden", invocation)
			}
			check(sub, full)
		}
	}
	check(rootCmd, "")
}
