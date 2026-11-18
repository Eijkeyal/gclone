package flutter

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rauni5/gclone/internal/adapter"
	"github.com/rauni5/gclone/internal/envfile"
	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/proc"
	"github.com/rauni5/gclone/internal/project"
)

// Adapter is the Flutter ecosystem.
type Adapter struct{}

var _ adapter.Adapter = Adapter{}

func (Adapter) Name() string                 { return "Flutter" }
func (Adapter) Units(p *project.Project) int { return len(p.FlutterProjects) }
func (Adapter) Markers() []string            { return []string{marker} }

func (Adapter) Analyze(p *project.Project, root string) {
	dirs := FindProjects(root)
	if len(dirs) == 0 {
		return
	}
	fv, dv := Installed()
	for _, d := range dirs {
		p.FlutterProjects = append(p.FlutterProjects, Analyze(root, d, fv, dv))
	}
}

func (Adapter) Warnings(p *project.Project) []string {
	var out []string
	for _, f := range p.FlutterProjects {
		for _, w := range f.Warnings {
			out = append(out, f.Rel+": "+w)
		}
	}
	return out
}

func (Adapter) Detected(p *project.Project) []string {
	var out []string
	for _, f := range p.FlutterProjects {
		line := fmt.Sprintf("%s: %s, Flutter", f.Rel, f.Name)
		if f.FlutterWanted != "" {
			line += " " + f.FlutterWanted
		} else if f.DartWanted != "" {
			line += ", Dart " + f.DartWanted
		}
		if len(f.Platforms) > 0 {
			line += " (" + strings.Join(f.Platforms, ", ") + ")"
		}
		if f.Firebase {
			line += ", Firebase"
		}
		out = append(out, line)
	}
	return out
}

func (Adapter) Prepare(p *project.Project, o project.Options) []project.Planned {
	var steps []project.Planned
	for _, f := range p.FlutterProjects {
		f := f
		if f.EnvExample == "" || fsx.FileExists(filepath.Join(f.Dir, ".env")) {
			continue
		}
		src, dst := filepath.Join(f.Dir, f.EnvExample), filepath.Join(f.Dir, ".env")
		steps = append(steps, project.Planned{
			Where: f.Rel,
			Name:  "Create .env from " + f.EnvExample,
			Cmd:   "copy " + f.EnvExample + " .env",
			Run: func() error {
				if err := fsx.CopyFile(src, dst); err != nil {
					return err
				}
				f.EnvCreated = true
				return nil
			},
		})
	}
	return steps
}

func (Adapter) Install(p *project.Project, o project.Options) []project.Planned {
	if o.NoInstall {
		return nil
	}
	var steps []project.Planned
	for _, f := range p.FlutterProjects {
		f := f
		steps = append(steps, project.Planned{
			Where:     f.Rel,
			Name:      "Install dependencies (flutter pub get)",
			Cmd:       "flutter pub get",
			Resumable: true,
			Run: func() error {
				if _, err := exec.LookPath("flutter"); err != nil {
					return fmt.Errorf("flutter was not found in PATH, install it from https://docs.flutter.dev/get-started/install")
				}
				return proc.RunStream(f.Dir, false, "flutter", "pub", "get")
			},
		})
	}
	return steps
}

func (Adapter) Runnables(p *project.Project, noBuild bool) []project.Runnable {
	var out []project.Runnable
	for _, f := range p.FlutterProjects {
		r := project.Runnable{
			Label:       f.Rel + " (Flutter)",
			Dir:         f.Dir,
			ExampleName: f.EnvExample,
			NoStart:     "a Flutter app runs on a device, an emulator or in a browser; list them with `flutter devices` and start it with `flutter run -d <id>`",
		}
		if !noBuild {
			r.Build = AnalyzeArgs
		}
		envPath := filepath.Join(f.Dir, ".env")
		if fsx.FileExists(envPath) {
			r.HasEnv = true
			for _, v := range envfile.ParseEnvFile(envPath) {
				if v.Kind == "blocking" {
					r.Blocking = append(r.Blocking, v.Key)
				}
			}
		}
		out = append(out, r)
	}
	return out
}

func prefix(rel string) string {
	if rel == "." {
		return ""
	}
	return "cd " + rel + " && "
}

func (Adapter) Next(p *project.Project) []string {
	var out []string
	for _, f := range p.FlutterProjects {
		out = append(out, prefix(f.Rel)+"flutter run")
	}
	return out
}

func (Adapter) Guide(p *project.Project) string {
	var b strings.Builder
	w := func(format string, a ...interface{}) { fmt.Fprintf(&b, format+"\n", a...) }
	for _, f := range p.FlutterProjects {
		title := f.Name
		if f.Rel != "." {
			title += " (`" + f.Rel + "`)"
		}
		w("### %s", title)
		w("")
		stack := "Flutter"
		if f.FlutterWanted != "" {
			stack += " " + f.FlutterWanted + " or newer"
		}
		if f.DartWanted != "" {
			stack += ", Dart " + f.DartWanted + " or newer"
		}
		w("Stack: %s", stack)
		if len(f.Platforms) > 0 {
			w("")
			w("Platforms in the project: %s", strings.Join(f.Platforms, ", "))
		}
		w("")
		w("```bash")
		if f.Rel != "." {
			w("cd %s", f.Rel)
		}
		w("flutter doctor              # check the toolchain, once per machine")
		w("flutter pub get             # dependencies")
		w("flutter devices             # emulators, phones and browsers it can run on")
		w("flutter run -d <device-id>  # start the app")
		if contains(f.Platforms, "web") {
			w("flutter run -d chrome       # or in the browser")
		}
		if f.HasTests {
			w("flutter test                # tests")
		}
		w("```")
		w("")
		w("gclone does not start a Flutter app. It runs on a device you choose.")
		w("")
	}
	return b.String()
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (Adapter) GuideConfig(p *project.Project) []string {
	var out []string
	for _, f := range p.FlutterProjects {
		var blocking, review []project.EnvVar
		for _, v := range f.EnvVars {
			switch v.Kind {
			case "blocking":
				blocking = append(blocking, v)
			case "review":
				review = append(review, v)
			}
		}
		if len(blocking) == 0 && len(review) == 0 && len(f.MissingFirebase) == 0 && len(f.BackendPorts) == 0 && !f.EnvCreated {
			continue
		}
		var b strings.Builder
		w := func(format string, a ...interface{}) { fmt.Fprintf(&b, format+"\n", a...) }
		w("### Configuration: %s", project.UnitLabel(f.Rel))
		w("")
		if f.EnvCreated {
			w("`.env` was created from `%s`.", f.EnvExample)
			w("")
		}
		if len(blocking) > 0 {
			w("**Set these first:**")
			w("")
			for _, v := range blocking {
				w("- `%s`: %s", v.Key, v.Reason)
			}
			w("")
		}
		if len(review) > 0 {
			w("**Works on defaults, change for real use:**")
			w("")
			for _, v := range review {
				w("- `%s`: %s", v.Key, v.Reason)
			}
			w("")
		}
		if len(f.MissingFirebase) > 0 {
			w("**Firebase files that are not in the repository:**")
			w("")
			for _, m := range f.MissingFirebase {
				w("- `%s`", m)
			}
			w("")
			w("These are usually kept out of git. Create them with `flutterfire configure`, or download them from your Firebase console. The app may not build or start without them.")
			w("")
		}
		if len(f.BackendPorts) > 0 {
			w("**A backend on this machine:** the code calls `localhost` or `10.0.2.2` (the Android emulator's name for your computer) on port %s. Start that backend first. A real phone cannot reach `localhost`, use your computer's address instead.", strings.Join(f.BackendPorts, ", "))
			w("")
		}
		out = append(out, b.String())
	}
	return out
}
