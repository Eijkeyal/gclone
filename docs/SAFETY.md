# Safety and security

**gclone runs code from the repository you point it at.** Installing dependencies runs
install scripts, and `verify` runs the project's own build and start scripts. gclone does
not sandbox this and does not analyze what those scripts do. Only use it on code you trust,
or use `--dry-run` first and read the plan.

- **Commands.** External programs are started with an argument list, not a shell string
  built from repository content.
- **Asking first.** The plan is shown before anything runs. Installing or starting a
  database, installing or starting Docker, cloning another repository, and running the
  verification each ask separately. Those default to *no* or *skip*. When it needs
  administrator rights (apt) it runs `sudo` visibly and you enter your password yourself.
- **Install scripts.** Before installing, gclone lists the repository's own `preinstall`,
  `install`, `postinstall` and `prepare` scripts. `--ignore-scripts` stops them and also
  stops your dependencies' install scripts.
- **Secrets.** `.env` is created with owner-only permissions, and an existing `.env` is never
  overwritten. The guide and the terminal output list variable *names* and notes, not their
  values. There are two exceptions: `PORT` is shown as the expected port, and when gclone
  creates a database container the guide shows that container's login, which comes from the
  example file's connection URL.
- **The guide stays local.** `SETUP_GUIDE.md` is added to `.git/info/exclude`, so it does not
  show up in `git status`. Check it before sharing it anyway.
- **Dry run.** `--dry-run` still clones (that is a normal `git clone`) and asks Docker whether
  it is running. It installs nothing, creates no files in the project and starts nothing.
- **Destructive actions.** gclone never deletes files, containers or volumes. It reuses an
  existing `gclone-<database>` container. Removal commands are printed in the guide for you
  to run yourself.

To report a vulnerability, see [SECURITY.md](../SECURITY.md).
