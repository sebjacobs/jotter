package cmd_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func startMCP(t *testing.T, dataDir string) *mcp.ClientSession {
	t.Helper()
	workdir := t.TempDir()
	body := fmt.Sprintf("data_dir = %q\n", dataDir)
	if err := os.WriteFile(filepath.Join(workdir, ".jotter"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	server := exec.Command(binaryPath, "mcp")
	server.Dir = workdir
	server.Env = append(os.Environ(), "HOME="+t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: server}, nil)
	if err != nil {
		t.Fatalf("connecting to jotter mcp: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callTool(t *testing.T, s *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("calling %s: %v", name, err)
	}
	var text []string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text = append(text, tc.Text)
		}
	}
	return strings.Join(text, "\n"), res.IsError
}

func TestMCP_ListsTools(t *testing.T) {
	s := startMCP(t, initDataDir(t))
	res, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	want := []string{"jotter_ls", "jotter_search", "jotter_tail", "jotter_write"}
	if !slices.Equal(names, want) {
		t.Errorf("tools = %v, want %v", names, want)
	}
}

func TestMCP_WriteThenTail(t *testing.T) {
	dir := initDataDir(t)
	s := startMCP(t, dir)

	out, isErr := callTool(t, s, "jotter_write", map[string]any{
		"project": "proj", "branch": "main", "type": "note", "content": "Via MCP", "next": "Keep going",
	})
	if isErr || !strings.Contains(out, "Wrote note entry to logs/proj/main.jsonl") {
		t.Fatalf("write: isError=%v out=%q", isErr, out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("output carries ANSI colour codes: %q", out)
	}
	if log := git(t, dir, "log", "--oneline"); !strings.Contains(log, "session: proj/main note") {
		t.Errorf("entry not committed: %s", log)
	}

	out, isErr = callTool(t, s, "jotter_tail", map[string]any{"project": "proj", "branch": "main"})
	if isErr || !strings.Contains(out, "Via MCP") || !strings.Contains(out, "Keep going") {
		t.Errorf("tail: isError=%v out=%q", isErr, out)
	}
}

func TestMCP_SearchAndLs(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main", "--type", "note", "--content", "needle here")
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main", "--type", "note", "--content", "haystack")
	s := startMCP(t, dir)

	out, isErr := callTool(t, s, "jotter_search", map[string]any{"term": "needle", "all": true})
	if isErr || !strings.Contains(out, "needle here") || strings.Contains(out, "haystack") {
		t.Errorf("search: isError=%v out=%q", isErr, out)
	}

	out, isErr = callTool(t, s, "jotter_ls", map[string]any{"all": true})
	if isErr || !strings.Contains(out, "proj") {
		t.Errorf("ls: isError=%v out=%q", isErr, out)
	}
}

func TestMCP_CommandFailureIsToolErrorWithoutUsage(t *testing.T) {
	s := startMCP(t, initDataDir(t))

	out, isErr := callTool(t, s, "jotter_write", map[string]any{
		"project": "proj", "branch": "main", "type": "bogus", "content": "x",
	})
	if !isErr || !strings.Contains(out, `invalid entry type "bogus"`) || strings.Contains(out, "Usage:") {
		t.Errorf("isError=%v out=%q", isErr, out)
	}

	out, isErr = callTool(t, s, "jotter_tail", map[string]any{"project": "proj", "branch": "missing"})
	if !isErr || !strings.Contains(out, "No log file for proj/missing") {
		t.Errorf("isError=%v out=%q", isErr, out)
	}

	if _, isErr = callTool(t, s, "jotter_ls", map[string]any{"all": true}); !isErr {
		t.Error("expected ls on an empty data dir to be a tool error")
	}
}
