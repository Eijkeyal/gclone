package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rauni5/gclone/internal/adapters"
	"github.com/rauni5/gclone/internal/analyze"
	"github.com/rauni5/gclone/internal/buildinfo"
	"github.com/rauni5/gclone/internal/doctor"
	"github.com/rauni5/gclone/internal/project"
)

// DoctorResult is what `gclone doctor --json` prints.
type DoctorResult struct {
	Schema   int           `json:"schema"`
	Gclone   string        `json:"gclone"`
	OK       bool          `json:"ok"`
	Dir      string        `json:"dir"`
	Verdict  string        `json:"verdict"`
	Detected []string      `json:"detected"`
	Checks   []DoctorCheck `json:"checks"`
}

// DoctorCheck is one line of the report. Status is pass, warn or fail.
type DoctorCheck struct {
	Group  string `json:"group"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

var doctorTag = map[string]string{"pass": "[ok]  ", "warn": "[warn]", "fail": "[FAIL]"}

// runDoctor reports whether this machine can run the project in dir. It changes nothing.
// The result is false when the verdict is NOT READY.
func runDoctor(w io.Writer, dir string, asJSON bool, env doctor.Env) (bool, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	if info, e := os.Stat(abs); e != nil || !info.IsDir() {
		return false, errors.New(dir + " is not a directory")
	}
	p := analyze.AnalyzeProject(abs)
	checks := doctor.Run(p, env)
	if !p.HasUnits() {
		checks = append(checks, doctor.Check{Group: "Project", Name: "stack", Status: "warn",
			Detail: "no supported project found in " + abs + ", only the machine itself was checked"})
	}
	verdict := doctor.Verdict(checks)

	if asJSON {
		r := DoctorResult{Schema: ResultSchema, Gclone: buildinfo.Version, OK: verdict != "NOT READY",
			Dir: filepath.ToSlash(abs), Verdict: verdict, Detected: detected(p), Checks: []DoctorCheck{}}
		for _, c := range checks {
			r.Checks = append(r.Checks, DoctorCheck(c))
		}
		return r.OK, writeJSON(w, r)
	}

	fmt.Fprintf(w, "Can this machine run %s?\n", abs)
	for _, d := range detected(p) {
		fmt.Fprintf(w, "  Detected: %s\n", d)
	}
	group := ""
	for _, c := range checks {
		if c.Group != group {
			group = c.Group
			fmt.Fprintf(w, "\n%s\n", group)
		}
		fmt.Fprintf(w, "  %s %-12s %s\n", doctorTag[c.Status], c.Name, c.Detail)
	}
	fmt.Fprintf(w, "\nVerdict: %s\n", verdict)
	if verdict == "NOT READY" {
		fmt.Fprintln(w, "Fix the [FAIL] lines first. Nothing was changed on this machine.")
	} else {
		fmt.Fprintln(w, "Nothing was changed on this machine.")
	}
	return verdict != "NOT READY", nil
}

func detected(p *project.Project) []string {
	var out []string
	for _, a := range adapters.All() {
		out = append(out, a.Detected(p)...)
	}
	for _, s := range p.OwnSetup {
		out = append(out, s.Rel+": own setup command "+s.Shown+" (runs only with --run-project-setup)")
	}
	return out
}
