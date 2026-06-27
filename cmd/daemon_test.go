//go:build darwin

package cmd

import (
	"strings"
	"testing"
)

func TestDaemonLabel_DefaultsToNeutralPrefix(t *testing.T) {
	t.Setenv("LAUNCHD_PREFIX", "")
	if got := daemonLabel(); got != "com.jotter" {
		t.Errorf("daemonLabel() = %q, want com.jotter", got)
	}
}

func TestDaemonLabel_HonoursLaunchdPrefix(t *testing.T) {
	t.Setenv("LAUNCHD_PREFIX", "com.example")
	if got := daemonLabel(); got != "com.example.jotter" {
		t.Errorf("daemonLabel() = %q, want com.example.jotter", got)
	}
}

func TestDaemonLabel_TrimsTrailingDotAndSpace(t *testing.T) {
	t.Setenv("LAUNCHD_PREFIX", " com.example. ")
	if got := daemonLabel(); got != "com.example.jotter" {
		t.Errorf("daemonLabel() = %q, want com.example.jotter", got)
	}
}

func TestRenderPlist_IncludesProgramArgumentsAndInterval(t *testing.T) {
	got := renderPlist("com.jotter.push", "/usr/local/bin/jotter", []string{"sync", "--all"}, 300, "/Users/x/.jotter.d/daemon.log")

	for _, want := range []string{
		"<string>com.jotter.push</string>",
		"<string>/usr/local/bin/jotter</string>",
		"<string>sync</string>",
		"<string>--all</string>",
		"<integer>300</integer>",
		"<string>/Users/x/.jotter.d/daemon.log</string>",
		"<key>RunAtLoad</key>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("plist missing %q:\n%s", want, got)
		}
	}

	if !strings.HasPrefix(got, `<?xml version="1.0"`) {
		t.Errorf("plist missing XML prolog:\n%s", got)
	}
}

func TestRenderPlist_EscapesXMLSpecials(t *testing.T) {
	got := renderPlist("com.jotter.push", "/opt/a&b/jotter", []string{"sync"}, 60, "/tmp/log")
	if !strings.Contains(got, "/opt/a&amp;b/jotter") {
		t.Errorf("ampersand not escaped:\n%s", got)
	}
	if strings.Contains(got, "/opt/a&b/jotter") {
		t.Errorf("raw ampersand leaked into plist:\n%s", got)
	}
}
