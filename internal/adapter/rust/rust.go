// Package rust is the Rust adapter: it finds Cargo packages and workspaces, reads
// Cargo.toml and rust-toolchain, compares the Rust version the project asks for with the
// one installed, and says how to fetch, build and test it.
//
// Cargo.toml is read with a small line parser, not a full TOML parser (gclone has no
// dependencies). It understands the tables and keys it needs and ignores the rest.
package rust

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/project"
)

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

// Manifest is what gclone reads from a Cargo.toml.
type Manifest struct {
	Name       string
	Edition    string
	RustWanted string
	Workspace  bool
	HasBinTbl  bool
	Deps       []string // names from [dependencies], [dev-dependencies] and [workspace.dependencies]
}

// ParseManifest reads the parts of a Cargo.toml that gclone needs.
func ParseManifest(text string) Manifest {
	var m Manifest
	section := ""
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.Trim(line, "[] ")
			switch section {
			case "workspace":
				m.Workspace = true
			case "bin":
				m.HasBinTbl = true
			}
			if strings.HasPrefix(section, "workspace") {
				m.Workspace = true
			}
			// A table such as [dependencies.serde] names a dependency.
			if i := strings.Index(section, "dependencies."); i >= 0 && (i == 0 || section[i-1] == '-' || section[i-1] == '.') {
				m.Deps = append(m.Deps, strings.Trim(section[i+len("dependencies."):], `"`))
			}
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		switch section {
		case "package":
			switch key {
			case "name":
				m.Name = unquote(val)
			case "edition":
				m.Edition = unquote(val)
			case "rust-version":
				m.RustWanted = unquote(val)
			}
		case "dependencies", "dev-dependencies", "build-dependencies", "workspace.dependencies":
			m.Deps = append(m.Deps, strings.Trim(key, `"`))
		}
	}
	return m
}

func stripComment(s string) string {
	inStr := false
	for i, r := range s {
		switch {
		case r == '"':
			inStr = !inStr
		case r == '#' && !inStr:
			return s[:i]
		}
	}
	return s
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// ParseToolchain reads the channel from rust-toolchain.toml, or from an old-style
// rust-toolchain file that holds just the channel.
func ParseToolchain(text string) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return ""
	}
	if !strings.Contains(t, "=") && !strings.Contains(t, "[") {
		return strings.TrimSpace(strings.SplitN(t, "\n", 2)[0])
	}
	for _, raw := range strings.Split(t, "\n") {
		key, val, ok := strings.Cut(stripComment(raw), "=")
		if ok && strings.TrimSpace(key) == "channel" {
			return unquote(val)
		}
	}
	return ""
}

// FindProjects returns the folders that hold a Cargo.toml: just the root when it has one
// (a workspace is built from its root), otherwise folders one level down.
func FindProjects(root string) []string {
	if fsx.FileExists(filepath.Join(root, "Cargo.toml")) {
		return []string{root}
	}
	var dirs []string
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || fsx.SkipDirs[e.Name()] || e.Name() == "target" {
			continue
		}
		d := filepath.Join(root, e.Name())
		if fsx.FileExists(filepath.Join(d, "Cargo.toml")) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// rustupCmd runs a rustup tool without ever letting rustup download anything. It runs in a
// neutral folder, not in the project: in a folder with a rust-toolchain file, rustup may
// install the pinned toolchain just because a version was asked for, and analysis must not
// change the machine.
func rustupCmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), "RUSTUP_AUTO_INSTALL=0")
	return cmd.Output()
}

// Installed returns the version of the default rustc, such as "1.82.0", or "" when rustc
// is missing. It does not read the project's rust-toolchain file (see rustupCmd).
func Installed() string {
	out, err := rustupCmd("rustc", "--version")
	if err != nil {
		return ""
	}
	if m := versionRe.FindStringSubmatch(string(out)); m != nil {
		return m[0]
	}
	return ""
}

// toolchainList returns the output of `rustup toolchain list`, and false when rustup is not
// installed. It is a variable so tests can replace it.
var toolchainList = func() (string, bool) {
	out, err := rustupCmd("rustup", "toolchain", "list")
	if err != nil {
		return "", false
	}
	return string(out), true
}

// toolchainInstalled reports whether a pinned version such as "1.75.0" is installed.
// known is false when rustup is missing, so nothing can be said.
func toolchainInstalled(version string) (installed, known bool) {
	list, ok := toolchainList()
	if !ok {
		return false, false
	}
	for _, line := range strings.Split(list, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), version) {
			return true, true
		}
	}
	return false, true
}

func parse(v string) [3]int {
	var r [3]int
	if m := versionRe.FindStringSubmatch(v); m != nil {
		r[0], _ = strconv.Atoi(m[1])
		r[1], _ = strconv.Atoi(m[2])
		r[2], _ = strconv.Atoi(m[3])
	}
	return r
}

// Older reports whether version a is older than b ("1.70" is older than "1.75.1").
func Older(a, b string) bool {
	x, y := parse(a), parse(b)
	for i := range x {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}

var frameworks = []struct{ crate, name string }{
	{"axum", "Axum"}, {"actix-web", "Actix Web"}, {"rocket", "Rocket"},
	{"poem", "Poem"}, {"warp", "Warp"}, {"tide", "Tide"}, {"salvo", "Salvo"},
	{"leptos", "Leptos"}, {"yew", "Yew"}, {"tauri", "Tauri"}, {"bevy", "Bevy"},
}

var dbCrates = map[string]string{
	"sqlx": "sqlx", "diesel": "Diesel", "sea-orm": "SeaORM", "tokio-postgres": "PostgreSQL",
	"postgres": "PostgreSQL", "mysql": "MySQL", "mongodb": "MongoDB", "redis": "Redis", "rusqlite": "SQLite",
}

func hasFile(dir string, parts ...string) bool {
	return fsx.FileExists(filepath.Join(append([]string{dir}, parts...)...))
}

func hasDir(dir, name string) bool {
	st, err := os.Stat(filepath.Join(dir, name))
	return err == nil && st.IsDir()
}

// Analyze reads one project. It only reads files, apart from running `rustc --version`.
func Analyze(root, dir string) (*project.RustProject, error) {
	b, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return nil, err
	}
	m := ParseManifest(string(b))
	name := m.Name
	if name == "" {
		name = filepath.Base(dir)
	}
	r := &project.RustProject{
		Dir: dir, Rel: fsx.Rel(root, dir), Name: name, Edition: m.Edition,
		RustWanted: m.RustWanted, Workspace: m.Workspace,
		HasBin:  m.HasBinTbl || hasFile(dir, "src", "main.rs") || hasDir(dir, filepath.Join("src", "bin")),
		HasLib:  hasFile(dir, "src", "lib.rs"),
		HasLock: hasFile(dir, "Cargo.lock"),
	}
	for _, f := range []string{"rust-toolchain.toml", "rust-toolchain"} {
		if tb, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
			r.Toolchain = ParseToolchain(string(tb))
			break
		}
	}

	deps := map[string]bool{}
	for _, d := range m.Deps {
		deps[d] = true
	}
	// A workspace keeps its dependencies in the member crates, so look one level in.
	if m.Workspace {
		for _, d := range memberDeps(dir) {
			deps[d] = true
		}
	}
	for _, f := range frameworks {
		if deps[f.crate] {
			r.Framework = f.name
			break
		}
	}
	seen := map[string]bool{}
	for d := range deps {
		if n, ok := dbCrates[d]; ok && !seen[n] {
			seen[n] = true
			r.DBHints = append(r.DBHints, n)
		}
	}
	sort.Strings(r.DBHints)

	r.Installed = Installed()
	switch {
	case r.Installed == "":
		r.Warnings = append(r.Warnings, "Rust was not found in PATH. Install it with rustup from https://rustup.rs")
	case m.RustWanted != "" && Older(r.Installed, m.RustWanted):
		r.Warnings = append(r.Warnings, fmt.Sprintf(
			"Cargo.toml asks for Rust %s or newer but Rust %s is installed. Update it with `rustup update`", m.RustWanted, r.Installed))
	}
	if isVersion(r.Toolchain) {
		if have, known := toolchainInstalled(r.Toolchain); known && !have {
			r.Warnings = append(r.Warnings, fmt.Sprintf(
				"rust-toolchain pins Rust %s, which is not installed. Run `rustup toolchain install %s` (cargo may otherwise download it the first time it runs)", r.Toolchain, r.Toolchain))
		} else if !known && Older(r.Installed, r.Toolchain) {
			r.Warnings = append(r.Warnings, fmt.Sprintf(
				"rust-toolchain asks for Rust %s but Rust %s is installed", r.Toolchain, r.Installed))
		}
	}
	if !r.HasBin && !r.HasLib && !m.Workspace {
		r.Warnings = append(r.Warnings, "no src/main.rs or src/lib.rs found, gclone only fetches and builds it")
	}
	if len(r.DBHints) > 0 {
		r.Warnings = append(r.Warnings, "uses "+strings.Join(r.DBHints, ", ")+": it likely needs a running database, check the README for the connection setting")
	}
	return r, nil
}

func isVersion(s string) bool { return s != "" && s[0] >= '0' && s[0] <= '9' }

// memberDeps lists dependency names from the Cargo.toml of folders one level down.
func memberDeps(dir string) []string {
	var out []string
	for _, group := range []string{".", "crates", "apps", "services"} {
		base := filepath.Join(dir, group)
		entries, _ := os.ReadDir(base)
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || e.Name() == "target" {
				continue
			}
			if b, err := os.ReadFile(filepath.Join(base, e.Name(), "Cargo.toml")); err == nil {
				out = append(out, ParseManifest(string(b)).Deps...)
			}
		}
	}
	return out
}

// BuildArgs is the build command.
func BuildArgs(*project.RustProject) []string {
	return []string{"cargo", "build"}
}

// Runnables describes how to build each project. gclone cannot know which port a Rust
// server listens on, so nothing is started; verification builds it and says so.
func Runnables(p *project.Project, noBuild bool) []project.Runnable {
	var out []project.Runnable
	for _, r := range p.RustProjects {
		note := "Rust project: built, not started (run it by hand, see the guide)"
		if r.Framework != "" && r.HasBin {
			note = "Rust project (" + r.Framework + "): built, not started because the port is not known (run it by hand, see the guide)"
		}
		run := project.Runnable{Label: r.Rel + " (Rust)", Dir: r.Dir, NoStart: note}
		if !noBuild {
			run.Build = BuildArgs(r)
		}
		out = append(out, run)
	}
	return out
}
