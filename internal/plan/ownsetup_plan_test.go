package plan

import (
	"strings"
	"testing"

	"github.com/rauni5/gclone/internal/project"
)

func projectWithOwnSetup() *project.Project {
	return &project.Project{Dir: ".", OwnSetup: []*project.OwnSetup{
		{Dir: ".", Rel: ".", Kind: "make", Argv: []string{"make", "setup"}, Shown: "make setup"},
	}}
}

func TestOwnSetupIsNotInThePlanByDefault(t *testing.T) {
	p := projectWithOwnSetup()
	for _, o := range []project.Options{{}, {Yes: true}, {CI: true, Yes: true}} {
		for _, s := range BuildPlan(p, o) {
			if strings.Contains(s.Name, "own setup") {
				t.Errorf("the repo's own setup must need --run-project-setup (options %+v)", o)
			}
		}
	}
}

func TestOwnSetupRunsLastWhenAskedFor(t *testing.T) {
	p := projectWithOwnSetup()
	steps := BuildPlan(p, project.Options{RunProjectSetup: true})
	if len(steps) != 1 || steps[0].Cmd != "make setup" || steps[0].Resumable {
		t.Fatalf("%+v", steps)
	}
	if !strings.Contains(steps[0].Name, "own setup") {
		t.Errorf("name %q", steps[0].Name)
	}
}

func TestOwnSetupNote(t *testing.T) {
	p := projectWithOwnSetup()
	if n := OwnSetupNote(p, project.Options{}); !strings.Contains(n, "make setup") || !strings.Contains(n, "--run-project-setup") {
		t.Errorf("note %q", n)
	}
	if OwnSetupNote(p, project.Options{RunProjectSetup: true}) != "" {
		t.Error("no note once it is included")
	}
	if OwnSetupNote(&project.Project{}, project.Options{}) != "" {
		t.Error("no note without an own setup")
	}
}

func TestMissingToolGivesAClearError(t *testing.T) {
	p := &project.Project{OwnSetup: []*project.OwnSetup{{Dir: ".", Rel: ".", Argv: []string{"definitely-not-installed-xyz", "setup"}, Shown: "definitely-not-installed-xyz setup"}}}
	steps := BuildPlan(p, project.Options{RunProjectSetup: true})
	err := steps[0].Run()
	if err == nil || !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("err %v", err)
	}
}
