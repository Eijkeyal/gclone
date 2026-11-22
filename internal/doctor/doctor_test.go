package doctor

import (
	"strings"
	"testing"

	"github.com/rauni5/gclone/internal/project"
)

func fake() Env {
	return Env{
		Have:        func(string) bool { return true },
		Version:     func(bin string, _ ...string) string { return bin + " 1.0" },
		DockerState: func() string { return "ready" },
		PortFree:    func(int) bool { return true },
		Reachable:   func(int) bool { return false },
		FreeBytes:   func(string) (uint64, error) { return 50 << 30, nil },
		VenvWorks:   func() bool { return true },
	}
}

func find(cs []Check, group, name string) *Check {
	for i := range cs {
		if cs[i].Group == group && cs[i].Name == name {
			return &cs[i]
		}
	}
	return nil
}

func TestHealthyMachine(t *testing.T) {
	p := &project.Project{Dir: t.TempDir(), GoModules: []*project.GoModule{{Rel: "."}}}
	cs := Run(p, fake())
	if v := Verdict(cs); v != "READY" {
		t.Errorf("verdict %s: %+v", v, cs)
	}
	if c := find(cs, "Tools", "go"); c == nil || c.Status != "pass" || c.Detail != "go 1.0" {
		t.Errorf("go check: %+v", c)
	}
}

func TestMissingToolFails(t *testing.T) {
	env := fake()
	env.Have = func(b string) bool { return b != "node" }
	p := &project.Project{Dir: t.TempDir(), Pkgs: []*project.Pkg{{Rel: ".", PM: "npm"}}}
	cs := Run(p, env)
	if c := find(cs, "Tools", "node"); c == nil || c.Status != "fail" {
		t.Errorf("node: %+v", c)
	}
	if Verdict(cs) != "NOT READY" {
		t.Error("a missing tool must make the verdict NOT READY")
	}
}

func TestPackageManagerNamedOnce(t *testing.T) {
	p := &project.Project{Dir: t.TempDir(), Pkgs: []*project.Pkg{{Rel: "a", PM: "pnpm"}, {Rel: "b", PM: "pnpm"}, {Rel: "c"}}}
	n := 0
	for _, c := range Run(p, fake()) {
		if c.Group == "Tools" && (c.Name == "pnpm" || c.Name == "npm") {
			n++
		}
	}
	if n != 2 {
		t.Errorf("want one pnpm and one npm check, got %d", n)
	}
}

func TestDockerAndServices(t *testing.T) {
	env := fake()
	env.DockerState = func() string { return "stopped" }
	p := &project.Project{Dir: t.TempDir(), Needs: []*project.Need{{Kind: "postgres", Name: "PostgreSQL", Port: 5432}, {Kind: "redis", Name: "Redis", Port: 6379, Remote: true}}}
	cs := Run(p, env)
	if c := find(cs, "Services", "docker"); c == nil || c.Status != "warn" {
		t.Errorf("docker: %+v", c)
	}
	if c := find(cs, "Services", "PostgreSQL"); c == nil || c.Status != "warn" || !strings.Contains(c.Detail, "5432") {
		t.Errorf("postgres: %+v", c)
	}
	if find(cs, "Services", "Redis") != nil {
		t.Error("a remote service is not this machine's business")
	}
	env.Reachable = func(int) bool { return true }
	if c := find(Run(p, env), "Services", "PostgreSQL"); c.Status != "pass" {
		t.Errorf("a running database must pass: %+v", c)
	}
	// Docker is not needed when nothing needs it.
	if find(Run(&project.Project{Dir: t.TempDir()}, env), "Services", "docker") != nil {
		t.Error("docker checked with no services")
	}
}

func TestFirstLineSkipsJavaNotice(t *testing.T) {
	out := "Picked up JAVA_TOOL_OPTIONS: -Dx=y\nopenjdk version \"17.0.2\" 2022-01-18\nOpenJDK Runtime"
	if got := firstLine(out); got != `openjdk version "17.0.2" 2022-01-18` {
		t.Errorf("got %q", got)
	}
	if firstLine("") != "" || len(firstLine(strings.Repeat("x", 300))) > 110 {
		t.Error("empty and long output")
	}
}

func TestMissingToolNotReportedTwice(t *testing.T) {
	p := &project.Project{Dir: t.TempDir(), Warnings: []string{".: Flutter was not found in PATH, install it"}}
	for _, c := range Run(p, fake()) {
		if c.Group == "Project" {
			t.Errorf("duplicate line: %+v", c)
		}
	}
}

func TestOwnSetupToolMissingIsOnlyAWarning(t *testing.T) {
	env := fake()
	env.Have = func(b string) bool { return b != "make" }
	p := &project.Project{Dir: t.TempDir(), OwnSetup: []*project.OwnSetup{{Rel: ".", Argv: []string{"make", "setup"}, Shown: "make setup"}}}
	cs := Run(p, env)
	c := find(cs, "Tools", "make")
	if c == nil || c.Status != "warn" {
		t.Fatalf("%+v", c)
	}
	if Verdict(cs) != "READY WITH WARNINGS" {
		t.Error("an optional tool must not make the machine NOT READY")
	}
}

func TestDisk(t *testing.T) {
	env := fake()
	for free, want := range map[uint64]string{50 << 30: "pass", 1 << 30: "warn", 100 << 20: "fail"} {
		env.FreeBytes = func(string) (uint64, error) { return free, nil }
		c := find(Run(&project.Project{Dir: t.TempDir()}, env), "Disk", "free space")
		if c == nil || c.Status != want {
			t.Errorf("%d bytes: got %+v, want %s", free, c, want)
		}
	}
}

func TestProjectWarningsAreWarnings(t *testing.T) {
	p := &project.Project{Dir: t.TempDir(), Warnings: []string{"Multiple packages expect port 3000"}}
	cs := Run(p, fake())
	if c := find(cs, "Project", "warning"); c == nil || c.Status != "warn" {
		t.Errorf("%+v", c)
	}
	if Verdict(cs) != "READY WITH WARNINGS" {
		t.Error("verdict")
	}
}
