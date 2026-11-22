# Supported today

What gclone does for each kind of project, and how well each part has been tested. Where it
says "not tested", it means it. Please report what happens on your projects.

## Features

### Available

- **Clone and analyze.** Plain `git clone`, then finds `package.json` at the root and one
  folder down (a typical `client/` and `server/` split).
- **Package manager and Node version.** Picks npm, pnpm, yarn or bun from the
  `packageManager` field or the lockfile, uses `npm ci` when there is a lockfile, and warns
  when your Node differs from `.nvmrc`, `.node-version`, `.tool-versions` or `engines`.
- **Environment files.** Creates `.env` from `.env.example` (or a similar name). It never
  overwrites an existing `.env`.
- **Environment triage.** Sorts variables into *must set* (empty or placeholder), *works on
  defaults, change for real use* (secrets, database URLs) and *fine*. It also lists variables
  the source code reads that the example file does not mention.
- **Databases and services.** Detects MongoDB, PostgreSQL, MySQL, Redis and RabbitMQ from
  dependencies, Prisma and connection URLs, and starts database services from a compose file.
  If one is missing it **asks** whether to run it in Docker or install it with your package
  manager. Handles Docker being missing, stopped, denied or unable to download an image.
- **Other repositories.** Notices when a frontend needs a backend that lives elsewhere
  (README links, `file:` dependencies, compose build contexts, an API URL pointing at
  localhost, git submodules), asks, clones it and sets it up too.
- **Verification.** `gclone verify` builds the project, starts it, waits for the port, makes
  one request and stops it. Exit code 1 means not ready, so it works in CI.
- **A guide, not just side effects.** `SETUP_GUIDE.md` records what ran, how to start each
  part, what to change, and the verification result. It is excluded from `git status`.
- **A record and a way to resume.** gclone writes `.gclone/state.json` (excluded from
  `git status`) with the steps it ran, the services the project needs and the last
  verification. `gclone status` shows it and what is out of date, and
  `gclone setup --resume` skips install steps that already succeeded at the same commit.
  The file holds names, commands and notes only, never variable values.
- **Scripts and CI.** `--ci` never asks or reads the terminal and exits 1 when a step or the
  verification fails. `--json` prints one JSON result on standard output. See
  [Use in scripts and CI](CI.md).
- **Dry run and safety prompts.** See [Safety](SAFETY.md).

### Experimental

These are implemented but have only been exercised in limited ways. Please report problems.

- **Native database installs** (winget, Homebrew, apt). The commands are written but have not
  been tested on real machines. The Winget package IDs in particular are unverified.
- **Installing or starting Docker itself.** Same status.
- **Windows and macOS.** The tests run there in CI, but the maintainer has not used gclone by
  hand on either yet.
- **pnpm, yarn and bun.** Detected and invoked as `<manager> install`, not yet tested against
  real projects.
- **Finding a backend from a README link.** Covered by unit tests only.

### Planned

See the [roadmap](ROADMAP.md).

## What is supported, and how well it is tested

| Area | Status |
|---|---|
| Node.js with npm | Tested end to end: clone, install, `.env`, verify |
| Node.js with pnpm, yarn, bun | Detected and invoked, not tested on real projects |
| Java: Maven, Gradle, Spring Boot | Reads the build file (Java version, Spring Boot), uses `mvnw`/`gradlew` or `mvn`/`gradle`, reads `application.properties` / `application.yml` (port, database URL, settings that need values), detects PostgreSQL, MySQL/MariaDB, MongoDB and Redis, runs a dependency download, and `verify` builds and starts Spring Boot. **Detection is unit-tested; the real build and start have not been run end to end yet** (no access to Maven Central in my test environment), and Gradle commands are untested. Please report what happens on your projects |
| Python: Django, FastAPI, Flask, Streamlit | Finds `requirements.txt`, `pyproject.toml`, `Pipfile` or `setup.py`, picks pip, Poetry, uv or Pipenv from the lock file, and reads the wanted Python version (`.python-version`, `requires-python`, Poetry, Pipfile, `runtime.txt`). For pip it creates a `.venv` inside the project and installs into it, never into your system Python. Finds the web framework and its start command (`manage.py runserver`, `uvicorn`, `flask run`, `streamlit run`) and default port, reads `.env.example`, detects PostgreSQL, MySQL, MongoDB and Redis from dependencies and URLs, and `verify` starts the app and checks it answers. **Tested end to end with a real Flask project** (venv, pip install, start, HTTP 200); Django, FastAPI, Streamlit, Poetry, uv and Pipenv are covered by detection tests only |
| Flutter | Finds apps whose `pubspec.yaml` depends on Flutter, compares the installed Flutter and Dart with `environment:`, runs `flutter pub get`, creates `.env` from an example file, and notes what an app usually needs by hand: a `.env` asset that is missing, Firebase files (`google-services.json`, `GoogleService-Info.plist`, `firebase_options.dart`) and a backend on `localhost` / `10.0.2.2`. `verify` runs `flutter analyze` (errors only) and never starts the app, because it needs a device you choose. **Tested only against a stand-in `flutter` command, not the real SDK** (no Flutter in my test environment). Please report what happens on your projects |
| Failure hints | A list of about 20 common failures (npm conflicts, missing build tools, port in use, wrong Java or Go version, registry login, network and proxy). It is a best guess from the output and says so. Unknown errors get no hint |
| Go modules | `go.mod` read, installed Go compared with the `go` / `toolchain` line, `go mod download`, build check in `verify`. Nothing is started. Tested on gclone itself and one small fixture |
| Node version check | `.nvmrc`, `.node-version`, `.tool-versions`, `engines`. It warns, it does not switch versions (use nvm, fnm or mise for that) |
| Repository layout | Root plus one folder level. Workspaces are analyzed at the root only |
| MongoDB, PostgreSQL, MySQL, Redis, RabbitMQ | Detection tested. Starting them in Docker tested with a stand-in `docker` command, not with a real Docker engine |
| Compose files | Database services only (postgres, mysql, mariadb, mongo, redis, valkey, rabbitmq). It never starts your app's own containers |
| Rust, PHP, Ruby | Detected and reported only. No setup yet |
