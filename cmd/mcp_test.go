package cmd_test

import (
	"context"
	"encoding/json"
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

func callToolStructured(t *testing.T, s *mcp.ClientSession, name string, args map[string]any) map[string][]map[string]any {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("calling %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("%s returned a tool error: %v", name, res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string][]map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s structured content is not an object of arrays: %v\n%s", name, err, raw)
	}
	return out
}

func TestMCP_ReadToolsDeclareOutputSchemas(t *testing.T) {
	s := startMCP(t, initDataDir(t))
	res, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "jotter_write" {
			continue
		}
		if tool.OutputSchema == nil {
			t.Errorf("%s has no output schema", tool.Name)
		}
	}
}

func TestMCP_TailReturnsStructuredEntries(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main", "--type", "stop", "--content", "Done", "--next", "Ship")
	s := startMCP(t, dir)

	got := callToolStructured(t, s, "jotter_tail", map[string]any{"project": "proj", "branch": "main"})
	entries := got["entries"]
	if len(entries) != 1 {
		t.Fatalf("entries = %v", got)
	}
	e := entries[0]
	if e["type"] != "stop" || e["content"] != "Done" || e["next"] != "Ship" || e["project"] != "proj" || e["branch"] != "main" {
		t.Errorf("entry = %v", e)
	}
	if id, _ := e["id"].(string); len(id) != 7 {
		t.Errorf("id = %v", e["id"])
	}
}

func TestMCP_SearchReturnsStructuredEntries(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main", "--type", "note", "--content", "needle here")
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main", "--type", "note", "--content", "haystack")
	s := startMCP(t, dir)

	got := callToolStructured(t, s, "jotter_search", map[string]any{"term": "needle", "all": true})
	if len(got["entries"]) != 1 || got["entries"][0]["content"] != "needle here" {
		t.Errorf("search = %v", got)
	}
}

func TestMCP_LsReturnsStructuredRowsForEachScope(t *testing.T) {
	dir := initDataDir(t)
	runJotter(t, dir, "write", "--project", "proj", "--branch", "main", "--type", "note", "--content", "First line\nmore")
	s := startMCP(t, dir)

	projects := callToolStructured(t, s, "jotter_ls", map[string]any{"all": true})["projects"]
	if len(projects) != 1 || projects[0]["name"] != "proj" {
		t.Errorf("projects = %v", projects)
	}

	branches := callToolStructured(t, s, "jotter_ls", map[string]any{"project": "proj"})["branches"]
	if len(branches) != 1 || branches[0]["name"] != "main" || branches[0]["entries"] != float64(1) {
		t.Errorf("branches = %v", branches)
	}

	entries := callToolStructured(t, s, "jotter_ls", map[string]any{"project": "proj", "branch": "main"})["entries"]
	if len(entries) != 1 || entries[0]["title"] != "First line" || entries[0]["type"] != "note" {
		t.Errorf("entries = %v", entries)
	}
}
