package rust

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rauni5/gclone/internal/project"
)

func TestParseManifest(t *testing.T) {
	m := ParseManifest(`
[package]
name = "demo-app"   # the name
edition = "2021"
rust-version = "1.75"

[dependencies]
axum = "0.7"
serde = { version = "1", features = ["derive"] }

[dependencies.tokio]
version = "1"

[dev-dependencies]
tower = "0.4"

[[bin]]
name = "demo"
`)
	if m.Name != "demo-app" || m.Edition != "2021" || m.RustWanted != "1.75" || !m.HasBinTbl || m.Workspace {
		t.Fatalf("got %+v", m)
	}
	have := strings.Join(m.Deps, ",")
	for _, d := range []string{"axum", "serde", "tokio", "tower"} {
		if !strings.Contains(have, d) {
			t.Errorf("dependency %q not found in %q", d, have)
		}
	}
}

func TestParseManifestWorkspace(t *testing.T) {
	m := ParseManifest("[workspace]\nmembers = [\"a\", \"b\"]\n\n[workspace.dependencies]\naxum = \"0.7\"\n")
	if !m.Workspace || m.Name != "" || len(m.Deps) != 1 || m.Deps[0] != "axum" {
		t.Fatalf("got %+v", m)
	}
}

func TestParseToolchain(t *testing.T) {
	cases := map[string]string{
		"[toolchain]\nchannel = \"1.75.0\"\n":           "1.75.0",
		"[toolchain]\nchannel = 'nightly-2024-05-01'\n": "nightly-2024-05-01",
		"stable\n": "stable",
		"":         "",
		"[toolchain]\ncomponents = [\"clippy\"]\n": "",
	}
	for in, want := range cases {
		if got := ParseToolchain(in); got != want {
			t.Errorf("ParseToolchain(%q)=%q want %q", in, got, want)
		}
	}
}

func TestOlder(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{{"1.70.0", "1.75", true}, {"1.75.0", "1.75", false}, {"1.82.1", "1.75", false}, {"", "1.75", true}}
	for _, c := range cases {
		if got := Older(c.a, c.b); got != c.want {
			t.Errorf("Older(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestFixture(t *testing.T) {
	a := Adapter{}
	root, _ := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "rust-axum"))
	p := &project.Project{Dir: root}
	a.Analyze(p, root)
	if a.Units(p) != 1 {
		t.Fatalf("units = %d", a.Units(p))
	}
	r := p.RustProjects[0]
	if r.Name != "rust-axum" || r.Framework != "Axum" || !r.HasBin || r.HasLib || r.Toolchain != "stable" || r.Edition != "2021" {
		t.Fatalf("got %+v", r)
	}
	if len(r.DBHints) != 1 || r.DBHints[0] != "sqlx" {
		t.Errorf("db hints = %v", r.DBHints)
	}
	d := a.Detected(p)
	if len(d) != 1 || !strings.Contains(d[0], "Rust 2021 edition, Axum") {
		t.Errorf("detected = %v", d)
	}
	steps := a.Install(p, project.Options{})
	if len(steps) != 1 || steps[0].Cmd != "cargo fetch" || !steps[0].Resumable {
		t.Errorf("install = %+v", steps)
	}
	if len(a.Install(p, project.Options{NoInstall: true})) != 0 {
		t.Error("NoInstall must give no steps")
	}
	rn := a.Runnables(p, false)
	if len(rn) != 1 || strings.Join(rn[0].Build, " ") != "cargo build" || rn[0].Start != nil || rn[0].NoStart == "" {
		t.Errorf("runnables = %+v", rn)
	}
	if len(a.Runnables(p, true)[0].Build) != 0 {
		t.Error("noBuild must skip the build")
	}
	g := a.Guide(p)
	for _, want := range []string{"cargo fetch", "cargo build", "cargo run", "cargo test", "sqlx", "Axum"} {
		if !strings.Contains(g, want) {
			t.Errorf("guide lacks %q:\n%s", want, g)
		}
	}
	if next := a.Next(p); len(next) != 2 || next[0] != "cargo run" {
		t.Errorf("next = %v", next)
	}
}

func TestWorkspaceFindsMemberDeps(t *testing.T) {
	root := t.TempDir()
	must := func(path, text string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("Cargo.toml", "[workspace]\nmembers = [\"crates/api\"]\n")
	must("crates/api/Cargo.toml", "[package]\nname = \"api\"\n[dependencies]\nactix-web = \"4\"\ndiesel = \"2\"\n")
	must("crates/api/src/main.rs", "fn main() {}\n")
	r, err := Analyze(root, root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Workspace || r.Framework != "Actix Web" || len(r.DBHints) != 1 || r.DBHints[0] != "Diesel" {
		t.Fatalf("got %+v", r)
	}
	if strings.Contains(strings.Join(r.Warnings, "|"), "no src/main.rs") {
		t.Errorf("a workspace must not get the no-main warning: %v", r.Warnings)
	}
}

func TestFindProjectsOneLevelDown(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"server", "target", ".hidden", "empty"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	for _, d := range []string{"server", "target", ".hidden"} {
		os.WriteFile(filepath.Join(root, d, "Cargo.toml"), []byte("[package]\nname=\"x\"\n"), 0o644)
	}
	got := FindProjects(root)
	if len(got) != 1 || filepath.Base(got[0]) != "server" {
		t.Fatalf("got %v", got)
	}
}

func TestPinnedToolchainWarning(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"x\"\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "src"), 0o755)
	os.WriteFile(filepath.Join(root, "src", "main.rs"), []byte("fn main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(root, "rust-toolchain.toml"), []byte("[toolchain]\nchannel = \"1.40.0\"\n"), 0o644)

	old := toolchainList
	defer func() { toolchainList = old }()

	toolchainList = func() (string, bool) { return "stable-x86_64-unknown-linux-gnu (default)\n", true }
	r, _ := Analyze(root, root)
	if !strings.Contains(strings.Join(r.Warnings, "|"), "1.40.0, which is not installed") {
		t.Errorf("want a not-installed warning, got %v", r.Warnings)
	}

	toolchainList = func() (string, bool) { return "1.40.0-x86_64-unknown-linux-gnu\nstable (default)\n", true }
	r, _ = Analyze(root, root)
	if strings.Contains(strings.Join(r.Warnings, "|"), "not installed") {
		t.Errorf("installed toolchain must not warn: %v", r.Warnings)
	}
}

// Analysis must not run tools inside the project folder, where rustup could download a
// toolchain. The rustc used here runs in the temp folder.
func TestInstalledDoesNotUseProjectFolder(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "rust-toolchain.toml"), []byte("[toolchain]\nchannel = \"1.40.0\"\n"), 0o644)
	t.Chdir(dir)
	_ = Installed() // would try to download 1.40.0 if it ran here and rustup allowed it
}
