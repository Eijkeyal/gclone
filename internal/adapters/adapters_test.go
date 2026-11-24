package adapters_test

import (
	"path/filepath"
	"testing"

	"github.com/rauni5/gclone/internal/adapters"
	"github.com/rauni5/gclone/internal/project"
)

// Every adapter must cope with a repository that holds none of its projects.
func TestAdaptersOnEmptyProject(t *testing.T) {
	p := &project.Project{Dir: t.TempDir()}
	seen := map[string]bool{}
	for _, a := range adapters.All() {
		if a.Name() == "" || seen[a.Name()] {
			t.Errorf("adapter name %q is empty or duplicated", a.Name())
		}
		seen[a.Name()] = true
		a.Analyze(p, p.Dir)
		if a.Units(p) != 0 || len(a.Detected(p)) != 0 || len(a.Warnings(p)) != 0 ||
			len(a.Prepare(p, project.Options{})) != 0 || len(a.Install(p, project.Options{})) != 0 ||
			len(a.Runnables(p, false)) != 0 || len(a.Next(p)) != 0 || a.Guide(p) != "" || len(a.GuideConfig(p)) != 0 {
			t.Errorf("%s returned something for an empty repository", a.Name())
		}
	}
}

// A repository with every ecosystem is picked up by the matching adapter each
// (extra fixture folders are allowed).
func TestAdaptersOnMixedRepository(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	p := &project.Project{Dir: root}
	got := map[string]int{}
	detected := map[string][]string{}
	for _, a := range adapters.All() {
		a.Analyze(p, root)
		got[a.Name()] = a.Units(p)
		detected[a.Name()] = a.Detected(p)
	}
	want := map[string]int{"Node.js": 1, "Go": 1, "Java": 1, "Python": 2, "Flutter": 1, "Rust": 1}
	for k, v := range want {
		// At least: adding another fixture under testdata must not break this test.
		if got[k] < v {
			t.Errorf("%s: got %d projects, want at least %d (found: %v)", k, got[k], v, detected[k])
		}
	}
}
