package python

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rauni5/gclone/internal/project"
)

// withPath sets PATH to entries for the duration of the test and restores the
// original value when the test finishes, even on failure.
func withPath(t *testing.T, entries ...string) {
	t.Helper()
	orig := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", orig) })
	os.Setenv("PATH", strings.Join(entries, string(os.PathListSeparator)))
}

// mkTool writes a fake executable named name into dir and returns dir. The fake
// exits 0 and does nothing, so exec.LookPath finds it without running project
// code. On Windows the file gets a .bat suffix so LookPath accepts it.
func mkTool(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		p += ".bat"
	}
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

//  VersionHint: each branch

// TestIssue29_VersionHint_NoTools verifies the fallback branch: when neither uv
// nor pyenv is on PATH, VersionHint points at python.org and does not name a
// tool the user does not have.
func TestIssue29_VersionHint_NoTools(t *testing.T) {
	withPath(t, t.TempDir()) // empty PATH
	got := VersionHint("3.12")
	if !strings.Contains(got, "python.org") {
		t.Fatalf("want python.org fallback, got %q", got)
	}
	if strings.Contains(got, "uv") || strings.Contains(got, "pyenv") {
		t.Fatalf("must not mention tools that aren't installed, got %q", got)
	}
}

// TestIssue29_VersionHint_UvInstalled verifies the uv branch: with uv on PATH,
// VersionHint suggests "uv python install <version>" and embeds the version.
func TestIssue29_VersionHint_UvInstalled(t *testing.T) {
	d := mkTool(t, t.TempDir(), "uv")
	withPath(t, d)
	got := VersionHint("3.12")
	if !strings.Contains(got, "uv python install 3.12") {
		t.Fatalf("want uv suggestion with version, got %q", got)
	}
}

// TestIssue29_VersionHint_PyenvInstalled verifies the pyenv branch: with pyenv
// on PATH (and no uv), VersionHint suggests "pyenv install <version>".
func TestIssue29_VersionHint_PyenvInstalled(t *testing.T) {
	d := mkTool(t, t.TempDir(), "pyenv")
	withPath(t, d)
	got := VersionHint("3.12")
	if !strings.Contains(got, "pyenv install 3.12") {
		t.Fatalf("want pyenv suggestion with version, got %q", got)
	}
}

// TestIssue29_VersionHint_PrecedenceUvOverPyenv verifies that when both tools
// are installed, uv wins. This pins the documented check order so the output
// stays deterministic.
func TestIssue29_VersionHint_PrecedenceUvOverPyenv(t *testing.T) {
	d := t.TempDir()
	mkTool(t, d, "uv")
	mkTool(t, d, "pyenv")
	withPath(t, d)
	got := VersionHint("3.12")
	if !strings.Contains(got, "uv python install 3.12") {
		t.Fatalf("uv must win when both are present, got %q", got)
	}
}

// VersionHint: edge cases

// TestIssue29_VersionHint_UvEmptyVersion verifies that an empty version string
// defaults to "3" instead of producing a dangling "install " command. The
// default is only observable on the tool branches, so this checks it via uv.
func TestIssue29_VersionHint_UvEmptyVersion(t *testing.T) {
	d := mkTool(t, t.TempDir(), "uv")
	withPath(t, d)
	got := VersionHint("")
	if !strings.Contains(got, "uv python install 3") {
		t.Fatalf("uv + empty version should suggest install 3, got %q", got)
	}
}

// TestIssue29_VersionHint_MajorMinorOnly verifies that a major.minor version
// such as 3.9 survives verbatim: it is not truncated, padded, or rounded.
func TestIssue29_VersionHint_MajorMinorOnly(t *testing.T) {
	d := mkTool(t, t.TempDir(), "uv")
	withPath(t, d)
	// The issue example uses 3.12; make sure minor versions survive verbatim.
	if got := VersionHint("3.9"); !strings.Contains(got, "install 3.9") {
		t.Fatalf("want install 3.9, got %q", got)
	}
}

//Integration: warning is emitted in Analyze()

// writePyProject creates a minimal pip project in a temporary directory. The
// pyproject.toml asks for the given minimum Python version, and the presence of
// requirements.txt makes the adapter classify it as a pip project.
func writePyProject(t *testing.T, want string) string {
	t.Helper()
	dir := t.TempDir()
	body := "requires-python = \">=" + want + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// requirements.txt so the adapter classifies it as a pip project.
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// findPyWarning returns the first warning that contains substr, or fails the
// test with the full warning list so a missing warning is easy to diagnose.
func findPyWarning(t *testing.T, y *project.PyProject, substr string) string {
	t.Helper()
	for _, w := range y.Warnings {
		if strings.Contains(w, substr) {
			return w
		}
	}
	t.Fatalf("no warning containing %q; warnings=%v", substr, y.Warnings)
	return ""
}

// TestIssue29_Analyze_LowerBound_WarnsWithHint verifies the lower-bound path end
// to end: a project wanting Python 3.99 with 3.8 installed produces a warning
// that names the source file and includes the uv suggestion.
func TestIssue29_Analyze_LowerBound_WarnsWithHint(t *testing.T) {
	dir := writePyProject(t, "3.99") // impossible to satisfy
	d := mkTool(t, t.TempDir(), "uv")
	withPath(t, d)

	y := Analyze(dir, dir, "3.8") // pretend installed 3.8
	w := findPyWarning(t, y, "asks for Python 3.99")
	if !strings.Contains(w, "uv python install 3.99") {
		t.Fatalf("warning must include the hint, got %q", w)
	}
	if !strings.Contains(w, "requires-python") {
		t.Fatalf("warning must keep the source, got %q", w)
	}
}

// TestIssue29_Analyze_UpperBound_WarnsWithHint verifies the upper-bound path end
// to end: a project that supports Python below 3.10, run on 3.12, warns and
// includes the pyenv suggestion. This branch is easy to miss because the lower
// bound is empty.
func TestIssue29_Analyze_UpperBound_WarnsWithHint(t *testing.T) {
	// Project says "below 3.10", installed is 3.12 -> upper-bound warning.
	dir := t.TempDir()
	body := "requires-python = \"<3.10\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := mkTool(t, t.TempDir(), "pyenv")
	withPath(t, d)

	y := Analyze(dir, dir, "3.12")
	w := findPyWarning(t, y, "supports Python below 3.10")
	if !strings.Contains(w, "pyenv install 3.10") {
		t.Fatalf("upper-bound warning must include the hint, got %q", w)
	}
}

// TestIssue29_Analyze_MatchingVersion_NoWarning verifies that a compatible
// installed version produces no Python warning at all, so the hint is not shown
// when there is nothing to fix.
func TestIssue29_Analyze_MatchingVersion_NoWarning(t *testing.T) {
	dir := writePyProject(t, "3.9")
	withPath(t, t.TempDir())

	y := Analyze(dir, dir, "3.11") // newer than wanted -> no warning
	for _, w := range y.Warnings {
		if strings.Contains(w, "Python") {
			t.Fatalf("unexpected Python warning when version is fine: %q", w)
		}
	}
}

// TestIssue29_Analyze_NoInterpreter_NoHintNeeded verifies that when no Python
// interpreter is found, gclone gives the generic "not found in PATH" advice and
// does not append a tool-specific hint, which would name a version the user
// cannot install yet.
func TestIssue29_Analyze_NoInterpreter_NoHintNeeded(t *testing.T) {
	dir := writePyProject(t, "3.9")
	withPath(t, t.TempDir())

	y := Analyze(dir, dir, "") // interpreter == "" branch
	var found bool
	for _, w := range y.Warnings {
		if strings.Contains(w, "Python was not found in PATH") {
			found = true
			if strings.Contains(w, "uv python install") || strings.Contains(w, "pyenv install") {
				t.Fatalf("no-interpreter branch should not use VersionHint, got %q", w)
			}
		}
	}
	if !found {
		t.Fatalf("expected the 'not found in PATH' warning, got %v", y.Warnings)
	}
}
