# Contributing to gclone

Thanks for wanting to help. Small contributions are welcome: a typo, a clearer error
message, a test for something that already works. You do not need to understand the whole
codebase to make a useful change.

## Quick start

You need Go 1.24 or newer and git. Node.js 20+ and npm are needed for the tests that start
a real server; those tests skip themselves when Node is missing.

```bash
git clone https://github.com/rauni5/gclone.git
cd gclone
go build -o gclone .
go test ./...
```

Once you have a built `gclone`, you can let it do the same for itself:
`./gclone rauni5/gclone my-copy` clones, downloads modules, and writes a `SETUP_GUIDE.md`;
`./gclone verify my-copy` builds it. This is how we check the Go support.

Try it on a throwaway repository:

```bash
./gclone --dry-run someone/app
```

## Project structure

```
main.go                  entry point: sets the version, calls internal/cli
internal/
  cli/                   arguments, and the flow for one repository (setupRepo)
  project/               shared data model: Project, Pkg, Need, Companion, Report, Options
  analyze/               read a repository and build a Project (no side effects)
  adapter/               the Adapter interface; one folder per ecosystem below
  adapters/              the list of adapters (add new ecosystems here)
  adapter/node/          Node.js: packages, package manager, Node version, runnables
  adapter/golang/        Go: go.mod, Go version check, build command
  adapter/java/          Java: Maven/Gradle, Spring Boot, application config, Java version
  adapter/flutter/       Flutter: pubspec, Flutter/Dart version, pub get, Firebase and .env notes
  adapter/python/        Python: pip/Poetry/uv/Pipenv, Django/FastAPI/Flask/Streamlit, .venv
  compose/               docker compose file parsing
  envfile/               env example files: parse, classify, find variables used in code
  services/              which databases a project needs, start or install them, ask first
  docker/                Docker states (missing, stopped, denied), install/start, images
  installer/             one install/start option, run after the user agrees
  plan/                  build, show and execute the setup steps
  repos/                 other repositories a project needs (README, local deps, submodules)
  verify/                build, start, port check, HTTP probe, report
  devcontainer/          `gclone export devcontainer`: builds the .devcontainer files as text
  doctor/                `gclone doctor`: tools, Docker, ports and disk, nothing changed
  guide/                 write SETUP_GUIDE.md
  explain/               plain-language hints for failed commands (a list of patterns)
  state/                 .gclone/state.json: write, read, `gclone status`, `--resume`
  proc/                  run programs, process groups (unix and windows files)
  fsx/, ui/, buildinfo/  small helpers: files, prompts, version
testdata/                small sample repositories used by tests
integration/             runs the built gclone binary on those samples
docs/                    architecture, good first issues, competitor analysis, growth plan
Makefile                 make build | test | check | release
```

[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) explains how the pieces fit together.

## Ground rules

These keep gclone safe to run on code you did not write.

1. **Ask before changing the machine.** Anything that installs software, starts a service,
   clones another repository or runs project code needs a prompt or an explicit flag. The
   default answer for risky actions is *no*.
2. **Never print or store secret values.** Variable names are fine; values from `.env` files
   are not. Check your log lines and the guide.
3. **Never overwrite or delete the user's files.** An existing `.env` stays as it is.
4. **No shell strings built from repository content.** Use `exec.Command(name, args...)`.
5. **Prefer the standard library.** gclone has no dependencies today. Open an issue before
   adding one.
6. **Say what you did not test.** If you cannot run a code path (Windows, a real Docker
   engine, winget), say so in the pull request.

## Coding conventions

- `gofmt` and `go vet` must be clean (CI checks both).
- Errors start with a lowercase letter and say what failed and what the user can do.
- User-facing text is plain and short. Terminal status markers are `[ok]`, `[..]`, `[!!]`,
  `[warn]`, `[FAIL]` and `[skip]`.
- Keep detection (reading files) separate from execution (running commands) where you can.
  Detection should be testable without touching the machine.
- Write a test with your change. Table tests are welcome.

## Running the tests

```bash
go test ./...            # everything
go test -run Verify ./...  # one group
make check            # gofmt, vet and tests, same as CI
```

Tests that need Node (`TestVerify*`) start real short-lived servers on free ports.

**Integration tests.** `integration/` builds the real binary once and runs it on the folders
in `testdata/`: a dry run of each (detected stack, planned steps, nothing written into the
fixture), one real run of the Go sample, `--ci` exiting 1 on a failed step, and the usage
errors. They need only Go and git, no network. `go test -short ./...` skips them.

**Adding a language?** Add a small sample under `testdata/<name>/` and a row in
`TestFixturesDryRun` (`integration/integration_test.go`), with the text the detected line and the
plan must contain. That row is how we know the adapter is wired in and works on a real folder.

**Faking Docker.** There is no Docker engine in most test environments. To test a Docker
code path, use the helper in `internal/docker/fake_docker_test.go`:
`NewFakeDocker(t, mode)` creates a temporary directory containing an executable
`docker` script, prepends it to `PATH` for the test, and records the arguments it
receives (`FakeDocker.Calls`). Supported modes: `FakeDockerReady`,
`FakeDockerStopped`, `FakeDockerPermissionDenied`, and
`FakeDockerImageDownloadFail`. Tests using it are `//go:build !windows` and skip
on Windows with a clear reason because the helper writes a POSIX shell script.
Use it to test `docker.State`, `docker.EnsureImage` and the OfferInstall
decisions (`internal/services/offer_install_options_test.go`). Real Docker
behaviour should still be checked by hand before you claim it works.

## Pull request workflow

1. Open an issue first for anything bigger than a small fix, so we can agree on the approach.
2. Fork, create a branch, make the change with tests.
3. Run `gofmt -l .`, `go vet ./...` and `go test ./...`.
4. Update `CHANGELOG.md` under **Unreleased** and any docs the change affects.
5. Open a pull request and fill in the template. Keep it focused: one change per PR.
6. A maintainer reviews it. Please expect some back and forth; it is about the code, not you.

Commit messages: a short imperative first line (`Add RabbitMQ detection`), then details if
needed.

## Using AI tools

AI assistants are welcome here; the maintainer uses one too. What matters is that a person
stands behind the change.

- **You are responsible for every line you submit.** Read it, run it, and be able to explain
  why it is the way it is when a reviewer asks.
- **Say so when an AI wrote a substantial part.** One line in the pull request is enough ("Drafted
  with Claude, I ran X and Y"). It is not a penalty. It tells the reviewer where to look
  harder.
- **Run it for real.** Say what you ran and what it printed. "Tests pass" is not enough for
  a change to Docker, package managers or a new language: say whether you tried it on a real
  project or only with a fake (ground rule 6).
- **No invented facts.** Do not claim a command, flag, version or behaviour that you did not
  check. Generated text in the README, comments or a pull request description gets the same
  review as code.
- **Keep it small and about one thing.** Large generated changes, drive-by rewrites and
  unrelated formatting are hard to review and will be asked to be split.

A pull request may be closed, with a short reason, when it clearly was not read or run by its
author, does not follow the ground rules above, or has no explanation. You are welcome to
reopen it after fixing that. For issues marked `good first issue`, please comment first so two
people do not work on the same one.

## Adding a feature

1. Find where it belongs using the structure above.
2. Add detection first and test it with plain files in a temporary directory.
3. Add the action second, behind a prompt or flag if it changes the machine.
4. Show it in the plan output and in `SETUP_GUIDE.md` so the user can see what happened.
5. Document the flag or behaviour in the README.

## Adding support for a new language or framework

Today the Node.js logic lives in `internal/adapter/node`.
A proper adapter interface is planned together with the Java adapter (see
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)), so **please open an issue before writing a
new ecosystem**, and we will agree on the shape together. In the meantime, you can help
with:

- more frameworks in the `frameworks` list in `internal/adapter/node/node.go`
- another service in `svcDefs`, `depKind` and `imageKind` in `internal/services/services.go`
- another env example file name in `envFileNames` in `internal/envfile/envfile.go`

Each of these is a small, self-contained change. See
[docs/GOOD_FIRST_ISSUES.md](docs/GOOD_FIRST_ISSUES.md).

## Reporting bugs

Use the bug report form. The most useful reports include the command you ran, the full
output, your OS, `gclone --version`, and a link to the repository if it is public. **Remove
secrets** from anything you paste, including `SETUP_GUIDE.md`.

For security problems, do not open a public issue. See [SECURITY.md](SECURITY.md).

## Code of Conduct

By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
