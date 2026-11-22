package devcontainer

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rauni5/gclone/internal/project"
)

func decode(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, s)
	}
	return m
}

func TestNodeProject(t *testing.T) {
	p := &project.Project{Pkgs: []*project.Pkg{{Rel: ".", PM: "npm", HasLock: true, NodeWanted: ">=20", EnvExample: ".env.example", Port: "3000"}}}
	r, err := Build(p, "app")
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, r.JSON)
	if m["image"] == nil || m["dockerComposeFile"] != nil {
		t.Errorf("no services: want a plain image, got %v", m)
	}
	feat := m["features"].(map[string]interface{})["ghcr.io/devcontainers/features/node:1"].(map[string]interface{})
	if feat["version"] != "20" {
		t.Errorf("node version %v", feat["version"])
	}
	cmd := m["postCreateCommand"].(string)
	if !strings.Contains(cmd, "[ -f .env ] || cp .env.example .env") || !strings.Contains(cmd, "npm ci") {
		t.Errorf("postCreateCommand %q", cmd)
	}
	if strings.Contains(r.JSON, `\u0026`) {
		t.Error("&& must stay readable")
	}
	if r.Compose != "" {
		t.Error("compose without services")
	}
}

func TestNoLockfileAndPnpm(t *testing.T) {
	p := &project.Project{Pkgs: []*project.Pkg{{Rel: "web", PM: "pnpm"}}}
	r, _ := Build(p, "x")
	cmd := decode(t, r.JSON)["postCreateCommand"].(string)
	if !strings.Contains(cmd, "corepack enable") || !strings.Contains(cmd, "(cd web && pnpm install)") {
		t.Errorf("%q", cmd)
	}
	if v := decode(t, r.JSON)["features"].(map[string]interface{})["ghcr.io/devcontainers/features/node:1"].(map[string]interface{})["version"]; v != "lts" {
		t.Errorf("default node version %v", v)
	}
}

func TestPythonUsesLinuxPaths(t *testing.T) {
	p := &project.Project{PyProjects: []*project.PyProject{
		{Rel: ".", Manager: "pip", Requirements: "requirements.txt", PyWanted: "3.11", Framework: "Flask", Port: "5000", Start: []string{"-m", "flask"}},
	}}
	r, _ := Build(p, "x")
	cmd := decode(t, r.JSON)["postCreateCommand"].(string)
	if cmd != "python -m venv .venv && .venv/bin/python -m pip install -r requirements.txt" {
		t.Errorf("%q", cmd)
	}
	if strings.Contains(cmd, `\`) {
		t.Error("a Windows path leaked into the container commands")
	}
	m := decode(t, r.JSON)
	if m["forwardPorts"].([]interface{})[0].(float64) != 5000 {
		t.Errorf("forwardPorts %v", m["forwardPorts"])
	}
}

func TestPoetryInstallsTheTool(t *testing.T) {
	p := &project.Project{PyProjects: []*project.PyProject{{Rel: ".", Manager: "poetry"}}}
	r, _ := Build(p, "x")
	if cmd := decode(t, r.JSON)["postCreateCommand"].(string); cmd != "pip install poetry && poetry install" {
		t.Errorf("%q", cmd)
	}
}

func TestGoAndJava(t *testing.T) {
	p := &project.Project{
		GoModules:    []*project.GoModule{{Rel: ".", GoWanted: "1.24"}},
		JavaProjects: []*project.JavaProject{{Rel: "api", Build: "maven", JavaWanted: 21}},
	}
	r, _ := Build(p, "x")
	f := decode(t, r.JSON)["features"].(map[string]interface{})
	if f["ghcr.io/devcontainers/features/go:1"].(map[string]interface{})["version"] != "1.24" {
		t.Error("go version")
	}
	j := f["ghcr.io/devcontainers/features/java:1"].(map[string]interface{})
	if j["version"] != "21" || j["installMaven"] != true || j["installGradle"] != false {
		t.Errorf("java %v", j)
	}
}

func TestServicesBecomeCompose(t *testing.T) {
	p := &project.Project{
		Pkgs: []*project.Pkg{{Rel: ".", PM: "npm"}},
		Needs: []*project.Need{
			{Kind: "postgres", Name: "PostgreSQL", Port: 5432},
			{Kind: "redis", Name: "Redis", Port: 6379},
			{Kind: "mongodb", Name: "MongoDB", Port: 27017, Remote: true},
		},
	}
	r, _ := Build(p, "x")
	m := decode(t, r.JSON)
	if m["service"] != "app" || m["dockerComposeFile"] != "docker-compose.yml" || m["image"] != nil {
		t.Errorf("%v", m)
	}
	for _, want := range []string{"postgres:16", "redis:7", "network_mode: service:postgres", "POSTGRES_PASSWORD: postgres", "postgres-data:"} {
		if !strings.Contains(r.Compose, want) {
			t.Errorf("compose lacks %q:\n%s", want, r.Compose)
		}
	}
	if strings.Contains(r.Compose, "mongo") {
		t.Error("a remote database must not get a container")
	}
	// The first service must not point at itself.
	if strings.Contains(r.Compose, "  postgres:\n    image: postgres:16\n    restart: unless-stopped\n    network_mode") {
		t.Error("the first service must not use network_mode")
	}
	if got := r.Files(); len(got) != 2 {
		t.Errorf("files %v", got)
	}
}

func TestLoginNoteOnlyWhenThereIsALogin(t *testing.T) {
	mongo, _ := Build(&project.Project{Needs: []*project.Need{{Kind: "mongodb", Port: 27017}}, GoModules: []*project.GoModule{{Rel: "."}}}, "x")
	for _, n := range mongo.Notes {
		if strings.Contains(n, "logins") {
			t.Errorf("MongoDB has no default login to mention: %q", n)
		}
	}
	pg, _ := Build(&project.Project{Needs: []*project.Need{{Kind: "postgres", Port: 5432}}, GoModules: []*project.GoModule{{Rel: "."}}}, "x")
	found := false
	for _, n := range pg.Notes {
		found = found || strings.Contains(n, "password `postgres`")
	}
	if !found {
		t.Errorf("notes %v", pg.Notes)
	}
}

func TestFlutterIsSkippedWithANote(t *testing.T) {
	p := &project.Project{FlutterProjects: []*project.FlutterProject{{Rel: "."}}}
	r, _ := Build(p, "x")
	found := false
	for _, n := range r.Notes {
		found = found || strings.Contains(n, "Flutter")
	}
	if !found {
		t.Errorf("notes %v", r.Notes)
	}
}

func TestOutputHasNoValuesFromEnvFiles(t *testing.T) {
	p := &project.Project{Pkgs: []*project.Pkg{{Rel: ".", PM: "npm", EnvVars: []project.EnvVar{{Key: "API_KEY", Value: "super-secret-value"}}}}}
	r, _ := Build(p, "x")
	if strings.Contains(r.JSON+r.Compose, "super-secret-value") {
		t.Error("a value from an env file reached the output")
	}
}
