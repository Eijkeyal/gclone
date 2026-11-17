package state

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rauni5/gclone/internal/project"
)

func sample() *project.Project {
	return &project.Project{
		URL:    "https://user:secret-token@github.com/someone/app.git",
		Commit: "abc1234",
		Steps: []project.Step{
			{Where: "repo root", Name: "Install dependencies (npm)", Cmd: "npm ci", Status: "ok"},
			{Where: "server", Name: "Download Go modules", Cmd: "go mod download", Status: "failed", Note: "exit status 1", Hint: "Check your internet connection."},
		},
		Needs: []*project.Need{{Kind: "mongodb", Port: 27017}},
	}
}

func TestSaveLoadRoundTripAndRedaction(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, FromProject(sample())); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(Path(dir))
	if strings.Contains(string(b), "secret-token") || strings.Contains(string(b), "user:") {
		t.Errorf("the remote's login must not be written: %s", b)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Commit != "abc1234" || len(s.Steps) != 2 || s.Services[0].Kind != "mongodb" || s.Source != "https://github.com/someone/app.git" {
		t.Errorf("unexpected state: %+v", s)
	}
}

func TestLoadMissingAndNewerSchema(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("want ErrNotExist, got %v", err)
	}
	os.MkdirAll(filepath.Join(dir, Dir), 0o755)
	os.WriteFile(Path(dir), []byte(`{"schema": 99}`), 0o644)
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "newer gclone") {
		t.Errorf("want a newer-schema error, got %v", err)
	}
}

func TestResume(t *testing.T) {
	prev := FromProject(sample())
	plan := []project.Planned{
		{Name: "Install dependencies (npm)", Where: ".", Cmd: "npm ci", Resumable: true},
		{Name: "Download Go modules", Where: "server", Cmd: "go mod download", Resumable: true}, // failed before
		{Name: "Start local services: mongo", Where: ".", Cmd: "docker compose up -d mongo"},    // never skipped
	}
	rem, skipped, why := Resume(prev, "abc1234", plan)
	if why != "" || len(skipped) != 1 || skipped[0].Name != "Install dependencies (npm)" || len(rem) != 2 {
		t.Fatalf("got remaining=%d skipped=%v why=%q", len(rem), skipped, why)
	}
	// A different commit means everything runs again.
	rem, skipped, why = Resume(prev, "def5678", plan)
	if len(rem) != 3 || len(skipped) != 0 || why == "" {
		t.Errorf("commit change should disable resume, got %d/%d %q", len(rem), len(skipped), why)
	}
	// No earlier state.
	if rem, _, why = Resume(nil, "abc1234", plan); len(rem) != 3 || why == "" {
		t.Errorf("no state should run everything")
	}
	// npm install becomes npm ci once a lockfile exists; that is still the same step.
	plan[0].Cmd = "npm install"
	if _, skipped, _ = Resume(prev, "abc1234", plan); len(skipped) != 1 {
		t.Errorf("the same step with a different command should still be skipped")
	}
	// An earlier successful step the new plan no longer contains is carried over.
	prev.Steps = append(prev.Steps, Step{Where: "repo root", Name: "Create .env from .env.example", Status: "ok"})
	if _, skipped, _ = Resume(prev, "abc1234", plan); len(skipped) != 2 {
		t.Errorf("a step that is done and no longer planned should be kept in the record: %v", skipped)
	}
}

func TestPrint(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := Print(&out, dir); err == nil || !strings.Contains(err.Error(), "gclone setup") {
		t.Errorf("want a hint to run setup, got %v", err)
	}
	if err := Save(dir, FromProject(sample())); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	err := Print(&out, dir)
	if !errors.Is(err, ErrProblems) {
		t.Errorf("a failed step should return ErrProblems, got %v", err)
	}
	for _, want := range []string{"[ok]", "[FAIL]", "Probably: Check your internet connection.", "not run yet", "--resume"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output should contain %q:\n%s", want, out.String())
		}
	}
	// A clean state with a READY verification returns no error.
	clean := FromProject(&project.Project{Commit: "abc1234", Steps: []project.Step{{Name: "x", Where: "repo root", Status: "ok"}}})
	clean.SetVerify(&project.Report{When: time.Now(), Verdict: "READY"})
	Save(dir, clean)
	out.Reset()
	if err := Print(&out, dir); err != nil {
		t.Errorf("clean state should not report problems: %v\n%s", err, out.String())
	}
}
