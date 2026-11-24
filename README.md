<div align="center">

# gclone

**`git clone`, plus everything the project needs to actually run.**

[![CI](https://github.com/rauni5/gclone/actions/workflows/ci.yml/badge.svg)](https://github.com/rauni5/gclone/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/rauni5/gclone)](go.mod)

</div>

Cloning a repository is the easy part. Then comes the hour of reading the README, finding
the right Node version, copying `.env.example`, working out which variables matter,
starting a database, and discovering that the backend lives in a different repository.

`gclone` does that part. It clones the repository, works out what the project needs,
**asks before it changes anything on your machine**, sets it up, checks that the app
really starts, and writes a `SETUP_GUIDE.md` that says what is left for you to decide.

> **Status: early (0.x).** Node.js is the most tested. Python, Java, Go, Rust and Flutter are newer
> and tested less. See [what works today](docs/SUPPORTED.md). Feedback from real repositories
> is the most useful thing you can give.

## Install

```bash
go install github.com/rauni5/gclone@latest
gclone --version
```

You need `git` and Go 1.24 or newer. Starting from nothing, or prefer a downloaded binary?
Follow the [full install guide](docs/INSTALL.md): git, then Go, then gclone, on Windows,
macOS and Linux.

## Use it

```bash
gclone someone/app                 # clone and set up (GitHub shorthand, a URL or a local path)
gclone someone/app --dry-run       # look first: clone and show the plan, change nothing else
cd app && gclone verify            # build it, start it, check that it answers
```

More commands (`setup`, `status`, `doctor`, `export devcontainer`), all flags and the
`--ci` / `--json` modes are in the [usage guide](docs/USAGE.md).

## What it looks like

Real output from a small Express project (npm's funding and audit lines trimmed). It uses a
local repository so the run is reproducible; a URL or `user/repo` works the same way.

```console
$ gclone ./shop-api demo --yes --verify
Cloning ./shop-api
Cloning into 'demo'...
done.

Detected:
  - .: shop-api, npm, Express

Plan:
  1. Create .env from .env.example   (in repo root)
       $ copy .env.example .env
  2. Install dependencies (npm)   (in repo root)
       $ npm install

[..] Create .env from .env.example (in repo root)
[ok] done

[..] Install dependencies (npm) (in repo root)
added 68 packages, and audited 69 packages in 4s
[ok] done

Verifying. Each part is built, started for a few seconds, probed and stopped.

  repo root
    [warn] env: still empty or a placeholder: STRIPE_API_KEY; read by the code but not defined: SMTP_HOST
    [ok]   start: npm run dev is listening on port 4000 after 400ms
    [ok]   http: GET / answered 200

Result: READY WITH WARNINGS (1 warning(s))

Guide written to demo/SETUP_GUIDE.md
```

## What it does

- **Sets up the project:** package manager and language version checks, `.env` from the
  example file, dependencies. It never overwrites an existing `.env`.
- **Sorts your environment variables** into must set, change for real use, and fine.
- **Handles databases:** detects them, starts compose services, and asks before running or
  installing anything else. It copes with Docker missing, stopped or unable to pull an image.
- **Finds other repositories** a project needs (a backend, say) and offers to set them up.
- **Verifies:** builds the project, starts it and makes one request, with an exit code for CI.
- **Writes a guide** (`SETUP_GUIDE.md`) and a record (`.gclone/state.json`) of what it did.

## Supported today

Node.js (npm, pnpm, yarn, bun), Python (Django, FastAPI, Flask, Streamlit; pip, Poetry, uv,
Pipenv), Java (Maven, Gradle, Spring Boot), Go, Rust (Cargo) and Flutter. How well each one is tested is
stated plainly in [docs/SUPPORTED.md](docs/SUPPORTED.md).

## Safety

**gclone runs code from the repository you point it at**: installing dependencies runs install
scripts, and `verify` runs the project's own build and start scripts. It asks before it
installs, starts or clones anything, never prints secret values, and never deletes anything.
Use `--dry-run` first on code you do not trust. Details: [docs/SAFETY.md](docs/SAFETY.md).

## Documentation

| Guide | What is in it |
|---|---|
| [Install](docs/INSTALL.md) | git, Go and gclone from scratch, on Windows, macOS and Linux, plus troubleshooting |
| [Usage](docs/USAGE.md) | Getting started, every command and flag, `doctor`, the repo's own setup command |
| [Scripts and CI](docs/CI.md) | `--ci`, `--json` and the fields of the JSON result |
| [Dev Containers](docs/DEVCONTAINER.md) | `gclone export devcontainer` |
| [Supported today](docs/SUPPORTED.md) | Features, and what is tested and what is not |
| [Safety and security](docs/SAFETY.md) | What runs, what asks first, what is never stored |
| [How gclone compares](docs/COMPARISON.md) | Rex, Bivvy, mise, Dev Containers |
| [Roadmap](docs/ROADMAP.md) | What is next |
| [Architecture](docs/ARCHITECTURE.md) | How it is built, for contributors |

## Contributing

Contributions are welcome, including small ones. Start with
[CONTRIBUTING.md](CONTRIBUTING.md) and the list of
[good first issues](docs/GOOD_FIRST_ISSUES.md).

## Community

- [Issues](https://github.com/rauni5/gclone/issues): bug reports, feature requests, documentation fixes
- [Support](SUPPORT.md): how to ask for help
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md)
- [Changelog](CHANGELOG.md)

## License

[MIT](LICENSE)
