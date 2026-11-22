package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rauni5/gclone/internal/analyze"
	"github.com/rauni5/gclone/internal/buildinfo"
	"github.com/rauni5/gclone/internal/docker"
	"github.com/rauni5/gclone/internal/doctor"
	"github.com/rauni5/gclone/internal/explain"
	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/guide"
	"github.com/rauni5/gclone/internal/plan"
	"github.com/rauni5/gclone/internal/proc"
	"github.com/rauni5/gclone/internal/project"
	"github.com/rauni5/gclone/internal/repos"
	"github.com/rauni5/gclone/internal/services"
	"github.com/rauni5/gclone/internal/state"
	"github.com/rauni5/gclone/internal/ui"
	"github.com/rauni5/gclone/internal/verify"
)

const usage = `gclone - git clone, plus environment setup and a run guide

Usage:
  gclone <repo-url | user/repo | local-path> [directory] [options]
  gclone setup [directory] [options]     set up a repo that is already cloned
  gclone verify [directory] [options]    build it, start it, check it answers; exit 1 if not ready
  gclone status [directory]              show what gclone did here and what is out of date
  gclone doctor [directory]              check, without changing anything, that this machine can run it
  gclone export devcontainer [directory] write .devcontainer/ for Codespaces and VS Code Dev Containers

Options:
  -n, --dry-run         Clone and analyze, show the plan, run nothing else
  -y, --yes             Do not ask for confirmation
      --ignore-scripts  Pass --ignore-scripts to the package manager
      --no-install      Skip installing dependencies
      --no-services     Do not start docker compose database services
      --install-services  Approve the first available way to start a missing
                        database without asking (default is to ask, skip on Enter)
      --clone-deps      Clone and set up other repositories this project needs
                        (for example a backend) without asking
      --no-deps         Do not look for other repositories this project needs
      --verify          Run the verification after setup without asking
      --no-build        Skip the build step when verifying
      --resume          With setup: skip install steps that already succeeded at this commit
      --ci              For scripts and CI: never ask or read the terminal (questions are
                        answered no), implies --yes, and exit 1 when a step or the
                        verification fails
      --json            Print one JSON result on standard output; messages go to standard
                        error. With status: print the recorded state as JSON
      --run-project-setup  Also run the setup command the repo ships (make setup, bin/setup,
                        a justfile recipe...) as the last step. It can run anything, so
                        it is never run by default, not even with --yes or --ci
      --print           With export: print the files instead of writing them
      --timeout=SECONDS How long verify waits for the app to listen (default 60)
  -v, --version         Print the version
  -h, --help            Print this help

Examples:
  gclone https://github.com/someone/app.git
  gclone someone/app my-folder --yes
  gclone someone/app --dry-run
  gclone setup ./app
`

func parseArgs(args []string) (project.Options, error) {
	var o project.Options
	var pos []string
	for _, a := range args {
		switch a {
		case "-n", "--dry-run":
			o.DryRun = true
		case "-y", "--yes":
			o.Yes = true
		case "--ignore-scripts":
			o.IgnoreScripts = true
		case "--no-install":
			o.NoInstall = true
		case "--no-services":
			o.NoServices = true
		case "--install-services":
			o.InstallServices = true
		case "--clone-deps":
			o.CloneCompanions = true
		case "--no-deps":
			o.NoCompanions = true
		case "--verify":
			o.Verify = true
		case "--no-build":
			o.NoBuild = true
		case "--resume":
			o.Resume = true
		case "--ci":
			o.CI = true
			o.Yes = true
		case "--json":
			o.JSON = true
		case "--print":
			o.Print = true
		case "--run-project-setup":
			o.RunProjectSetup = true
		case "-v", "--version":
			fmt.Println("gclone", buildinfo.Version)
			os.Exit(0)
		case "-h", "--help":
			fmt.Print(usage)
			os.Exit(0)
		default:
			if strings.HasPrefix(a, "--timeout=") {
				n, err := strconv.Atoi(strings.TrimPrefix(a, "--timeout="))
				if err != nil || n <= 0 {
					return o, fmt.Errorf("--timeout needs a number of seconds, for example --timeout=90")
				}
				o.Timeout = n
				continue
			}
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				return o, fmt.Errorf("unknown option %q", a)
			}
			pos = append(pos, a)
		}
	}
	if len(pos) > 0 && pos[0] == "export" {
		if len(pos) < 2 || pos[1] != "devcontainer" {
			return o, fmt.Errorf("export needs a target: gclone export devcontainer [directory]")
		}
		if len(pos) > 3 {
			return o, fmt.Errorf("export devcontainer takes at most one directory")
		}
		o.ExportCmd, o.Dir = true, "."
		if len(pos) == 3 {
			o.Dir = pos[2]
		}
		return o, nil
	}
	if len(pos) > 0 && (pos[0] == "setup" || pos[0] == "verify" || pos[0] == "status" || pos[0] == "doctor") {
		o.Setup = pos[0] == "setup"
		o.VerifyCmd = pos[0] == "verify"
		o.StatusCmd = pos[0] == "status"
		o.DoctorCmd = pos[0] == "doctor"
		o.Dir = "."
		if len(pos) == 2 {
			o.Dir = pos[1]
		} else if len(pos) > 2 {
			return o, fmt.Errorf("%s takes at most one directory", pos[0])
		}
		return o, nil
	}
	if len(pos) == 0 || len(pos) > 2 {
		return o, fmt.Errorf("expected a repository and an optional directory")
	}
	o.Url = pos[0]
	if len(pos) == 2 {
		o.Dir = pos[1]
	}
	return o, nil
}

func Main() {
	o, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "gclone:", err)
		fmt.Fprint(os.Stderr, "\n"+usage)
		os.Exit(2)
	}
	if o.CI {
		ui.NonInteractive = true
		proc.NoStdin = true
	}
	if o.StatusCmd && o.JSON {
		abs, err := filepath.Abs(o.Dir)
		if err == nil {
			var ok bool
			if ok, err = statusJSON(os.Stdout, abs); err == nil && !ok {
				os.Exit(1)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "gclone:", err)
			os.Exit(1)
		}
		return
	}
	if o.DoctorCmd {
		ok, err := runDoctor(os.Stdout, o.Dir, o.JSON, doctor.DefaultEnv())
		if err != nil {
			fmt.Fprintln(os.Stderr, "gclone:", err)
			os.Exit(1)
		}
		if !ok {
			os.Exit(1)
		}
		return
	}
	if o.ExportCmd {
		if err := runExportDevcontainer(os.Stdout, os.Stderr, o); err != nil {
			fmt.Fprintln(os.Stderr, "gclone:", err)
			os.Exit(1)
		}
		return
	}
	// With --json the result is the only thing on standard output. Everything else the
	// program prints goes to standard error.
	result := os.Stdout
	if o.JSON {
		os.Stdout = os.Stderr
	}
	dir, p, err := run(o)
	if o.JSON {
		if werr := WriteResult(result, BuildResult(dir, p, err)); werr != nil {
			fmt.Fprintln(os.Stderr, "gclone: could not write the result:", werr)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gclone:", err)
		os.Exit(1)
	}
}

// run does what the command line asked. It returns the folder it worked in and the project
// it analyzed, for --json, which are empty when it stopped early.
func run(o project.Options) (string, *project.Project, error) {
	if o.StatusCmd {
		abs, err := filepath.Abs(o.Dir)
		if err != nil {
			return "", nil, err
		}
		return abs, nil, state.Print(os.Stdout, abs)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return "", nil, fmt.Errorf("git was not found in PATH")
	}

	var abs string
	var err error
	if o.Setup || o.VerifyCmd {
		abs, err = filepath.Abs(o.Dir)
		if err != nil {
			return "", nil, err
		}
		if info, e := os.Stat(abs); e != nil || !info.IsDir() {
			return abs, nil, fmt.Errorf("%s is not a directory", o.Dir)
		}
		if o.VerifyCmd {
			p, err := verify.VerifyCommand(abs, o)
			return abs, p, err
		}
	} else {
		url := repos.ExpandURL(o.Url)
		dir := o.Dir
		if dir == "" {
			dir = repos.RepoName(url)
		}
		if abs, err = filepath.Abs(dir); err != nil {
			return "", nil, err
		}
		fmt.Printf("Cloning %s\n", url)
		if err := proc.RunStream("", true, "git", "clone", url, dir); err != nil {
			if h, ok := explain.FromError(err); ok {
				explain.Print(h)
			}
			return abs, nil, fmt.Errorf("git clone failed: %w", err)
		}
	}

	p, err := setupRepo(abs, o, 0)
	if err == nil && o.CI && p != nil {
		err = ciFailure(p)
	}
	return abs, p, err
}

// ciFailure is the error --ci turns a failed step or verification into, so the exit code is
// 1. Without --ci a failed step is reported in the guide and the exit code stays 0.
func ciFailure(p *project.Project) error {
	if n := failedSteps(p); n > 0 {
		return fmt.Errorf("%d setup step(s) failed, see above", n)
	}
	if p.Verify != nil && verifyBad(p.Verify.Verdict) {
		return fmt.Errorf("verification: %s", p.Verify.Verdict)
	}
	return nil
}

// setupRepo analyzes, sets up and documents one repository. depth is 0 for the
// repository the user asked for and 1 for a companion repository; companions
// of companions are never followed.
func setupRepo(abs string, o project.Options, depth int) (*project.Project, error) {
	var subWarn []string
	if fsx.FileExists(filepath.Join(abs, ".gitmodules")) {
		subWarn = repos.InitSubmodules(abs, o)
	}

	p := analyze.AnalyzeProject(abs)
	p.Warnings = append(p.Warnings, subWarn...)
	p.URL = proc.GitOutput(abs, "remote", "get-url", "origin")
	if p.URL == "" {
		p.URL = "local repository"
	}
	p.Commit = proc.GitOutput(abs, "rev-parse", "--short", "HEAD")
	if p.Commit == "" {
		p.Commit = "unknown"
	}
	if depth == 0 && !o.NoCompanions {
		p.Companions = repos.DetectCompanions(p)
	}

	needsDocker := len(p.Needs) > 0
	for _, c := range p.Composes {
		if len(c.DBServices) > 0 {
			needsDocker = true
		}
	}
	if needsDocker {
		p.DockerState = docker.State()
	}

	steps := plan.BuildPlan(p, o)
	if o.Resume {
		prev, _ := state.Load(abs)
		var skipped []project.Step
		var why string
		steps, skipped, why = state.Resume(prev, p.Commit, steps)
		if why != "" {
			fmt.Println("\nResume: running everything, because " + why + ".")
		}
		for _, s := range skipped {
			fmt.Printf("\n[skip] %s (in %s): already done in an earlier run\n", s.Name, s.Where)
		}
		p.Steps = append(p.Steps, skipped...)
	}
	plan.PrintPlan(p, steps, o)
	services.PrintNeeds(p, o)
	repos.PrintCompanions(p)
	if !p.HasUnits() {
		return p, nil
	}
	if o.DryRun {
		fmt.Println("\nDry run: nothing was installed or changed.")
		return p, nil
	}
	if len(steps) > 0 && !o.Yes && !ui.AskYesNo("\nRun this plan?", true) {
		fmt.Println("Skipped. Run `gclone setup " + filepath.Base(abs) + "` later to apply it.")
		return p, nil
	}

	plan.ExecutePlan(p, steps)
	p.Executed = true
	services.ProvisionServices(p, o)
	if depth == 0 {
		repos.HandleCompanions(p, o, setupRepo)
		if o.Verify || ui.AskYesNo("\nVerify now? This builds the project, starts it for a few seconds and checks that it answers.", false) {
			p.Verify = verify.RunVerify(p, o)
		}
	}
	if err := state.Save(abs, state.FromProject(p)); err != nil {
		fmt.Printf("[warn] could not write %s: %v\n", state.Path(abs), err)
	}
	path, err := guide.WriteGuide(p)
	if err != nil {
		return p, fmt.Errorf("could not write the guide: %w", err)
	}
	fmt.Printf("\nGuide written to %s\n", path)
	plan.PrintNext(p)
	return p, nil
}
