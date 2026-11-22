package ownsetup

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMakefileSetupTarget(t *testing.T) {
	d := t.TempDir()
	write(t, d, "Makefile", ".PHONY: all setup\nCC := gcc\n\nall:\n\tgo build\n\nsetup: deps\n\t./scripts/prepare.sh\n")
	got := Detect(d)
	if len(got) != 1 || got[0].Shown != "make setup" || got[0].Kind != "make" || got[0].Rel != "." {
		t.Fatalf("%+v", got)
	}
}

func TestInstallIsNotASetupTarget(t *testing.T) {
	d := t.TempDir()
	write(t, d, "Makefile", "install:\n\tcp app /usr/local/bin\n\ninit:\n\techo hi\n")
	if got := Detect(d); len(got) != 0 {
		t.Errorf("install and init must not count: %+v", got)
	}
}

func TestVariableAndIndentedLinesAreNotTargets(t *testing.T) {
	d := t.TempDir()
	write(t, d, "Makefile", "setup := yes\nbuild:\n\techo setup: done\n  setup:\n# setup: commented\n")
	if got := Detect(d); len(got) != 0 {
		t.Errorf("%+v", got)
	}
}

func TestPreferenceOrderAndCustomMakefileName(t *testing.T) {
	d := t.TempDir()
	write(t, d, "GNUmakefile", "bootstrap:\n\ttrue\nsetup:\n\ttrue\n")
	got := Detect(d)
	if len(got) != 1 || got[0].Shown != "make -f GNUmakefile setup" {
		t.Errorf("%+v", got)
	}
}

func TestJustRecipeWithParameters(t *testing.T) {
	d := t.TempDir()
	write(t, d, "justfile", "default:\n  just --list\n\nsetup target='dev':\n  echo hi\n")
	got := Detect(d)
	if len(got) != 1 || got[0].Shown != "just setup" || got[0].Kind != "just" {
		t.Errorf("%+v", got)
	}
}

func TestScripts(t *testing.T) {
	d := t.TempDir()
	write(t, d, "bin/setup", "#!/usr/bin/env ruby\n")
	write(t, d, "api/scripts/bootstrap.sh", "#!/bin/sh\n")
	got := Detect(d)
	if len(got) != 2 || got[0].Shown != "./bin/setup" || got[1].Shown != "sh scripts/bootstrap.sh" || got[1].Rel != "api" {
		t.Fatalf("%+v", got)
	}
}

func TestSkipsVendoredFolders(t *testing.T) {
	d := t.TempDir()
	write(t, d, "node_modules/pkg/Makefile", "setup:\n\ttrue\n")
	write(t, d, ".git/Makefile", "setup:\n\ttrue\n")
	if got := Detect(d); len(got) != 0 {
		t.Errorf("%+v", got)
	}
}
