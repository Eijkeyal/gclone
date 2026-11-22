package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rauni5/gclone/internal/analyze"
	"github.com/rauni5/gclone/internal/devcontainer"
	"github.com/rauni5/gclone/internal/project"
	"github.com/rauni5/gclone/internal/ui"
)

// runExportDevcontainer writes .devcontainer/ for the project in o.Dir. With --print or
// --dry-run it prints the files and writes nothing. It never overwrites a file that exists.
func runExportDevcontainer(out, msg io.Writer, o project.Options) error {
	abs, err := filepath.Abs(o.Dir)
	if err != nil {
		return err
	}
	if info, e := os.Stat(abs); e != nil || !info.IsDir() {
		return errors.New(o.Dir + " is not a directory")
	}
	p := analyze.AnalyzeProject(abs)
	if !p.HasUnits() {
		return errors.New("no supported project found in " + abs + ", nothing to export")
	}
	res, err := devcontainer.Build(p, filepath.Base(abs))
	if err != nil {
		return err
	}
	files := map[string]string{".devcontainer/devcontainer.json": res.JSON}
	if res.Compose != "" {
		files[".devcontainer/docker-compose.yml"] = res.Compose
	}

	if o.Print || o.DryRun {
		for _, name := range res.Files() {
			fmt.Fprintf(out, "# ---- %s ----\n%s\n", name, files[name])
		}
		notes(msg, res.Notes)
		return nil
	}

	for _, name := range res.Files() {
		if _, err := os.Stat(filepath.Join(abs, filepath.FromSlash(name))); err == nil {
			return fmt.Errorf("%s already exists, nothing was changed (use --print to see what gclone would write)", name)
		}
	}
	fmt.Fprintf(msg, "gclone will create in %s:\n", abs)
	for _, name := range res.Files() {
		fmt.Fprintf(msg, "  %s\n", name)
	}
	if !o.Yes && !ui.AskYesNo("Write these files?", true) {
		fmt.Fprintln(msg, "Nothing written.")
		return nil
	}
	for _, name := range res.Files() {
		path := filepath.Join(abs, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(files[name]), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(msg, "[ok]   wrote %s\n", name)
	}
	fmt.Fprintln(msg, "\nThese are project files, not hidden ones: review them, and commit them if you want others to use them.")
	notes(msg, res.Notes)
	return nil
}

func notes(w io.Writer, ns []string) {
	for _, n := range ns {
		fmt.Fprintf(w, "Note: %s\n", n)
	}
}
