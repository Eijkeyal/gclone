# Scripts and CI

`--ci` is for places where nobody can answer a question. Every prompt is answered "no" (so no
database is installed and no other repository is cloned unless you pass `--install-services`
or `--clone-deps`), nothing reads the terminal, and the exit code is 1 when a setup step or
the verification fails. Without `--ci`, a failed step is recorded in the guide and the exit
code stays 0.

```bash
gclone setup . --ci --verify         # set up the checked-out repository and verify it
gclone <repo-url> app --ci --json    # clone, set up, print the result as JSON
gclone status --json                 # the recorded state as JSON, exit 1 if something failed
```

`--json` keeps standard output clean: the one JSON document is the only thing on it, and
every other message goes to standard error. The fields:

| Field | Meaning |
|---|---|
| `schema`, `gclone` | JSON format version and gclone version |
| `ok` | `true` when there was no error, no failed step and the verdict is not `NOT READY` or `INTERRUPTED` |
| `dir`, `commit` | The folder and the short commit |
| `detected`, `warnings` | The lines of the "Detected:" and "Warnings:" lists |
| `steps` | Each step with `where`, `name`, `cmd`, `status` (`ok` or `failed`), `note` and `hint` |
| `services` | Databases and other services the project needs, with `kind`, `port`, `remote` |
| `verify` | `verdict`, counts, and every check with `status` (`pass`, `warn`, `fail`, `skip`) |
| `error` | The error message, when there was one |

Like the state file, it holds names, commands and notes, never variable values. Fields may be
added; `schema` changes only when one is removed or changes meaning. A GitHub Actions job can
be as short as `go install github.com/rauni5/gclone@latest` followed by `gclone setup . --ci
--verify` (the workflow itself has not been run on GitHub yet).
