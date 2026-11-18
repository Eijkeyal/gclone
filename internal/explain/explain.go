// Package explain turns the output of a failed command into a short, plain-language
// hint. It is rule based: a list of patterns, each with what it probably means and what to
// try. It never claims certainty, and it never changes anything on the machine.
package explain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/rauni5/gclone/internal/proc"
)

// Hint is what we think went wrong and what the person can try.
type Hint struct {
	ID      string
	Meaning string
	Try     string
}

type rule struct {
	id    string
	re    *regexp.Regexp
	build func(m []string) (meaning, try string)
}

func fixed(meaning, try string) func([]string) (string, string) {
	return func([]string) (string, string) { return meaning, try }
}

// Order matters: specific rules come before general ones.
var rules = []rule{
	{"go-version", regexp.MustCompile(`requires go >= ([\d.]+)`), func(m []string) (string, string) {
		return "This Go module needs Go " + m[1] + " or newer, and the installed Go is older.",
			"Install a newer Go from https://go.dev/dl/. Go 1.21 and newer can also download the right toolchain by itself when GOTOOLCHAIN=auto."
	}},
	{"java-too-old", regexp.MustCompile(`(?i)invalid target release|release version \d+ not supported|UnsupportedClassVersionError|class file version \d+`),
		fixed("The project was built for a newer Java than the one that is running.",
			"Install the JDK version the project asks for (see the guide, or the java.version line in pom.xml) and make sure `java -version` shows it.")},
	{"gradle-java-mismatch", regexp.MustCompile(`(?i)Unsupported class file major version`),
		fixed("This Gradle version is too old for the Java that is installed (or the other way round).",
			"Use the project's `./gradlew` wrapper rather than a global Gradle, or switch to the Java version the project expects.")},
	{"java-home", regexp.MustCompile(`(?i)JAVA_HOME.{0,40}(not set|not defined correctly|invalid)`),
		fixed("The build tool cannot find a Java installation.",
			"Set JAVA_HOME to your JDK folder (a JDK, not only a JRE), then open a new terminal and try again.")},
	{"flutter-sdk-constraint", regexp.MustCompile(`requires SDK version|current Dart SDK version is|Because .* requires Flutter SDK`),
		fixed("The app asks for a newer Dart or Flutter than the one installed.",
			"Run `flutter upgrade`, or install the version the project pins (the guide lists it). fvm can keep several versions side by side.")},
	{"pub-solving", regexp.MustCompile(`version solving failed`),
		fixed("pub could not find package versions that work together.",
			"Read the lines above: they name the packages that conflict. Often a newer Flutter or an older `pubspec.lock` is the cause. Do not delete pubspec.lock without asking the maintainers.")},
	{"android-licenses", regexp.MustCompile(`Android license status unknown|licenses not accepted|cmdline-tools component is missing`),
		fixed("The Android SDK is missing a part or its licenses are not accepted, so Android builds cannot run.",
			"Run `flutter doctor` and follow its Android lines, then `flutter doctor --android-licenses`. You can still run the app on web or desktop.")},
	{"pip-externally-managed", regexp.MustCompile(`externally-managed-environment`),
		fixed("This Python is managed by the operating system, so pip refuses to install into it.",
			"Use a virtual environment (gclone creates `.venv` for pip projects) rather than the system Python, and do not use `--break-system-packages`.")},
	{"python-venv-missing", regexp.MustCompile(`ensurepip is not\s+available|install the python3(\.\d+)?-venv\s+package`),
		fixed("Python's venv module cannot set up pip in the new virtual environment because the python3-venv package is missing.",
			"Install it with `sudo apt install python3-venv` (or the versioned package the error names, such as python3.12-venv), then run again.")},
	{"pip-pg-config", regexp.MustCompile(`pg_config executable not found|libpq-fe\.h`),
		fixed("The PostgreSQL driver (psycopg2) needs PostgreSQL's client libraries to compile.",
			"Install the client libraries (Linux: libpq-dev; macOS: `brew install libpq`), or ask whether the project can use `psycopg2-binary` instead.")},
	{"pip-build", regexp.MustCompile(`Failed building wheel|Microsoft Visual C\+\+ 14\.0|error: command 'gcc' failed|Failed to build`),
		fixed("A Python package needs to compile native code and the build tools are missing, or the package does not support this Python version yet.",
			"Install the build tools (Windows: Visual Studio Build Tools; macOS: `xcode-select --install`; Linux: build-essential and python3-dev), or use an older Python the project supports.")},
	{"pip-no-dist", regexp.MustCompile(`No matching distribution found|Could not find a version that satisfies the requirement`),
		fixed("pip could not find a version of a package that works with this Python, or the package name or version is wrong.",
			"Check the version in the requirements file. If it is a very new or old Python, try the version the project asks for (see the guide).")},
	{"npm-eresolve", regexp.MustCompile(`ERESOLVE`),
		fixed("npm found packages whose version requirements conflict with each other, usually a peer dependency on an older version of a library.",
			"Try `npm install --legacy-peer-deps` in that folder. If it works, the project's own docs may already say to use it. Otherwise ask the maintainers to update the dependencies.")},
	{"npm-target", regexp.MustCompile(`ETARGET|notarget`),
		fixed("A dependency asks for a version that is not on the registry. The version may have been removed or mistyped, or the package is private.",
			"Look at the package named in the error in package.json. Try a version that exists (`npm view <package> versions`), or ask the maintainers.")},
	{"npm-lifecycle", regexp.MustCompile(`ELIFECYCLE`),
		fixed("One of the project's own scripts (often postinstall or prepare) failed during install.",
			"Scroll up to the first error above this message, it names the real cause. You can also run again with `--ignore-scripts` to install without running scripts.")},
	{"node-gyp", regexp.MustCompile(`(?i)node-gyp|gyp ERR!`),
		fixed("A package needs to compile native code, and either the build tools are missing or the Node version is too new for that package.",
			"Install the build tools (Windows: Visual Studio Build Tools; macOS: `xcode-select --install`; Linux: build-essential and python3), or use the Node version the project asks for (nvm, fnm or mise).")},
	{"engine", regexp.MustCompile(`EBADENGINE|Unsupported engine`),
		fixed("The installed Node version is different from the one the project says it supports.",
			"Switch to the version in `.nvmrc` or `engines` (nvm, fnm or mise). Many projects still work, but this is the first thing to rule out.")},
	{"port-in-use", regexp.MustCompile(`(?i)EADDRINUSE|address already in use|Port \d+ (is )?already in use|Web server failed to start\. Port (\d+) was already in use`), func(m []string) (string, string) {
		return "Another program is already using the port this app wants.",
			"Stop that program, or start this app on another port (usually with the PORT environment variable or `server.port` for Spring Boot)."
	}},
	{"tls-intercept", regexp.MustCompile(`(?i)SELF_SIGNED_CERT_IN_CHAIN|unable to get local issuer certificate|PKIX path building failed|CERT_HAS_EXPIRED`),
		fixed("A secure connection was refused because the certificate could not be trusted. This often happens behind a company or school proxy.",
			"Ask your network admin for the proxy's CA certificate and add it to the tool (for npm: `NODE_EXTRA_CA_CERTS`; for Java: the truststore). Do not turn certificate checks off.")},
	{"auth", regexp.MustCompile(`(?i)\bE401\b|\bE403\b|401 Unauthorized|403 Forbidden|authentication required|Could not transfer artifact.*(401|403)`),
		fixed("The package registry refused the request, usually because a package is private and needs a login or a token.",
			"Log in to the registry (`npm login`, or add a token to the project's `.npmrc` / Maven `settings.xml`). Ask the project's owner which registry it uses.")},
	{"git-access", regexp.MustCompile(`(?i)Repository not found|Permission denied \(publickey\)|Authentication failed|could not read Username`),
		fixed("Git could not open the repository: the address may be wrong, or it is private and you are not signed in.",
			"Check the address. For a private repository sign in first (`gh auth login`, or set up an SSH key) and try again.")},
	{"docker-pull", regexp.MustCompile(`(?i)pull access denied|manifest unknown|repository does not exist`),
		fixed("Docker could not download an image: the name is wrong, or the image is private.",
			"Check the image name in the compose file. For a private image run `docker login` first.")},
	{"network", regexp.MustCompile(`(?i)ENOTFOUND|EAI_AGAIN|getaddrinfo|Could not resolve host|Unknown host|Name or service not known|ETIMEDOUT|ECONNRESET|Temporary failure in name resolution|no such host`),
		fixed("The computer could not reach a server it needed (a package registry or download site).",
			"Check your internet connection, VPN or proxy and try again. If you are behind a proxy, set `HTTPS_PROXY`.")},
	{"disk", regexp.MustCompile(`(?i)ENOSPC|No space left on device`),
		fixed("The disk is full.", "Free some space, for example by clearing old package caches, and run again.")},
	{"permission", regexp.MustCompile(`EACCES|(?i)permission denied`),
		fixed("A program was not allowed to write to a folder.",
			"Do not use `sudo`. Check that you own the project folder and the package cache folder, and fix their ownership if not.")},
}

// For returns a hint for the output of a failed command.
func For(output string) (Hint, bool) {
	for _, r := range rules {
		if m := r.re.FindStringSubmatch(output); m != nil {
			meaning, try := r.build(m)
			return Hint{ID: r.id, Meaning: meaning, Try: try}, true
		}
	}
	return Hint{}, false
}

// FromError looks at the output captured by proc.RunStream, if the error carries any.
func FromError(err error) (Hint, bool) {
	var re *proc.RunError
	if errors.As(err, &re) {
		return For(re.Output)
	}
	return Hint{}, false
}

// Print writes the hint under a failed step.
func Print(h Hint) {
	fmt.Println()
	fmt.Println("What this probably means: " + h.Meaning)
	fmt.Println("What to try: " + h.Try)
}

// Line is the one-line form used in the setup guide.
func (h Hint) Line() string {
	return strings.TrimSpace(h.Meaning + " " + h.Try)
}
