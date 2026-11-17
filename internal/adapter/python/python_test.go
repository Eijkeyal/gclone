package python

import (
	"os"
	"path/filepath"
	"strings"
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

func TestOlder(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{{"3.9", "3.10", true}, {"3.12", "3.10", false}, {"3.10", "3.10", false}, {"2.7", "3.8", true}}
	for _, c := range cases {
		if got := Older(c.a, c.b); got != c.want {
			t.Errorf("Older(%s,%s)=%v", c.a, c.b, got)
		}
	}
}

func TestFlask(t *testing.T) {
	d := t.TempDir()
	write(t, d, "requirements.txt", "flask>=3\n")
	write(t, d, "app.py", "from flask import Flask\napp = Flask(__name__)\n")
	y := Analyze(d, d, "3.12")
	if y.Framework != "Flask" || y.Manager != "pip" || y.Port != "5000" {
		t.Fatalf("got %+v", y)
	}
	if got := strings.Join(y.Start, " "); !strings.Contains(got, "flask") || !strings.Contains(got, "app:app") {
		t.Errorf("start = %q", got)
	}
}

func TestDjango(t *testing.T) {
	d := t.TempDir()
	write(t, d, "requirements.txt", "Django==5.0\n")
	write(t, d, "manage.py", "")
	y := Analyze(d, d, "3.12")
	if y.Framework != "Django" || y.Port != "8000" || strings.Join(y.Start, " ") != "manage.py runserver" {
		t.Fatalf("got %+v", y)
	}
}

func TestFastAPIAndStreamlit(t *testing.T) {
	d := t.TempDir()
	write(t, d, "requirements.txt", "fastapi\nuvicorn\n")
	write(t, d, "main.py", "from fastapi import FastAPI\napp = FastAPI()\n")
	if y := Analyze(d, d, "3.12"); y.Framework != "FastAPI" || y.Port != "8000" {
		t.Fatalf("fastapi: %+v", y)
	}
	s := t.TempDir()
	write(t, s, "requirements.txt", "streamlit\n")
	write(t, s, "app.py", "import streamlit as st\n")
	if y := Analyze(s, s, "3.12"); y.Framework != "Streamlit" || y.Port != "8501" {
		t.Fatalf("streamlit: %+v", y)
	}
}

func TestManagerChoice(t *testing.T) {
	for file, want := range map[string]string{"poetry.lock": "poetry", "uv.lock": "uv", "Pipfile": "pipenv"} {
		d := t.TempDir()
		write(t, d, file, "")
		write(t, d, "pyproject.toml", "[project]\nname=\"x\"\n")
		if got := Manager(d, "[project]\nname=\"x\"\n"); got != want {
			t.Errorf("%s: got %s want %s", file, got, want)
		}
	}
}

func TestVersionWarning(t *testing.T) {
	d := t.TempDir()
	write(t, d, "pyproject.toml", "[project]\nname=\"x\"\nrequires-python = \">=3.12\"\n")
	y := Analyze(d, d, "3.9")
	if len(y.Warnings) == 0 {
		t.Errorf("expected a warning for Python 3.9 against >=3.12")
	}
}

func TestFindProjectsSkipsVenv(t *testing.T) {
	d := t.TempDir()
	write(t, d, "requirements.txt", "flask\n")
	write(t, d, ".venv/lib/site-packages/x/requirements.txt", "junk\n")
	if got := FindProjects(d); len(got) != 1 {
		t.Errorf("got %v", got)
	}
}
