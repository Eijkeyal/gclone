// Package ownsetup finds the setup commands a repository ships itself: a `setup` target in a
// Makefile or justfile, or a script such as bin/setup or scripts/bootstrap.sh. gclone never
// runs them by default, because they can run anything. It lists them, and runs one only when
// asked (--run-project-setup) and approved.
package ownsetup

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/project"
)

// Names are the targets and recipes that mean "get this project ready". `install` and `init`
// are left out on purpose: in a Makefile `install` usually copies a program into the system.
var names = []string{"setup", "bootstrap", "dev-setup", "setup-dev"}

var scripts = []string{
	"bin/setup", "script/setup", "script/bootstrap",
	"scripts/setup.sh", "scripts/setup", "scripts/bootstrap.sh", "scripts/bootstrap", "scripts/dev-setup.sh",
}

var makefiles = []string{"Makefile", "makefile", "GNUmakefile"}
var justfiles = []string{"justfile", "Justfile", ".justfile"}

// A target or recipe line: a name, optional parameters (just), then a colon that is not the
// start of ":=" (a variable).
var ruleRe = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.-]*)[^:=\n]*:($|[^=])`)

// A just recipe may have parameters with defaults ("setup target='dev':"), so "=" is allowed
// between the name and the colon.
var recipeRe = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_-]*)(\s+[^:\n]*)?:($|[^=])`)

func definedNames(path string, re *regexp.Regexp) map[string]bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "#") {
			continue
		}
		if m := re.FindStringSubmatch(line); m != nil {
			out[m[1]] = true
		}
	}
	return out
}

func first(have map[string]bool) string {
	for _, n := range names {
		if have[n] {
			return n
		}
	}
	return ""
}

// Detect looks in the repository root and the folders directly under it.
func Detect(root string) []*project.OwnSetup {
	dirs := []string{root}
	if entries, err := os.ReadDir(root); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !fsx.SkipDirs[e.Name()] {
				dirs = append(dirs, filepath.Join(root, e.Name()))
			}
		}
	}
	var out []*project.OwnSetup
	for _, d := range dirs {
		rel := fsx.Rel(root, d)
		for _, f := range makefiles {
			if t := first(definedNames(filepath.Join(d, f), ruleRe)); t != "" {
				argv := []string{"make", t}
				if f != "Makefile" {
					argv = []string{"make", "-f", f, t}
				}
				out = append(out, &project.OwnSetup{Dir: d, Rel: rel, Kind: "make", Argv: argv, Shown: strings.Join(argv, " ")})
				break
			}
		}
		for _, f := range justfiles {
			if t := first(definedNames(filepath.Join(d, f), recipeRe)); t != "" {
				out = append(out, &project.OwnSetup{Dir: d, Rel: rel, Kind: "just", Argv: []string{"just", t}, Shown: "just " + t})
				break
			}
		}
		for _, s := range scripts {
			if !fsx.FileExists(filepath.Join(d, filepath.FromSlash(s))) {
				continue
			}
			argv := []string{"./" + s}
			if strings.HasSuffix(s, ".sh") {
				argv = []string{"sh", s}
			}
			out = append(out, &project.OwnSetup{Dir: d, Rel: rel, Kind: "script", Argv: argv, Shown: strings.Join(argv, " ")})
			break
		}
	}
	return out
}
