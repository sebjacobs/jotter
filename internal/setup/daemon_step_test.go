//go:build darwin

package setup

import (
	"io"
	"testing"
)

type stubDaemon struct {
	installed     bool
	installCalled bool
	installErr    error
}

func (s *stubDaemon) Installed() bool { return s.installed }

func (s *stubDaemon) Install(io.Writer) error {
	s.installCalled = true
	return s.installErr
}

func daemonCtx(t *testing.T, daemon DaemonManager, remote string, confirm bool) *Context {
	ctx := newStepCtx(t, &stubPrompter{confirms: []bool{confirm}})
	ctx.Daemon = daemon
	ctx.Answers.RemoteURL = remote
	ctx.Out = io.Discard
	return ctx
}

func TestDaemonStep_NotApplicableWithoutManager(t *testing.T) {
	ctx := daemonCtx(t, nil, "git@example.com:me/logs.git", true)
	state, err := daemonStep{}.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != NotApplicable {
		t.Errorf("state = %v, want NotApplicable", state)
	}
}

func TestDaemonStep_NotApplicableWithoutRemote(t *testing.T) {
	ctx := daemonCtx(t, &stubDaemon{}, "", true)
	state, err := daemonStep{}.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != NotApplicable {
		t.Errorf("state = %v, want NotApplicable", state)
	}
}

func TestDaemonStep_AlreadyDoneWhenInstalled(t *testing.T) {
	ctx := daemonCtx(t, &stubDaemon{installed: true}, "git@example.com:me/logs.git", true)
	state, err := daemonStep{}.Detect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state != AlreadyDone {
		t.Errorf("state = %v, want AlreadyDone", state)
	}
}

func TestDaemonStep_InstallsOnConfirm(t *testing.T) {
	daemon := &stubDaemon{}
	ctx := daemonCtx(t, daemon, "git@example.com:me/logs.git", true)

	if state, _ := (daemonStep{}).Detect(ctx); state != NeedsRun {
		t.Fatalf("expected NeedsRun, got %v", state)
	}
	result, err := daemonStep{}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !daemon.installCalled {
		t.Error("Install was not called")
	}
	if result.Status != StatusUpdated {
		t.Errorf("status = %v, want StatusUpdated", result.Status)
	}
}

func TestDaemonStep_SkipsOnDecline(t *testing.T) {
	daemon := &stubDaemon{}
	ctx := daemonCtx(t, daemon, "git@example.com:me/logs.git", false)

	result, err := daemonStep{}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if daemon.installCalled {
		t.Error("Install should not be called when the user declines")
	}
	if result.Status != StatusSkipped {
		t.Errorf("status = %v, want StatusSkipped", result.Status)
	}
}
