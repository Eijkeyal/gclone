package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/rauni5/gclone/internal/adapters"
	"github.com/rauni5/gclone/internal/buildinfo"
	"github.com/rauni5/gclone/internal/proc"
	"github.com/rauni5/gclone/internal/project"
	"github.com/rauni5/gclone/internal/state"
)

// ResultSchema is the version of the JSON that --json prints. It changes only when a field
// is removed or changes meaning.
const ResultSchema = 1

// Result is what --json prints on standard output. It holds names, commands, notes and
// results only, never variable values, the same as the state file.
type Result struct {
	Schema   int             `json:"schema"`
	Gclone   string          `json:"gclone"`
	OK       bool            `json:"ok"`
	Dir      string          `json:"dir,omitempty"`
	Commit   string          `json:"commit,omitempty"`
	Detected []string        `json:"detected"`
	Warnings []string        `json:"warnings"`
	Steps    []state.Step    `json:"steps"`
	Services []state.Service `json:"services"`
	Verify   *VerifyResult   `json:"verify,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// VerifyResult is the outcome of the verification.
type VerifyResult struct {
	Verdict  string       `json:"verdict"`
	Failed   int          `json:"failed"`
	Warnings int          `json:"warnings"`
	Services []CheckJSON  `json:"services"`
	Units    []UnitResult `json:"units"`
}

// UnitResult holds the checks of one part of the repository.
type UnitResult struct {
	Label  string      `json:"label"`
	Checks []CheckJSON `json:"checks"`
}

// CheckJSON is one verification check. Status is pass, warn, fail or skip.
type CheckJSON struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func checks(cs []project.Check) []CheckJSON {
	out := []CheckJSON{}
	for _, c := range cs {
		out = append(out, CheckJSON{Name: c.Name, Status: c.Status, Detail: c.Detail})
	}
	return out
}

func verifyBad(verdict string) bool { return verdict == "NOT READY" || verdict == "INTERRUPTED" }

// failedSteps counts the setup steps that failed.
func failedSteps(p *project.Project) int {
	n := 0
	for _, s := range p.Steps {
		if s.Status == "failed" {
			n++
		}
	}
	return n
}

// BuildResult turns what a run produced into the JSON document. p may be nil when the run
// stopped before anything was analyzed.
func BuildResult(dir string, p *project.Project, err error) Result {
	r := Result{Schema: ResultSchema, Gclone: buildinfo.Version, Dir: dir,
		Detected: []string{}, Warnings: []string{}, Steps: []state.Step{}, Services: []state.Service{}}
	if p != nil {
		st := state.FromProject(p)
		r.Commit = p.Commit
		r.Steps = st.Steps
		if st.Services != nil {
			r.Services = st.Services
		}
		r.Warnings = append(r.Warnings, p.Warnings...)
		for _, a := range adapters.All() {
			r.Detected = append(r.Detected, a.Detected(p)...)
			r.Warnings = append(r.Warnings, a.Warnings(p)...)
		}
		if p.Verify != nil {
			f, w := p.Verify.Counts()
			v := &VerifyResult{Verdict: p.Verify.Verdict, Failed: f, Warnings: w,
				Services: checks(p.Verify.Services), Units: []UnitResult{}}
			for _, u := range p.Verify.Units {
				v.Units = append(v.Units, UnitResult{Label: u.Label, Checks: checks(u.Checks)})
			}
			r.Verify = v
		}
	}
	r.OK = err == nil && (p == nil || failedSteps(p) == 0) && (r.Verify == nil || !verifyBad(r.Verify.Verdict))
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

// WriteResult prints the document as indented JSON.
func WriteResult(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// statusJSON prints the recorded state for `gclone status --json`. The second result is
// false when a step or the last verification failed.
func statusJSON(w io.Writer, dir string) (ok bool, err error) {
	s, err := state.Load(dir)
	if errors.Is(err, os.ErrNotExist) {
		return false, errors.New("no gclone state in " + dir + ". Run `gclone setup " + dir + "` first")
	}
	if err != nil {
		return false, err
	}
	ok = true
	for _, st := range s.Steps {
		if st.Status == "failed" {
			ok = false
		}
	}
	if s.Verify != nil && verifyBad(s.Verify.Verdict) {
		ok = false
	}
	cur := ""
	if c := gitCommit(dir); c != "" {
		cur = c
	}
	doc := struct {
		OK            bool         `json:"ok"`
		Dir           string       `json:"dir"`
		State         *state.State `json:"state"`
		CurrentCommit string       `json:"current_commit,omitempty"`
		UpToDate      bool         `json:"up_to_date"`
	}{ok, filepath.ToSlash(dir), s, cur, cur != "" && cur == s.Commit}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return ok, enc.Encode(doc)
}

func gitCommit(dir string) string { return proc.GitOutput(dir, "rev-parse", "--short", "HEAD") }
