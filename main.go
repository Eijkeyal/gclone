// Command gclone clones a git repository and sets up everything it needs to run.
package main

import (
	"github.com/rauni5/gclone/internal/buildinfo"
	"github.com/rauni5/gclone/internal/cli"
)

// version is set at build time with -ldflags "-X main.version=vX.Y.Z". When it is left at its
// development default, a binary installed with `go install ...@vX.Y.Z` shows that tag instead.
var version = "0.2.1-dev"

func main() {
	buildinfo.Version = buildinfo.Pick(version, buildinfo.Module())
	cli.Main()
}
