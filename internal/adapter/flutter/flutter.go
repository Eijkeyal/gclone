// Package flutter is the Flutter adapter. It finds Flutter apps (a pubspec.yaml that depends
// on the Flutter SDK), compares the installed Flutter and Dart with what the app asks for,
// runs `flutter pub get`, and notes the files an app usually needs by hand: a .env asset,
// Firebase configuration and a backend on localhost. Flutter apps run on a device, an
// emulator or in a browser, so gclone never starts one; `verify` runs `flutter analyze`.
package flutter

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rauni5/gclone/internal/envfile"
	"github.com/rauni5/gclone/internal/fsx"
	"github.com/rauni5/gclone/internal/project"
)

const marker = "pubspec.yaml"

var flutterDepRe = regexp.MustCompile(`(?m)^\s+sdk:\s*["']?flutter["']?\s*(#.*)?$`)

// isFlutter reports whether a pubspec depends on the Flutter SDK. A pure Dart package or
// command line tool does not, and is left alone.
func isFlutter(pubspec string) bool { return flutterDepRe.MatchString(pubspec) }

func readPubspec(dir string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(dir, marker))
	if err != nil {
		return "", false
	}
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	return s, isFlutter(s)
}

// FindProjects returns the Flutter apps: the root when it is one, otherwise folders one
// level down.
func FindProjects(root string) []string {
	if _, ok := readPubspec(root); ok {
		return []string{root}
	}
	var dirs []string
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || fsx.SkipDirs[e.Name()] {
			continue
		}
		d := filepath.Join(root, e.Name())
		if _, ok := readPubspec(d); ok {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// ---- versions

var versionRe = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)
var lowerRe = regexp.MustCompile(`(?:>=|\^)\s*(\d+\.\d+(?:\.\d+)?)`)

// Lower returns the lowest version a pubspec constraint allows (">=3.3.0 <4.0.0" or "^3.3.0"),
// or "" when it names none.
func Lower(constraint string) string {
	if m := lowerRe.FindStringSubmatch(constraint); m != nil {
		return m[1]
	}
	return ""
}

func parts(v string) [3]int {
	var out [3]int
	for i, s := range strings.SplitN(versionRe.FindString(v), ".", 3) {
		out[i], _ = strconv.Atoi(s)
	}
	return out
}

// Older reports whether version a is lower than b.
func Older(a, b string) bool {
	x, y := parts(a), parts(b)
	for i := range x {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}

var envBlockRe = regexp.MustCompile(`(?m)^environment:[ \t]*\n((?:[ \t]+.*\n?)+)`)
var envSdkRe = regexp.MustCompile(`(?m)^[ \t]+sdk:\s*["']?([^"'\n#]+)`)
var envFlutterRe = regexp.MustCompile(`(?m)^[ \t]+flutter:\s*["']?([^"'\n#]+)`)

// Wanted reads the Dart and Flutter lower bounds from the environment block of a pubspec.
func Wanted(pubspec string) (dart, flutter string) {
	m := envBlockRe.FindStringSubmatch(pubspec)
	if m == nil {
		return "", ""
	}
	if s := envSdkRe.FindStringSubmatch(m[1]); s != nil {
		dart = Lower(s[1])
	}
	if f := envFlutterRe.FindStringSubmatch(m[1]); f != nil {
		flutter = Lower(f[1])
	}
	return dart, flutter
}

// Installed asks the Flutter tool for its version. Both results are "" when Flutter is not
// installed or does not answer within a short time.
func Installed() (flutter, dart string) {
	if _, err := exec.LookPath("flutter"); err != nil {
		return "", ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "flutter", "--version", "--machine").Output()
	if err != nil {
		return "", ""
	}
	// Flutter may print a download notice before the JSON.
	if i := strings.Index(string(out), "{"); i >= 0 {
		out = out[i:]
	}
	var v struct {
		Framework string `json:"frameworkVersion"`
		Dart      string `json:"dartSdkVersion"`
	}
	if json.Unmarshal(out, &v) != nil {
		return "", ""
	}
	return versionRe.FindString(v.Framework), versionRe.FindString(v.Dart)
}

// ---- reading one app

var nameRe = regexp.MustCompile(`(?m)^name:\s*["']?([\w.-]+)`)
var envAssetRe = regexp.MustCompile(`(?m)^\s*-\s*["']?(?:[\w./-]*/)?\.env["']?\s*$`)
var firebaseDepRe = regexp.MustCompile(`(?m)^\s+firebase_core:`)
var backendRe = regexp.MustCompile(`(?:localhost|127\.0\.0\.1|10\.0\.2\.2):(\d{2,5})`)

// codeGenPackages need `dart run build_runner build` before the app compiles.
var codeGenPackages = []string{"build_runner", "freezed", "json_serializable", "riverpod_generator"}

// CodeGen lists the code generation packages a pubspec names under dependencies or
// dev_dependencies, in a fixed order.
func CodeGen(pubspec string) []string {
	var out []string
	for _, name := range codeGenPackages {
		if regexp.MustCompile(`(?m)^[ 	]+` + name + `:`).MatchString(pubspec) {
			out = append(out, name)
		}
	}
	return out
}

var platformDirs = []string{"android", "ios", "web", "windows", "linux", "macos"}
var envExamples = []string{".env.example", ".env.sample", ".env.template", "example.env"}

// Analyze reads one Flutter app. flutterVer and dartVer are the installed versions or "".
func Analyze(root, dir, flutterVer, dartVer string) *project.FlutterProject {
	pub, _ := readPubspec(dir)
	f := &project.FlutterProject{Dir: dir, Rel: fsx.Rel(root, dir), Name: filepath.Base(dir),
		FlutterInstalled: flutterVer, DartInstalled: dartVer}
	if m := nameRe.FindStringSubmatch(pub); m != nil {
		f.Name = m[1]
	}
	f.DartWanted, f.FlutterWanted = Wanted(pub)
	for _, p := range platformDirs {
		if st, err := os.Stat(filepath.Join(dir, p)); err == nil && st.IsDir() {
			f.Platforms = append(f.Platforms, p)
		}
	}
	f.EnvAsset = envAssetRe.MatchString(pub)
	for _, n := range envExamples {
		if fsx.FileExists(filepath.Join(dir, n)) {
			f.EnvExample = n
			break
		}
	}
	for _, name := range []string{".env", f.EnvExample} {
		if name != "" && fsx.FileExists(filepath.Join(dir, name)) {
			f.EnvVars = envfile.ParseEnvFile(filepath.Join(dir, name))
			break
		}
	}
	if firebaseDepRe.MatchString(pub) {
		f.Firebase = true
		f.MissingFirebase = missingFirebase(dir, f.Platforms)
	}
	f.CodeGen = CodeGen(pub)
	f.BackendPorts = backendPorts(dir)
	f.HasTests = hasDartTests(dir)

	switch {
	case flutterVer == "":
		f.Warnings = append(f.Warnings, "Flutter was not found in PATH, install it from https://docs.flutter.dev/get-started/install")
	default:
		if f.FlutterWanted != "" && Older(flutterVer, f.FlutterWanted) {
			f.Warnings = append(f.Warnings, "the app asks for Flutter "+f.FlutterWanted+" or newer, Flutter "+flutterVer+" is installed (run `flutter upgrade`)")
		}
		if f.DartWanted != "" && dartVer != "" && Older(dartVer, f.DartWanted) {
			f.Warnings = append(f.Warnings, "the app asks for Dart "+f.DartWanted+" or newer, Dart "+dartVer+" is installed (a newer Flutter brings a newer Dart: run `flutter upgrade`, or use fvm for a version the project pins)")
		}
	}
	if f.EnvAsset && !fsx.FileExists(filepath.Join(dir, ".env")) && f.EnvExample == "" {
		f.Warnings = append(f.Warnings, ".env is listed as an asset in pubspec.yaml but the file does not exist, the build will fail until you create it")
	}
	return f
}

func missingFirebase(dir string, platforms []string) []string {
	has := map[string]bool{}
	for _, p := range platforms {
		has[p] = true
	}
	var out []string
	check := func(rel string) {
		if !fsx.FileExists(filepath.Join(dir, filepath.FromSlash(rel))) {
			out = append(out, rel)
		}
	}
	if has["android"] {
		check("android/app/google-services.json")
	}
	if has["ios"] {
		check("ios/Runner/GoogleService-Info.plist")
	}
	check("lib/firebase_options.dart")
	return out
}

// backendPorts lists the ports of localhost addresses written in the Dart sources.
func backendPorts(dir string) []string {
	seen := map[string]bool{}
	files := 0
	_ = filepath.WalkDir(filepath.Join(dir, "lib"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".dart" {
			return nil
		}
		if files++; files > 400 {
			return filepath.SkipAll
		}
		b, err := os.ReadFile(path)
		if err != nil || len(b) > 1<<20 {
			return nil
		}
		for _, m := range backendRe.FindAllSubmatch(b, -1) {
			seen[string(m[1])] = true
		}
		return nil
	})
	var out []string
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func hasDartTests(dir string) bool {
	found := false
	_ = filepath.WalkDir(filepath.Join(dir, "test"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, "_test.dart") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// AnalyzeArgs is the command `verify` runs. It fails only on errors, so a lint in someone
// else's code does not make a good setup look broken.
var AnalyzeArgs = []string{"flutter", "analyze", "--no-fatal-infos", "--no-fatal-warnings"}
