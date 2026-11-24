package rust

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rauni5/gclone/internal/adapter"
	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/proc"
	"github.com/rauni5/gclone/internal/project"
)

// Adapter is the Rust ecosystem.
type Adapter struct{}

var _ adapter.Adapter = Adapter{}

func (Adapter) Name() string                                                { return "Rust" }
func (Adapter) Units(p *project.Project) int                                { return len(p.RustProjects) }
func (Adapter) Markers() []string                                           { return []string{"Cargo.toml"} }
func (Adapter) Prepare(*project.Project, project.Options) []project.Planned { return nil }
func (Adapter) GuideConfig(*project.Project) []string                       { return nil }

func (Adapter) Analyze(p *project.Project, root string) {
	for _, d := range FindProjects(root) {
		r, err := Analyze(root, d)
		if err != nil {
			p.Warnings = append(p.Warnings, fmt.Sprintf("%s: %v", fsx.Rel(root, d), err))
			continue
		}
		p.RustProjects = append(p.RustProjects, r)
	}
}

func (Adapter) Warnings(p *project.Project) []string {
	var out []string
	for _, r := range p.RustProjects {
		for _, w := range r.Warnings {
			out = append(out, r.Rel+": "+w)
		}
	}
	return out
}

func (Adapter) Detected(p *project.Project) []string {
	var out []string
	for _, r := range p.RustProjects {
		line := r.Rel + ": " + r.Name + ", Rust"
		if r.Edition != "" {
			line += " " + r.Edition + " edition"
		}
		if r.Workspace {
			line += ", workspace"
		}
		if r.Framework != "" {
			line += ", " + r.Framework
		}
		out = append(out, line)
	}
	return out
}

func (Adapter) Install(p *project.Project, o project.Options) []project.Planned {
	if o.NoInstall {
		return nil
	}
	var steps []project.Planned
	for _, r := range p.RustProjects {
		r := r
		steps = append(steps, project.Planned{
			Where:     r.Rel,
			Name:      "Download Rust crates",
			Resumable: true,
			Cmd:       "cargo fetch",
			Run: func() error {
				if _, err := exec.LookPath("cargo"); err != nil {
					return fmt.Errorf("cargo was not found in PATH, install Rust with rustup from https://rustup.rs")
				}
				return proc.RunStream(r.Dir, false, "cargo", "fetch")
			},
		})
	}
	return steps
}

func (Adapter) Runnables(p *project.Project, noBuild bool) []project.Runnable {
	return Runnables(p, noBuild)
}

func (Adapter) Next(p *project.Project) []string {
	var out []string
	for _, r := range p.RustProjects {
		prefix := ""
		if r.Rel != "." {
			prefix = "cd " + r.Rel + " && "
		}
		if r.HasBin {
			out = append(out, prefix+"cargo run")
		}
		out = append(out, prefix+"cargo test")
	}
	return out
}

func (Adapter) Guide(p *project.Project) string {
	var b strings.Builder
	w := func(format string, a ...interface{}) { fmt.Fprintf(&b, format+"\n", a...) }
	for _, r := range p.RustProjects {
		title := r.Name
		if r.Rel != "." {
			title += " (`" + r.Rel + "`)"
		}
		w("### %s", title)
		w("")
		stack := "Rust"
		if r.RustWanted != "" {
			stack += " " + r.RustWanted + " or newer"
		} else if isVersion(r.Toolchain) {
			stack += " " + r.Toolchain
		}
		if r.Framework != "" {
			stack += ", " + r.Framework
		}
		w("Stack: %s", stack)
		if r.Toolchain != "" && !isVersion(r.Toolchain) {
			w("")
			w("The project pins the `%s` toolchain (rust-toolchain file). rustup picks it when you run cargo in this folder.", r.Toolchain)
		}
		w("")
		w("```bash")
		if r.Rel != "." {
			w("cd %s", r.Rel)
		}
		w("cargo fetch                     # download crates")
		w("cargo build                     # compile (the first build takes a while)")
		if r.HasBin {
			w("cargo run                       # run it")
		}
		w("cargo test                      # tests")
		w("```")
		if len(r.DBHints) > 0 {
			w("")
			w("It uses %s. Start that service first; the README or an `.env.example` file says which connection setting to use.", strings.Join(r.DBHints, ", "))
		}
		w("")
	}
	return b.String()
}
