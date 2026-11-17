package fsx

import (
	"os"
	"path/filepath"
	"strings"
)

var SkipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true, ".next": true, "vendor": true,
}

func IsInside(parent, child string) bool {
	r, err := filepath.Rel(parent, child)
	return err == nil && r != "." && !strings.HasPrefix(r, "..")
}

func FileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func Rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(r)
}

func CopyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o600)
}

// ExcludeFromGit keeps a generated file or folder out of `git status` by listing it in
// .git/info/exclude, which is local to this clone and never committed.
func ExcludeFromGit(dir, name string) {
	info := filepath.Join(dir, ".git", "info")
	if err := os.MkdirAll(info, 0o755); err != nil {
		return
	}
	path := filepath.Join(info, "exclude")
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), name) {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	if len(b) > 0 && !strings.HasSuffix(string(b), "\n") {
		_, _ = f.WriteString("\n")
	}
	_, _ = f.WriteString(name + "\n")
}
