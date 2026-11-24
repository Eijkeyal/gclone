// Package devcontainer builds a Dev Container configuration (.devcontainer/devcontainer.json,
// and a docker-compose.yml when the project needs databases) from what gclone detected. It
// only builds text; writing the files, and asking first, is the caller's job.
package devcontainer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rauni5/gclone/internal/adapter/java"
	"github.com/rauni5/gclone/internal/adapter/node"
	"github.com/rauni5/gclone/internal/adapters"
	"github.com/rauni5/gclone/internal/project"
)

// Result is the files to write and the things the reader should know about them.
type Result struct {
	JSON    string   // .devcontainer/devcontainer.json
	Compose string   // .devcontainer/docker-compose.yml, "" when no service is needed
	Notes   []string // limits and things to check, in plain words
}

type config struct {
	Name              string                            `json:"name"`
	Image             string                            `json:"image,omitempty"`
	DockerComposeFile string                            `json:"dockerComposeFile,omitempty"`
	Service           string                            `json:"service,omitempty"`
	WorkspaceFolder   string                            `json:"workspaceFolder,omitempty"`
	Features          map[string]map[string]interface{} `json:"features,omitempty"`
	ForwardPorts      []int                             `json:"forwardPorts,omitempty"`
	PostCreateCommand string                            `json:"postCreateCommand,omitempty"`
	Customizations    *customizations                   `json:"customizations,omitempty"`
}

type customizations struct {
	VSCode struct {
		Extensions []string `json:"extensions"`
	} `json:"vscode"`
}

const baseImage = "mcr.microsoft.com/devcontainers/base:ubuntu"

var majorRe = regexp.MustCompile(`\d+`)

// Build makes the configuration for the project p, which AnalyzeProject has already read.
func Build(p *project.Project, name string) (Result, error) {
	cfg := config{Name: name, Features: map[string]map[string]interface{}{}}
	var res Result
	var cmds []string
	var exts []string
	addExt := func(e string) {
		for _, x := range exts {
			if x == e {
				return
			}
		}
		exts = append(exts, e)
	}
	in := func(rel, cmd string) {
		if rel == "" || rel == "." {
			cmds = append(cmds, cmd)
			return
		}
		cmds = append(cmds, "(cd "+rel+" && "+cmd+")")
	}
	envCopy := func(rel, example string) {
		if example != "" {
			in(rel, "[ -f .env ] || cp "+example+" .env")
		}
	}

	// Node.js
	if len(p.Pkgs) > 0 {
		version := "lts"
		for _, pkg := range p.Pkgs {
			if m := majorRe.FindString(pkg.NodeWanted); m != "" {
				version = m
				break
			}
		}
		cfg.Features["ghcr.io/devcontainers/features/node:1"] = map[string]interface{}{"version": version}
		addExt("dbaeumer.vscode-eslint")
		prep := map[string]bool{}
		for _, pkg := range p.Pkgs {
			switch pkg.PM {
			case "pnpm", "yarn":
				if !prep[pkg.PM] {
					prep[pkg.PM] = true
					cmds = append(cmds, "corepack enable")
				}
			case "bun":
				if !prep["bun"] {
					prep["bun"] = true
					cmds = append(cmds, "npm install -g bun")
				}
			}
		}
		for _, pkg := range p.Pkgs {
			envCopy(pkg.Rel, pkg.EnvExample)
			name, args := node.InstallCmd(pkg, false)
			in(pkg.Rel, name+" "+strings.Join(args, " "))
		}
	}

	// Go
	if len(p.GoModules) > 0 {
		version := "latest"
		for _, m := range p.GoModules {
			if m.GoWanted != "" {
				version = m.GoWanted
				break
			}
		}
		cfg.Features["ghcr.io/devcontainers/features/go:1"] = map[string]interface{}{"version": version}
		addExt("golang.go")
		for _, m := range p.GoModules {
			in(m.Rel, "go mod download")
		}
	}

	// Rust
	if len(p.RustProjects) > 0 {
		version := "latest"
		for _, r := range p.RustProjects {
			if len(r.Toolchain) > 0 && r.Toolchain[0] >= '0' && r.Toolchain[0] <= '9' {
				version = r.Toolchain
				break
			}
		}
		cfg.Features["ghcr.io/devcontainers/features/rust:1"] = map[string]interface{}{"version": version}
		addExt("rust-lang.rust-analyzer")
		for _, r := range p.RustProjects {
			in(r.Rel, "cargo fetch")
		}
	}

	// Java
	if len(p.JavaProjects) > 0 {
		major, maven, gradle := 0, false, false
		for _, j := range p.JavaProjects {
			if j.JavaWanted > major {
				major = j.JavaWanted
			}
			if !j.Wrapper {
				maven = maven || j.Build == "maven"
				gradle = gradle || j.Build == "gradle"
			}
		}
		if major == 0 {
			major = 17
		}
		cfg.Features["ghcr.io/devcontainers/features/java:1"] = map[string]interface{}{
			"version": strconv.Itoa(major), "installMaven": maven, "installGradle": gradle}
		addExt("vscjava.vscode-java-pack")
		for _, j := range p.JavaProjects {
			in(j.Rel, java.Display(j, java.DependencyGoals(j)...))
		}
	}

	// Python (Linux commands, whatever system gclone runs on)
	if len(p.PyProjects) > 0 {
		version := "latest"
		for _, y := range p.PyProjects {
			if y.PyWanted != "" {
				version = y.PyWanted
				break
			}
		}
		cfg.Features["ghcr.io/devcontainers/features/python:1"] = map[string]interface{}{"version": version}
		addExt("ms-python.python")
		tools := map[string]bool{}
		for _, y := range p.PyProjects {
			envCopy(y.Rel, y.EnvExample)
			switch y.Manager {
			case "poetry", "uv", "pipenv":
				if !tools[y.Manager] {
					tools[y.Manager] = true
					cmds = append(cmds, "pip install "+y.Manager)
				}
			}
			switch y.Manager {
			case "poetry":
				in(y.Rel, "poetry install")
			case "uv":
				in(y.Rel, "uv sync")
			case "pipenv":
				in(y.Rel, "pipenv install --dev")
			default:
				install := ".venv/bin/python -m pip install -e ."
				if y.Requirements != "" {
					install = ".venv/bin/python -m pip install -r " + y.Requirements
				}
				in(y.Rel, "python -m venv .venv && "+install)
			}
		}
	}

	// Flutter has no official Dev Container feature.
	if len(p.FlutterProjects) > 0 {
		res.Notes = append(res.Notes, "Flutter is not included: there is no official Dev Container feature for it. Install Flutter in the container yourself, or use the community features.")
	}

	// Ports the apps want
	ports := map[int]bool{}
	for _, a := range adapters.All() {
		for _, r := range a.Runnables(p, true) {
			if r.Port > 0 {
				ports[r.Port] = true
			}
		}
	}
	for n := range ports {
		cfg.ForwardPorts = append(cfg.ForwardPorts, n)
	}
	sort.Ints(cfg.ForwardPorts)

	// Services
	var kinds []string
	seen := map[string]bool{}
	for _, n := range p.Needs {
		if !n.Remote && image(n.Kind) != "" && !seen[n.Kind] {
			seen[n.Kind] = true
			kinds = append(kinds, n.Kind)
		}
	}
	if len(kinds) > 0 {
		res.Compose = compose(kinds)
		cfg.DockerComposeFile = "docker-compose.yml"
		cfg.Service = "app"
		cfg.WorkspaceFolder = "/workspaces/${localWorkspaceFolderBasename}"
		res.Notes = append(res.Notes,
			"The databases ("+strings.Join(kinds, ", ")+") run next to your code and answer on localhost, with the default ports.")
		var logins []string
		for _, k := range kinds {
			switch k {
			case "postgres":
				logins = append(logins, "PostgreSQL user `postgres`, password `postgres`")
			case "mysql":
				logins = append(logins, "MySQL user `root`, password `root`")
			}
		}
		if len(logins) > 0 {
			res.Notes = append(res.Notes, "Development logins: "+strings.Join(logins, "; ")+". Make the variables in your `.env` match them.")
		}
	} else {
		cfg.Image = baseImage
	}

	if len(exts) > 0 {
		cfg.Customizations = &customizations{}
		cfg.Customizations.VSCode.Extensions = exts
	}
	if len(cmds) > 0 {
		cfg.PostCreateCommand = strings.Join(cmds, " && ")
	}
	if len(cfg.Features) == 0 {
		cfg.Features = nil
	}
	res.Notes = append(res.Notes, "Not run against a real Docker engine or Codespaces by gclone. Open it once and check that it builds.")

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false) // keep && readable
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return res, err
	}
	res.JSON = b.String()
	return res, nil
}

// images are the container images for the services gclone knows. They match the ones
// `gclone setup` starts when it runs a missing database in Docker.
var images = map[string]string{
	"mongodb": "mongo:7", "postgres": "postgres:16", "mysql": "mysql:8", "redis": "redis:7", "rabbitmq": "rabbitmq:3",
}

func image(kind string) string { return images[kind] }

// serviceEnv holds the settings an image needs to start at all.
var serviceEnv = map[string][]string{
	"postgres": {"POSTGRES_PASSWORD: postgres"},
	"mysql":    {"MYSQL_ROOT_PASSWORD: root"},
}

var volumePath = map[string]string{
	"postgres": "/var/lib/postgresql/data", "mysql": "/var/lib/mysql",
	"mongodb": "/data/db", "redis": "/data", "rabbitmq": "/var/lib/rabbitmq",
}

// compose writes a docker-compose.yml with the app container and one container per service.
// All of them share the network of the first service, so every database answers on localhost.
func compose(kinds []string) string {
	first := kinds[0]
	var b strings.Builder
	w := func(format string, a ...interface{}) { fmt.Fprintf(&b, format+"\n", a...) }
	w("# Written by gclone. The databases share one network with the app container, so the app")
	w("# reaches each of them on localhost and its usual port.")
	w("services:")
	w("  app:")
	w("    image: %s", baseImage)
	w("    volumes:")
	w("      - ../..:/workspaces:cached")
	w("    command: sleep infinity")
	w("    network_mode: service:%s", first)
	w("    depends_on:")
	for _, k := range kinds {
		w("      - %s", k)
	}
	for i, k := range kinds {
		w("")
		w("  %s:", k)
		w("    image: %s", image(k))
		w("    restart: unless-stopped")
		if i > 0 {
			w("    network_mode: service:%s", first)
		}
		if env := serviceEnv[k]; len(env) > 0 {
			w("    environment:")
			for _, e := range env {
				w("      %s", e)
			}
		}
		w("    volumes:")
		w("      - %s-data:%s", k, volumePath[k])
	}
	w("")
	w("volumes:")
	for _, k := range kinds {
		w("  %s-data:", k)
	}
	return b.String()
}

// Files returns the paths, relative to the project, of the files Build produced.
func (r Result) Files() []string {
	out := []string{filepath.ToSlash(filepath.Join(".devcontainer", "devcontainer.json"))}
	if r.Compose != "" {
		out = append(out, ".devcontainer/docker-compose.yml")
	}
	return out
}
