// Package state keeps a small record of what gclone did in a repository, in
// .gclone/state.json. `gclone status` reads it, and `gclone setup --resume` uses it to skip
// work that already succeeded. It holds names, commands and notes, never variable values.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/rauni5/gclone/internal/buildinfo"
	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/proc"
	"github.com/rauni5/gclone/internal/project"
)

// Dir is the folder inside the repository, and File the file inside it.
const (
	Dir    = ".gclone"
	File   = "state.json"
	Schema = 1
)

// ErrProblems is returned by Print when a step or the verification failed.
var ErrProblems = errors.New("gclone found problems, see above")

// Step is one setup step as recorded.
type Step struct {
	Where  string `json:"where"`
	Name   string `json:"name"`
	Cmd    string `json:"cmd"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

// Service is a database or service the project needs.
type Service struct {
	Kind   string `json:"kind"`
	Port   int    `json:"port"`
	Remote bool   `json:"remote"`
}

// Verify is the last verification result.
type Verify struct {
	At       string `json:"at"`
	Verdict  string `json:"verdict"`
	Failed   int    `json:"failed"`
	Warnings int    `json:"warnings"`
}

// State is the content of .gclone/state.json.
type State struct {
	Schema    int       `json:"schema"`
	Gclone    string    `json:"gclone"`
	UpdatedAt string    `json:"updated_at"`
	Source    string    `json:"source"`
	Commit    string    `json:"commit"`
	Steps     []Step    `json:"steps"`
	Services  []Service `json:"services,omitempty"`
	Verify    *Verify   `json:"verify,omitempty"`
}

// Path returns where the state file for a repository lives.
func Path(dir string) string { return filepath.Join(dir, Dir, File) }

// cleanURL removes a login from a remote address, so it is never written to disk.
func cleanURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = nil
	return u.String()
}

// FromProject builds the state for a project after setup.
func FromProject(p *project.Project) *State {
	s := &State{
		Schema:    Schema,
		Gclone:    buildinfo.Version,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Source:    cleanURL(p.URL),
		Commit:    p.Commit,
		Steps:     []Step{},
	}
	for _, st := range p.Steps {
		s.Steps = append(s.Steps, Step{Where: st.Where, Name: st.Name, Cmd: st.Cmd, Status: st.Status, Note: st.Note, Hint: st.Hint})
	}
	for _, n := range p.Needs {
		s.Services = append(s.Services, Service{Kind: n.Kind, Port: n.Port, Remote: n.Remote})
	}
	if p.Verify != nil {
		s.SetVerify(p.Verify)
	}
	return s
}

// SetVerify records a verification result.
func (s *State) SetVerify(r *project.Report) {
	f, w := r.Counts()
	s.Verify = &Verify{At: r.When.Format(time.RFC3339), Verdict: r.Verdict, Failed: f, Warnings: w}
	s.UpdatedAt = time.Now().Format(time.RFC3339)
}

// Load reads the state file. The error wraps os.ErrNotExist when there is none.
func Load(dir string) (*State, error) {
	b, err := os.ReadFile(Path(dir))
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", Path(dir), err)
	}
	if s.Schema > Schema {
		return nil, fmt.Errorf("%s was written by a newer gclone (schema %d), please update gclone", Path(dir), s.Schema)
	}
	return &s, nil
}

// Save writes the state file and keeps the folder out of `git status`.
func Save(dir string, s *State) error {
	if err := os.MkdirAll(filepath.Join(dir, Dir), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := Path(dir) + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, Path(dir)); err != nil {
		return err
	}
	fsx.ExcludeFromGit(dir, Dir+"/")
	return nil
}

// RecordVerify stores a verification result, creating the state file when needed.
func RecordVerify(dir string, p *project.Project, r *project.Report) error {
	s, err := Load(dir)
	if err != nil {
		s = FromProject(p)
	}
	s.SetVerify(r)
	return Save(dir, s)
}

// key identifies a step by what it is and where it runs. The command is left out on purpose:
// `npm install` becomes `npm ci` once the first install has created a lockfile.
func key(name, where string) string {
	return name + "|" + project.UnitLabel(where)
}

// Resume removes the steps that already succeeded. A step is skipped only when it is marked
// Resumable, succeeded last time, and the repository is still at the commit it was set up at.
// It returns the steps still to run and records for the work that is already done, which
// also includes earlier steps the new plan no longer needs (for example a created .env).
func Resume(prev *State, commit string, plan []project.Planned) (remaining []project.Planned, skipped []project.Step, reason string) {
	if prev == nil {
		return plan, nil, "no earlier setup was recorded"
	}
	if prev.Commit != commit {
		return plan, nil, fmt.Sprintf("the repository moved from commit %s to %s since the last setup", prev.Commit, commit)
	}
	done := map[string]Step{}
	for _, s := range prev.Steps {
		if s.Status == "ok" {
			done[key(s.Name, s.Where)] = s
		}
	}
	planned := map[string]bool{}
	for _, p := range plan {
		k := key(p.Name, p.Where)
		planned[k] = true
		if _, ok := done[k]; ok && p.Resumable {
			skipped = append(skipped, project.Step{Where: project.UnitLabel(p.Where), Name: p.Name, Cmd: p.Cmd, Status: "ok", Note: "done in an earlier run"})
			continue
		}
		remaining = append(remaining, p)
	}
	for _, s := range prev.Steps {
		if s.Status == "ok" && !planned[key(s.Name, s.Where)] {
			skipped = append(skipped, project.Step{Where: s.Where, Name: s.Name, Cmd: s.Cmd, Status: "ok", Note: "done in an earlier run"})
		}
	}
	return remaining, skipped, ""
}

func mark(status string) string {
	switch status {
	case "ok":
		return "[ok]  "
	case "failed":
		return "[FAIL]"
	}
	return "[..]  "
}

// Print shows the recorded state and compares it with the repository as it is now. It
// returns ErrProblems when a step or the last verification failed.
func Print(w io.Writer, dir string) error {
	s, err := Load(dir)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no gclone state in %s. Run `gclone setup %s` first", dir, dir)
	}
	if err != nil {
		return err
	}
	problems := false
	fmt.Fprintf(w, "gclone status for %s\n", dir)
	fmt.Fprintf(w, "Set up with gclone %s on %s, at commit %s\n", s.Gclone, formatTime(s.UpdatedAt), s.Commit)
	if s.Source != "" {
		fmt.Fprintf(w, "Source: %s\n", s.Source)
	}
	fmt.Fprintln(w, "\nSteps:")
	if len(s.Steps) == 0 {
		fmt.Fprintln(w, "  none recorded")
	}
	for _, st := range s.Steps {
		fmt.Fprintf(w, "  %s %s (in %s)\n", mark(st.Status), st.Name, project.UnitLabel(st.Where))
		if st.Status == "failed" {
			problems = true
			if st.Note != "" {
				fmt.Fprintf(w, "         %s\n", st.Note)
			}
			if st.Hint != "" {
				fmt.Fprintf(w, "         Probably: %s\n", st.Hint)
			}
		} else if st.Note != "" {
			fmt.Fprintf(w, "         %s\n", st.Note)
		}
	}
	if len(s.Services) > 0 {
		fmt.Fprintln(w, "\nServices the project needs:")
		for _, sv := range s.Services {
			where := "local"
			if sv.Remote {
				where = "remote"
			}
			fmt.Fprintf(w, "  - %s, port %d (%s)\n", sv.Kind, sv.Port, where)
		}
	}
	fmt.Fprintln(w, "\nVerification:")
	switch {
	case s.Verify == nil:
		fmt.Fprintf(w, "  not run yet. Try: gclone verify %s\n", dir)
	default:
		line := fmt.Sprintf("  %s on %s", s.Verify.Verdict, formatTime(s.Verify.At))
		if s.Verify.Failed > 0 || s.Verify.Warnings > 0 {
			line += fmt.Sprintf(" (%d failed, %d warning(s))", s.Verify.Failed, s.Verify.Warnings)
		}
		fmt.Fprintln(w, line)
		if s.Verify.Verdict == "NOT READY" || s.Verify.Verdict == "INTERRUPTED" {
			problems = true
		}
	}

	fmt.Fprintln(w, "\nRight now:")
	cur := proc.GitOutput(dir, "rev-parse", "--short", "HEAD")
	switch {
	case cur == "":
		fmt.Fprintln(w, "  - could not read the git commit")
	case cur != s.Commit:
		fmt.Fprintf(w, "  - the repository is at commit %s, but setup was done at %s. Run `gclone setup %s` again\n", cur, s.Commit, dir)
	default:
		fmt.Fprintln(w, "  - the repository is at the commit it was set up at")
	}
	if proc.GitOutput(dir, "status", "--porcelain", "--untracked-files=no") != "" {
		fmt.Fprintln(w, "  - there are uncommitted changes to tracked files, so setup may be out of date")
	}
	if problems {
		fmt.Fprintf(w, "\nFix the failed items, then run `gclone setup --resume %s`.\n", dir)
		return ErrProblems
	}
	return nil
}

func formatTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02 15:04")
}
