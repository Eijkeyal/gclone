package doctor

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rauni5/gclone/internal/adapter/python"
	"github.com/rauni5/gclone/internal/adapters"
	"github.com/rauni5/gclone/internal/docker"
	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/project"
	"github.com/rauni5/gclone/internal/services"
)

// Check is one line of the report. Status is pass, warn or fail.
type Check struct {
	Group  string
	Name   string
	Status string
	Detail string
}

// Env is everything doctor asks of the machine. Tests replace it.
type Env struct {
	Have        func(bin string) bool                   // is the program in PATH
	Version     func(bin string, args ...string) string // first line of a version command, "" when unknown
	DockerState func() string
	PortFree    func(port int) bool
	Reachable   func(port int) bool
	FreeBytes   func(dir string) (uint64, error)
	VenvWorks   func() bool // can the interpreter make a virtual environment
}

// DefaultEnv looks at the real machine.
func DefaultEnv() Env {
	return Env{
		Have: func(bin string) bool { _, err := exec.LookPath(bin); return err == nil },
		Version: func(bin string, args ...string) string {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			out, _ := exec.CommandContext(ctx, bin, args...).CombinedOutput()
			return firstLine(string(out))
		},
		DockerState: docker.State,
		PortFree: func(port int) bool {
			l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
			if err != nil {
				return false
			}
			l.Close()
			return true
		},
		Reachable: services.Reachable,
		FreeBytes: fsx.FreeBytes,
		VenvWorks: func() bool {
			cmd, _ := python.Interpreter()
			if len(cmd) == 0 {
				return false
			}
			args := append(append([]string{}, cmd[1:]...), "-c", "import venv, ensurepip")
			return exec.Command(cmd[0], args...).Run() == nil
		},
	}
}

// firstLine is the first line that says something about the version. Java prints a
// "Picked up JAVA_TOOL_OPTIONS" notice first when that variable is set.
func firstLine(out string) string {
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "Picked up ") {
			continue
		}
		if len(l) > 100 {
			l = l[:100] + "..."
		}
		return l
	}
	return ""
}

const (
	pass = "pass"
	warn = "warn"
	fail = "fail"

	warnBytes = 2 << 30   // 2 GiB
	failBytes = 500 << 20 // 500 MiB
)

// Run checks the project p, which AnalyzeProject has already read.
func Run(p *project.Project, env Env) []Check {
	var out []Check
	add := func(group, name, status, detail string) {
		out = append(out, Check{group, name, status, detail})
	}

	// Tools
	tool := func(bin, why string, versionArgs ...string) {
		if !env.Have(bin) {
			add("Tools", bin, fail, "not found in PATH ("+why+")")
			return
		}
		add("Tools", bin, pass, env.Version(bin, versionArgs...))
	}
	tool("git", "gclone and most projects need it", "--version")
	if len(p.Pkgs) > 0 {
		tool("node", "Node.js projects", "--version")
		seen := map[string]bool{}
		for _, pkg := range p.Pkgs {
			pm := pkg.PM
			if pm == "" {
				pm = "npm"
			}
			if !seen[pm] {
				seen[pm] = true
				tool(pm, "the package manager for "+pkg.Rel, "--version")
			}
		}
	}
	if len(p.GoModules) > 0 {
		tool("go", "Go modules", "version")
	}
	if len(p.JavaProjects) > 0 {
		tool("java", "Java projects", "-version")
		for _, j := range p.JavaProjects {
			if !j.Wrapper {
				tool(j.Build, "no wrapper script in "+j.Rel, "--version")
			}
		}
	}
	if len(p.PyProjects) > 0 {
		cmd, _ := python.Interpreter()
		if len(cmd) == 0 {
			add("Tools", "python", fail, "Python 3 not found in PATH (Python projects)")
		} else {
			add("Tools", "python", pass, env.Version(cmd[0], append(append([]string{}, cmd[1:]...), "--version")...))
		}
		seen := map[string]bool{}
		for _, y := range p.PyProjects {
			switch {
			case y.Manager == "pip" && !seen["venv"]:
				seen["venv"] = true
				if len(cmd) > 0 && !env.VenvWorks() {
					add("Tools", "python venv", fail, "this Python cannot create virtual environments (Debian and Ubuntu: sudo apt install python3-venv)")
				}
			case y.Manager != "pip" && !seen[y.Manager]:
				seen[y.Manager] = true
				tool(y.Manager, "the package manager for "+y.Rel, "--version")
			}
		}
	}
	if len(p.FlutterProjects) > 0 {
		tool("flutter", "Flutter apps", "--version")
	}

	// What the project's own files ask for (versions, missing files, shared ports)
	for _, w := range collectWarnings(p) {
		// A missing tool already has its own line under Tools.
		if strings.Contains(w, "not found in PATH") {
			continue
		}
		add("Project", "warning", warn, w)
	}

	// Services
	needsDocker := len(p.Composes) > 0
	var local []*project.Need
	for _, n := range p.Needs {
		if !n.Remote {
			local = append(local, n)
		}
	}
	if len(local) > 0 || needsDocker {
		switch st := env.DockerState(); st {
		case "ready":
			add("Services", "docker", pass, "running")
		case "stopped":
			add("Services", "docker", warn, "installed but not running, gclone will ask to start it")
		case "denied":
			add("Services", "docker", warn, "running, but this user may not use it")
		default:
			add("Services", "docker", warn, "not installed, gclone will ask about installing it")
		}
	}
	for _, n := range local {
		name := n.Name
		if name == "" {
			name = n.Kind
		}
		if n.Port > 0 && env.Reachable(n.Port) {
			add("Services", name, pass, fmt.Sprintf("already answering on port %d", n.Port))
		} else {
			add("Services", name, warn, fmt.Sprintf("not running on port %d, gclone can start it when you approve", n.Port))
		}
	}

	// Ports the apps want
	ports := map[int][]string{}
	for _, a := range adapters.All() {
		for _, r := range a.Runnables(p, true) {
			if r.Port > 0 {
				ports[r.Port] = append(ports[r.Port], r.Label)
			}
		}
	}
	var nums []int
	for n := range ports {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		label := strings.Join(ports[n], ", ")
		if env.PortFree(n) {
			add("Ports", strconv.Itoa(n), pass, "free ("+label+")")
		} else {
			add("Ports", strconv.Itoa(n), warn, "already in use, "+label+" will not be able to start here without changing its port")
		}
	}

	// Disk
	if free, err := env.FreeBytes(p.Dir); err == nil {
		gb := float64(free) / (1 << 30)
		switch {
		case free < failBytes:
			add("Disk", "free space", fail, fmt.Sprintf("%.1f GB free, dependencies will not fit", gb))
		case free < warnBytes:
			add("Disk", "free space", warn, fmt.Sprintf("%.1f GB free, installing dependencies may fill the disk", gb))
		default:
			add("Disk", "free space", pass, fmt.Sprintf("%.1f GB free", gb))
		}
	}
	return out
}

func collectWarnings(p *project.Project) []string {
	out := append([]string{}, p.Warnings...)
	for _, a := range adapters.All() {
		out = append(out, a.Warnings(p)...)
	}
	return out
}

// Verdict summarises the checks.
func Verdict(cs []Check) string {
	v := "READY"
	for _, c := range cs {
		switch c.Status {
		case fail:
			return "NOT READY"
		case warn:
			v = "READY WITH WARNINGS"
		}
	}
	return v
}
