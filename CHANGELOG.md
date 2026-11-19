# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/). While the version is 0.x, minor versions may
include breaking changes.

## [Unreleased]

### Added

- README: examples of `--json` with `jq` for the failed steps, the last verification
  verdict, the services a project needs, the error message, and `ok` as a CI gate.
- Flutter: notices `build_runner`, `freezed`, `json_serializable` and `riverpod_generator` in
  `pubspec.yaml`, adds `dart run build_runner build` to the plan after `flutter pub get`, and
  mentions it in the guide.
- Integration tests (`integration/`): build the real binary and run it on every sample
  project (dry run, detected stack, planned steps, nothing written), one real run of the Go
  sample with `--resume`, `--ci` exiting 1 on a failed step, and usage errors. Skipped with
  `go test -short`.
- `--ci`: for scripts and CI. It never asks or reads the terminal (every question is answered
  no), implies `--yes`, and exits 1 when a setup step or the verification fails.
- `--json`: prints one JSON result on standard output (`ok`, detected stacks, steps,
  services, verification checks) and sends every other message to standard error.
  `gclone status --json` prints the recorded state. No variable values are included.
- Flutter adapter: finds Flutter apps, checks the installed Flutter and Dart against the
  app, runs `flutter pub get`, creates `.env` from an example, and notes missing Firebase
  files and a backend on `localhost`. `verify` runs `flutter analyze` (errors only) and does
  not start the app. New failure hints for Dart SDK constraints, pub version solving and
  Android licenses. Tested with a stand-in `flutter` command only.
- Python adapter: Django, FastAPI, Flask and Streamlit, with pip, Poetry, uv or Pipenv. pip
  projects get a `.venv` inside the project (never the system Python). It reads the wanted
  Python version, `.env.example`, database needs (including SQLAlchemy URLs such as
  `postgresql+psycopg2://`), and `verify` starts the app. New failure hints for pip
  (externally managed Python, `pg_config`, wheel build failures, no matching version).
- `.gclone/state.json`: a record of the setup steps, the services the project needs and the
  last verification, kept out of `git status`. It holds no variable values and no remote
  login.
- `gclone status [directory]` shows that record and whether the repository has moved since
  setup. It exits with 1 when a step or the last verification failed.
- `gclone setup --resume` skips install steps that already succeeded at the same commit.
  Services are always started again.

### Changed

- `gclone verify` says "no supported project found" instead of listing three ecosystems.
- The message shown when `npm ci` falls back to `npm install` now explains why and what to
  do next (#11, thanks @dbdefault).
- Node.js, Python, Go, Java and Flutter now implement one shared `Adapter` interface, so adding an ecosystem
  no longer means editing every stage. The "Detected:" list is now in the order Node.js, Python,
  Go, Java, Flutter, and the guide says "This repo has N projects" across all ecosystems.

## [0.1.1] - 2026-11-14

### Added

- Plain-language hints when a setup step fails (npm dependency conflicts, missing build
  tools, port already in use, wrong Java or Go version, registry login, network and proxy
  problems and more). Shown in the terminal and in the guide, always worded as a best guess.
- Java support (early): Maven and Gradle projects, Spring Boot detection, Java version check,
  `application.properties` / `application.yml` reading, database detection from JDBC URLs, a
  dependency download step, and build plus start in `verify`. Not yet run end to end against
  real builds.
- Go support: gclone reads `go.mod`, warns when the installed Go is older than the module
  asks for, runs `go mod download`, and `verify` builds the module. gclone can now set up
  its own repository.
- Restructured into `main.go` plus `internal/` packages, with a Makefile and test fixtures.
- Release tooling: `scripts/build-release.sh`, a release workflow that publishes archives
  and SHA-256 checksums, CI and lint workflows.
- Contributor documentation, issue and pull request templates, security and support policies.

## [0.1.0]

### Added

- `gclone <repo> [directory]` clones a repository and sets up Node.js projects (npm, pnpm,
  yarn, bun) found at the repository root or one folder down.
- `gclone setup [directory]` does the same for a repository that is already cloned.
- Node version check against `.nvmrc`, `.node-version`, `.tool-versions` and `engines`.
- `.env` creation from example files, variable triage (must set, change for real use,
  fine) and detection of variables the code reads but the example omits.
- Detection of MongoDB, PostgreSQL, MySQL and Redis; starting compose database services; an
  ask-first flow to run a missing database in Docker or install it natively.
- Handling of Docker not installed, not running, permission denied and image download
  failures.
- Detection of other repositories a project needs (README links, local `file:` dependencies,
  compose build contexts, API URLs, git submodules), with ask-first cloning and setup.
- `gclone verify`: environment, services, build, start, port and HTTP checks, with exit
  code 1 when not ready. Optional after setup, and recorded in the guide.
- `SETUP_GUIDE.md` generation, excluded from `git status`.
- `--dry-run`, `--yes`, `--ignore-scripts`, `--no-install`, `--no-services`,
  `--install-services`, `--clone-deps`, `--no-deps`, `--verify`, `--no-build`,
  `--timeout`.

### Security

- The guide and terminal output list variable names and notes, not values from env files
  (exceptions: `PORT` as the expected port, and the login of a database container gclone
  creates).

[Unreleased]: https://github.com/rauni5/gclone/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/rauni5/gclone/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/rauni5/gclone/releases/tag/v0.1.0
