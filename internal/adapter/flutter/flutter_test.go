package flutter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rauni5/gclone/internal/project"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const app = `name: demo
environment:
  sdk: ">=3.3.0 <4.0.0"
  flutter: ">=3.19.0"
dependencies:
  flutter:
    sdk: flutter
`

func TestIsFlutter(t *testing.T) {
	if !isFlutter(app) {
		t.Error("app not recognised")
	}
	dart := "name: tool\nenvironment:\n  sdk: '>=3.0.0 <4.0.0'\ndependencies:\n  args: ^2.0.0\n"
	if isFlutter(dart) {
		t.Error("plain Dart package taken for a Flutter app")
	}
}

func TestWantedAndLower(t *testing.T) {
	d, f := Wanted(app)
	if d != "3.3.0" || f != "3.19.0" {
		t.Errorf("got %q %q", d, f)
	}
	if Lower("^3.5.1") != "3.5.1" || Lower("any") != "" {
		t.Error("Lower")
	}
	if !Older("3.19.0", "3.24.1") || Older("3.24.1", "3.24.1") || Older("3.30.0", "3.4.0") {
		t.Error("Older")
	}
}

func TestFindProjects(t *testing.T) {
	d := t.TempDir()
	write(t, d, "mobile/pubspec.yaml", app)
	write(t, d, "tools/pubspec.yaml", "name: t\ndependencies:\n  args: ^2.0.0\n")
	got := FindProjects(d)
	if len(got) != 1 || filepath.Base(got[0]) != "mobile" {
		t.Errorf("got %v", got)
	}
}

func TestAnalyze(t *testing.T) {
	d := t.TempDir()
	write(t, d, "pubspec.yaml", app+"  firebase_core: ^3.0.0\nflutter:\n  assets:\n    - .env\n")
	write(t, d, "android/app/build.gradle", "")
	write(t, d, "lib/api.dart", "const u = 'http://10.0.2.2:8080/x'; const v = 'http://localhost:3000';")
	write(t, d, "test/a_test.dart", "")
	write(t, d, ".env.example", "API_KEY=\n")
	f := Analyze(d, d, "3.10.0", "3.0.0")
	if f.Name != "demo" || !f.Firebase || !f.EnvAsset || f.EnvExample != ".env.example" || !f.HasTests {
		t.Fatalf("got %+v", f)
	}
	if strings.Join(f.BackendPorts, ",") != "3000,8080" {
		t.Errorf("ports %v", f.BackendPorts)
	}
	if len(f.MissingFirebase) != 2 {
		t.Errorf("firebase %v", f.MissingFirebase)
	}
	if len(f.Platforms) != 1 || f.Platforms[0] != "android" {
		t.Errorf("platforms %v", f.Platforms)
	}
	// Installed Flutter 3.10 is older than the 3.19 the app wants; Dart 3.0 older than 3.3.
	if len(f.Warnings) != 2 {
		t.Errorf("warnings %v", f.Warnings)
	}
}

func TestCodeGen(t *testing.T) {
	if got := CodeGen(app); len(got) != 0 {
		t.Errorf("plain app: %v", got)
	}
	pub := app + `freezed_annotation: ^2.4.0
  riverpod_generator: ^2.4.0
dev_dependencies:
  build_runner: ^2.4.0
  freezed: ^2.5.0
  json_serializable: ^6.8.0\n
`
	got := strings.Join(CodeGen(pub), ",")
	if got != "build_runner,freezed,json_serializable,riverpod_generator" {
		t.Errorf("got %q", got)
	}
}

func TestCodeGenStep(t *testing.T) {
	d := t.TempDir()
	write(t, d, "pubspec.yaml", app+`dev_dependencies:
  build_runner: ^2.4.0
`)
	p := &project.Project{Dir: d}
	a := Adapter{}
	a.Analyze(p, d)
	steps := a.Install(p, project.Options{})
	if len(steps) != 2 || steps[1].Cmd != "dart run build_runner build" {
		t.Fatalf("steps %+v", steps)
	}
	if len(a.Install(p, project.Options{NoInstall: true})) != 0 {
		t.Error("NoInstall ignored")
	}
	if !strings.Contains(a.Guide(p), "dart run build_runner build") {
		t.Error("guide does not mention the build step")
	}
}

func TestMissingFlutterWarns(t *testing.T) {
	d := t.TempDir()
	write(t, d, "pubspec.yaml", app)
	f := Analyze(d, d, "", "")
	if len(f.Warnings) == 0 || !strings.Contains(f.Warnings[0], "not found in PATH") {
		t.Errorf("warnings %v", f.Warnings)
	}
}

func TestEnvAssetWithoutFileWarns(t *testing.T) {
	d := t.TempDir()
	write(t, d, "pubspec.yaml", app+"flutter:\n  assets:\n    - .env\n")
	f := Analyze(d, d, "3.24.0", "3.5.0")
	if len(f.Warnings) != 1 || !strings.Contains(f.Warnings[0], ".env") {
		t.Errorf("warnings %v", f.Warnings)
	}
}

func TestAdapterPlan(t *testing.T) {
	d := t.TempDir()
	write(t, d, "pubspec.yaml", app)
	write(t, d, ".env.example", "API_KEY=\n")
	p := &project.Project{Dir: d}
	a := Adapter{}
	a.Analyze(p, d)
	if a.Units(p) != 1 {
		t.Fatal("not found")
	}
	if len(a.Prepare(p, project.Options{})) != 1 || len(a.Install(p, project.Options{})) != 1 {
		t.Error("steps")
	}
	if len(a.Install(p, project.Options{NoInstall: true})) != 0 {
		t.Error("NoInstall ignored")
	}
	r := a.Runnables(p, false)
	if len(r) != 1 || len(r[0].Start) != 0 || r[0].NoStart == "" || len(r[0].Build) == 0 {
		t.Errorf("runnables %+v", r)
	}
	if rr := a.Runnables(p, true); len(rr[0].Build) != 0 {
		t.Error("noBuild ignored")
	}
}
