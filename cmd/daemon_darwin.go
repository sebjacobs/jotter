//go:build darwin

package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sebjacobs/jotter/internal"
)

// daemonLabel is the launchd job label and the plist basename.
const daemonLabel = "com.jotter.push"

// renderPlist builds a launchd LaunchAgent plist that runs execPath with args
// every interval seconds, logging stdout and stderr to logPath.
func renderPlist(label, execPath string, args []string, interval int, logPath string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString("<dict>\n")
	b.WriteString("\t<key>Label</key>\n\t<string>" + plistEscape(label) + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	b.WriteString("\t\t<string>" + plistEscape(execPath) + "</string>\n")
	for _, a := range args {
		b.WriteString("\t\t<string>" + plistEscape(a) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	fmt.Fprintf(&b, "\t<key>StartInterval</key>\n\t<integer>%d</integer>\n", interval)
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	b.WriteString("\t<key>StandardOutPath</key>\n\t<string>" + plistEscape(logPath) + "</string>\n")
	b.WriteString("\t<key>StandardErrorPath</key>\n\t<string>" + plistEscape(logPath) + "</string>\n")
	b.WriteString("</dict>\n")
	b.WriteString("</plist>\n")
	return b.String()
}

func plistEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// plistPath returns the LaunchAgent plist path for the current user.
func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", daemonLabel+".plist"), nil
}

// daemonLogPath returns the file launchd writes the job's stdout/stderr to.
func daemonLogPath() (string, error) {
	stateDir, err := internal.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "daemon.log"), nil
}

func installDaemon(out io.Writer, interval int) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating jotter binary: %w", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}

	logPath, err := daemonLogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	plist, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plist), 0o755); err != nil {
		return fmt.Errorf("creating LaunchAgents dir: %w", err)
	}

	body := renderPlist(daemonLabel, exe, []string{"sync", "--all"}, interval, logPath)
	if err := os.WriteFile(plist, []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	// Reload so an updated interval or binary path takes effect. Unload first
	// (ignore the error — the job may not be loaded yet), then load.
	_ = exec.Command("launchctl", "unload", "-w", plist).Run()
	if err := exec.Command("launchctl", "load", "-w", plist).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Installed background push timer (%s), running `jotter sync --all` every %ds\n",
		internal.Dim(plist), interval)
	_, _ = fmt.Fprintf(out, "Logs: %s\n", internal.Dim(logPath))
	return nil
}

func uninstallDaemon(out io.Writer) error {
	plist, err := plistPath()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(plist); os.IsNotExist(statErr) {
		_, _ = fmt.Fprintln(out, "Background push timer is not installed")
		return nil
	}

	_ = exec.Command("launchctl", "unload", "-w", plist).Run()
	if err := os.Remove(plist); err != nil {
		return fmt.Errorf("removing plist: %w", err)
	}
	_, _ = fmt.Fprintln(out, "Removed the background push timer")
	return nil
}

func statusDaemon(out io.Writer) error {
	plist, err := plistPath()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(plist); os.IsNotExist(statErr) {
		_, _ = fmt.Fprintln(out, "Background push timer: not installed (run `jotter daemon install`)")
		return nil
	}

	loaded := exec.Command("launchctl", "list", daemonLabel).Run() == nil
	state := internal.Dim("installed but not loaded")
	if loaded {
		state = internal.Bold("loaded")
	}
	_, _ = fmt.Fprintf(out, "Background push timer: %s\n", state)
	_, _ = fmt.Fprintf(out, "Plist: %s\n", internal.Dim(plist))

	logPath, err := daemonLogPath()
	if err != nil {
		return err
	}
	if tail := tailFile(logPath, 10); tail != "" {
		_, _ = fmt.Fprintf(out, "\nRecent log (%s):\n%s", internal.Dim(logPath), tail)
	}
	return nil
}

// tailFile returns up to the last n lines of the file at path (newline
// terminated), or "" if it cannot be read or is empty.
func tailFile(path string, n int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n") + "\n"
}
