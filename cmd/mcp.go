package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run an MCP server over stdio",
	Long: "Serve jotter's write, tail, search and ls commands as Model Context\n" +
		"Protocol tools over stdin/stdout, so an MCP client such as Claude Code can\n" +
		"read and write the log without shelling out.\n\n" +
		"Register it with:\n\n" +
		"  claude mcp add --scope user jotter -- jotter mcp    # Claude Code\n" +
		"  codex mcp add jotter -- jotter mcp                  # Codex CLI, ChatGPT desktop\n\n" +
		"See the README for Claude Desktop.\n\n" +
		"Each tool call runs the matching jotter command as a subprocess, so project\n" +
		"and branch default from the server's working directory exactly as they do\n" +
		"on the command line.",
	Args: cobra.NoArgs,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

type scopeArgs struct {
	Project string `json:"project,omitempty" jsonschema:"project name (default: the current git project)"`
	Branch  string `json:"branch,omitempty" jsonschema:"branch name (default: the current git branch)"`
}

type windowArgs struct {
	Since string `json:"since,omitempty" jsonschema:"only entries from this date/time, YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive"`
	Until string `json:"until,omitempty" jsonschema:"only entries up to this date/time, YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, inclusive"`
}

type writeArgs struct {
	scopeArgs
	Type    string `json:"type" jsonschema:"entry type: start, checkpoint, note, break, stop or handover"`
	Content string `json:"content" jsonschema:"entry content"`
	Next    string `json:"next,omitempty" jsonschema:"what to do next"`
}

type tailArgs struct {
	scopeArgs
	Limit int `json:"limit,omitempty" jsonschema:"number of entries to return (default 1)"`
}

type searchArgs struct {
	scopeArgs
	windowArgs
	Term string `json:"term,omitempty" jsonschema:"case-insensitive text to match in entry content"`
	Type string `json:"type,omitempty" jsonschema:"only entries of this type"`
	All  bool   `json:"all,omitempty" jsonschema:"search every project and branch, ignoring the current repo"`
}

type lsArgs struct {
	scopeArgs
	windowArgs
	All bool `json:"all,omitempty" jsonschema:"list every project, ignoring the current repo"`
}

func (s scopeArgs) flags() []string {
	return appendFlag(appendFlag(nil, "--project", s.Project), "--branch", s.Branch)
}

func (w windowArgs) flags() []string {
	return appendFlag(appendFlag(nil, "--since", w.Since), "--until", w.Until)
}

func appendFlag(args []string, name, value string) []string {
	if value == "" {
		return args
	}
	return append(args, name, value)
}

func boolFlag(name string, on bool) []string {
	if !on {
		return nil
	}
	return []string{name}
}

func newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "jotter", Version: version}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "jotter_write",
		Description: "Append an entry to the jotter session log for a project branch and commit it.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in writeArgs) (*mcp.CallToolResult, any, error) {
		args := []string{"write", "--type", in.Type, "--content", in.Content}
		args = appendFlag(args, "--next", in.Next)
		return runJotterTool(ctx, append(args, in.flags()...))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "jotter_tail",
		Description: "Show the most recent jotter entries for a project branch, newest last.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tailArgs) (*mcp.CallToolResult, any, error) {
		args := []string{"tail"}
		if in.Limit > 0 {
			args = append(args, "--limit", strconv.Itoa(in.Limit))
		}
		return runJotterTool(ctx, append(args, in.flags()...))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "jotter_search",
		Description: "Search jotter entries by text, type and date. Defaults to the current project and branch.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchArgs) (*mcp.CallToolResult, any, error) {
		args := []string{"search"}
		args = appendFlag(args, "--type", in.Type)
		args = append(args, boolFlag("--all", in.All)...)
		args = append(args, in.scopeArgs.flags()...)
		args = append(args, in.windowArgs.flags()...)
		if in.Term != "" {
			args = append(args, "--", in.Term)
		}
		return runJotterTool(ctx, args)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "jotter_ls",
		Description: "List logged projects, the branches of a project, or the entries of a branch.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in lsArgs) (*mcp.CallToolResult, any, error) {
		args := []string{"ls"}
		args = append(args, boolFlag("--all", in.All)...)
		args = append(args, in.scopeArgs.flags()...)
		args = append(args, in.windowArgs.flags()...)
		return runJotterTool(ctx, args)
	})

	return server
}

func runMCP(cmd *cobra.Command, _ []string) error {
	return newMCPServer().Run(cmd.Context(), &mcp.StdioTransport{})
}

func runJotterTool(ctx context.Context, args []string) (*mcp.CallToolResult, any, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("locating jotter binary: %w", err)
	}

	var stdout, stderr bytes.Buffer
	c := exec.CommandContext(ctx, self, args...)
	c.Stdout = &stdout
	c.Stderr = &stderr
	c.Env = append(os.Environ(), "NO_COLOR=1")

	runErr := c.Run()
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())

	var exitErr *exec.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		return nil, nil, fmt.Errorf("running jotter %s: %w", args[0], runErr)
	}
	if runErr != nil {
		errOut, _, _ = strings.Cut(errOut, "\nUsage:")
		msg := strings.TrimSpace(strings.Join([]string{errOut, out}, "\n"))
		if msg == "" {
			msg = runErr.Error()
		}
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, nil, nil
	}

	text := out
	if errOut != "" {
		text = strings.TrimSpace(text + "\n\n" + errOut)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
}
