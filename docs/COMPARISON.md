# How gclone compares

From each project's own documentation, read in November 2026. Where a page does not
mention something, that means "not documented", not "not supported". Please open an issue if
anything here is out of date.

| Tool | What it is | Needs a config file in the repo? |
|---|---|---|
| **gclone** | Clone, set up services and env, verify, write a guide | No |
| [Rex](https://github.com/rexrun-dev/rex) | Zero-config project runner for many stacks, with `rex clone <url>` | No (optional `rex.toml`) |
| [Bivvy](https://bivvy.dev/docs/) | Declarative setup steps with state tracking | Yes (`.bivvy/config.yml`) |
| [mise](https://mise.jdx.dev/) | Tool versions, env vars and tasks | Yes (`mise.toml`) |
| [Dev Containers](https://containers.dev/) | Containerized environment spec | Yes (`devcontainer.json`) |

- **Rex is the closest.** `rex clone` also clones, detects the stack and installs
  dependencies, and Rex supports many more languages than gclone does today. gclone's
  focus is the part after the install: environment variables, databases, other repositories
  and checking that the app starts.
- **Bivvy** tracks which steps are done and re-runs only what changed, which gclone does not
  do yet. It expects you (or the project) to write the steps.
- **mise** installs and switches tool versions. gclone only warns about a mismatch. They work
  well together: let mise provide Node, let gclone do the rest.
- **Dev Containers** and **Nix** give isolated or reproducible environments. gclone sets
  things up on your own machine and does not replace either. Use them when the project
  ships one.

gclone is meant to complement these, not replace them.

The longer research behind this table is in [COMPETITOR_ANALYSIS.md](COMPETITOR_ANALYSIS.md).
