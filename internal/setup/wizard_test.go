package setup

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// stubStep is a minimal Step for runner tests.
type stubStep struct {
	name       string
	state      State
	detectErr  error
	runResult  Result
	runErr     error
	detectRuns int
	runRuns    int
}

func (s *stubStep) Name() string { return s.name }
func (s *stubStep) Detect(_ *Context) (State, error) {
	s.detectRuns++
	return s.state, s.detectErr
}
func (s *stubStep) Run(_ *Context) (Result, error) {
	s.runRuns++
	return s.runResult, s.runErr
}

func newCtx(out *bytes.Buffer) *Context {
	return &Context{
		Home:     "/fake/home",
		Answers:  &Answers{},
		Prompter: nil,
		Out:      out,
	}
}

func TestRunAllStepsSucceed(t *testing.T) {
	a := &stubStep{name: "a", state: NeedsRun, runResult: Result{Status: StatusOK, Message: "did a"}}
	b := &stubStep{name: "b", state: NeedsRun, runResult: Result{Status: StatusUpdated, Message: "changed b"}}

	var out bytes.Buffer
	if err := Run(newCtx(&out), []Step{a, b}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if a.runRuns != 1 || b.runRuns != 1 {
		t.Errorf("expected both steps to run once; got a=%d b=%d", a.runRuns, b.runRuns)
	}

	got := out.String()
	for _, want := range []string{"✓ a — did a", "✎ b — changed b", "Summary:"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q; got:\n%s", want, got)
		}
	}
}

func TestRunSkipsAlreadyDone(t *testing.T) {
	a := &stubStep{name: "a", state: AlreadyDone}
	b := &stubStep{name: "b", state: NeedsRun, runResult: Result{Status: StatusOK, Message: "did b"}}

	var out bytes.Buffer
	if err := Run(newCtx(&out), []Step{a, b}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if a.runRuns != 0 {
		t.Errorf("expected step a not to run (AlreadyDone); got %d runs", a.runRuns)
	}
	if b.runRuns != 1 {
		t.Errorf("expected step b to run; got %d runs", b.runRuns)
	}

	got := out.String()
	if !strings.Contains(got, "↷ a — already done") {
		t.Errorf("output missing skip message for a; got:\n%s", got)
	}
}

func TestRunShortCircuitsOnFailure(t *testing.T) {
	a := &stubStep{name: "a", state: NeedsRun, runErr: errors.New("boom")}
	b := &stubStep{name: "b", state: NeedsRun, runResult: Result{Status: StatusOK, Message: "did b"}}

	var out bytes.Buffer
	err := Run(newCtx(&out), []Step{a, b})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "step \"a\" failed") {
		t.Errorf("error message missing step name wrap; got: %v", err)
	}
	if b.detectRuns != 0 || b.runRuns != 0 {
		t.Errorf("expected step b not to run after a failed; got detect=%d run=%d", b.detectRuns, b.runRuns)
	}

	got := out.String()
	if !strings.Contains(got, "✗ a — boom") {
		t.Errorf("output missing failure line for a; got:\n%s", got)
	}
	if !strings.Contains(got, "Summary:") {
		t.Errorf("summary should still print after failure; got:\n%s", got)
	}
}

func TestRunSkipsNotApplicable(t *testing.T) {
	a := &stubStep{name: "claude-check", state: NotApplicable}

	var out bytes.Buffer
	if err := Run(newCtx(&out), []Step{a}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if a.runRuns != 0 {
		t.Errorf("expected step not to run (NotApplicable); got %d runs", a.runRuns)
	}
	if !strings.Contains(out.String(), "↷ claude-check — not applicable") {
		t.Errorf("output missing not-applicable message; got:\n%s", out.String())
	}
}
