// Package buildinfo holds values set when the binary is built.
package buildinfo

import (
	"runtime/debug"
	"strings"
)

// Version is the gclone version. The root main package sets it at start-up with Pick.
var Version = "0.2.1-dev"

// Module returns the version Go recorded when the binary was installed from a tagged
// module with `go install github.com/rauni5/gclone@v0.2.0`. It is "" for a build from a
// checkout, where Go records "(devel)".
func Module() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return module(bi.Main.Version)
	}
	return ""
}

func module(v string) string {
	if v == "" || v == "(devel)" {
		return ""
	}
	return v
}

// Pick chooses the version to show. A release build sets a real version with
// -ldflags "-X main.version=..." and that wins. A development default (ending in "-dev")
// gives way to the module version when there is one, so `go install ...@latest` shows the tag
// instead of a stale development number.
func Pick(set, moduleVersion string) string {
	if !strings.HasSuffix(set, "-dev") || moduleVersion == "" {
		return set
	}
	return strings.TrimPrefix(moduleVersion, "v")
}
