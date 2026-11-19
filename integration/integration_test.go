// Package integration runs the real gclone binary on the fixture projects in testdata/.
// Nothing here needs the network or any tool except Go and git. The tests build the binary
// once, so they take a few seconds; `go test -short ./...` skips them.
package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

// gclone builds the binary once and returns its path.
func gclone(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("integration tests are skipped with -short")
	}
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "gclone-integration")
		if err != nil {
			buildErr = err
			return
		}
		name := "gclone"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		binPath = filepath.Join(dir, name)
		out, err := exec.Command("go", "build", "-o", binPath, "..").CombinedOutput()
		if err != nil {
			buildErr = errors.New(err.Error() + "\n" + string(out))
		}
	})
	if buildErr != nil {
		t.Fatalf("could not build gclone: %v", buildErr)
	}
	return binPath
}

type run struct {
	stdout, stderr string
	code           int
}

// do runs gclone with no input on the terminal, the way a CI job would.
func do(t *testing.T, args ...string) run {
	t.Helper()
	cmd := exec.Command(gclone(t), args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.Stdin = strings.NewReader("")
	err := cmd.Run()
	r := run{stdout: out.String(), stderr: errb.String()}
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		r.code = ee.ExitCode()
	default:
		t.Fatal(err)
	}
	return r
}

type result struct {
	OK       bool     `json:"ok"`
	Detected []string `json:"detected"`
	Error    string   `json:"error"`
	Steps    []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"steps"`
}

func parse(t *testing.T, r run) result {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(r.stdout))
	var res result
	if err := dec.Decode(&res); err != nil {
		t.Fatalf("standard output is not JSON: %v\n%s\nstderr:\n%s", err, r.stdout, r.stderr)
	}
	if _, err := dec.Token(); err != io.EOF {
		t.Errorf("--json must print exactly one document, found more on standard output")
	}
	return res
}

func fixture(name string) string { return filepath.Join("..", "testdata", name) }

// Each fixture must be detected and planned, and a dry run must leave it untouched.
func TestFixturesDryRun(t *testing.T) {
	cases := []struct{ name, detected, plan string }{
		{"node-express", "Express", "Install dependencies (npm)"},
		{"go-cli", "Go ", "Download Go modules"},
		{"spring-maven", "Spring Boot", "Download Java dependencies"},
		{"python-flask", "Flask", "Install dependencies (pip)"},
		{"python-flask-factory", "Flask", "Install dependencies (pip)"},
		{"python-fastapi-factory", "FastAPI", "Install dependencies (pip)"},
		{"python-django", "Django", "Create virtual environment"},
		{"flutter-app", "Flutter", "flutter pub get"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			dir := fixture(c.name)
			if _, err := os.Stat(dir); err != nil {
				t.Skip("fixture not present")
			}
			r := do(t, "setup", dir, "--dry-run", "--json")
			if r.code != 0 {
				t.Fatalf("exit %d\n%s", r.code, r.stderr)
			}
			res := parse(t, r)
			if !res.OK {
				t.Errorf("ok is false: %s", res.Error)
			}
			if !strings.Contains(strings.Join(res.Detected, "\n"), c.detected) {
				t.Errorf("detected %q, want a line containing %q", res.Detected, c.detected)
			}
			if !strings.Contains(r.stderr, c.plan) {
				t.Errorf("the plan on standard error does not mention %q:\n%s", c.plan, r.stderr)
			}
			for _, leftover := range []string{"SETUP_GUIDE.md", ".gclone"} {
				if _, err := os.Stat(filepath.Join(dir, leftover)); err == nil {
					t.Errorf("a dry run created %s in the fixture", leftover)
				}
			}
		})
	}
}

func copyDir(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.Walk(from, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		dst := filepath.Join(to, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A real run, not a dry run: the Go fixture has no dependencies, so it needs only Go.
func TestRealRunOnGoFixture(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	dir := t.TempDir()
	copyDir(t, fixture("go-cli"), dir)
	r := do(t, "setup", dir, "--ci", "--json", "--no-services")
	if r.code != 0 {
		t.Fatalf("exit %d\n%s", r.code, r.stderr)
	}
	res := parse(t, r)
	if !res.OK || len(res.Steps) == 0 {
		t.Fatalf("%+v\n%s", res, r.stderr)
	}
	for _, s := range res.Steps {
		if s.Status != "ok" {
			t.Errorf("step %q is %s", s.Name, s.Status)
		}
	}
	for _, f := range []string{"SETUP_GUIDE.md", filepath.Join(".gclone", "state.json")} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s was not written", f)
		}
	}
	// The recorded state is readable, and nothing failed.
	st := do(t, "status", dir, "--json")
	if st.code != 0 {
		t.Errorf("status exit %d\n%s", st.code, st.stderr)
	}
	// Resuming skips what already succeeded.
	again := do(t, "setup", dir, "--ci", "--resume", "--no-services")
	if again.code != 0 || !strings.Contains(again.stderr+again.stdout, "already done") {
		t.Errorf("resume did not skip finished steps (exit %d):\n%s", again.code, again.stdout+again.stderr)
	}
}

func TestCIFailsOnAFailedStep(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	dir := t.TempDir()
	// A module that needs a dependency which cannot exist, so `go mod download` fails.
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/broken\n\ngo 1.21\n\nrequire example.invalid/nothing v1.0.0\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	r := do(t, "setup", dir, "--ci", "--json", "--no-services")
	res := parse(t, r)
	if r.code != 1 || res.OK {
		t.Fatalf("a failed step must give exit 1 and ok=false, got exit %d ok=%v", r.code, res.OK)
	}
	failed := false
	for _, s := range res.Steps {
		failed = failed || s.Status == "failed"
	}
	if !failed {
		t.Errorf("no step is marked failed: %+v", res.Steps)
	}
}

func TestStatusWithoutState(t *testing.T) {
	r := do(t, "status", t.TempDir())
	if r.code != 1 || !strings.Contains(r.stderr, "no gclone state") {
		t.Errorf("exit %d, stderr %q", r.code, r.stderr)
	}
}

func TestUsageErrors(t *testing.T) {
	if r := do(t, "--no-such-option"); r.code != 2 {
		t.Errorf("unknown option: exit %d, want 2", r.code)
	}
	r := do(t, "--version")
	if r.code != 0 || !strings.HasPrefix(r.stdout, "gclone ") {
		t.Errorf("version: exit %d %q", r.code, r.stdout)
	}
}
