# Export a Dev Container

`gclone export devcontainer [directory]` writes `.devcontainer/devcontainer.json` from what
gclone detected, so people who use GitHub Codespaces or VS Code Dev Containers get the same
setup without running gclone. It asks before writing (`--yes` skips the question), never
overwrites a file that exists, and `--print` (or `--dry-run`) shows the files without writing.

What goes in:

- A **feature** for each stack, at the version the project asks for: Node.js, Go, Java (with
  Maven or Gradle when there is no wrapper) and Python.
- A **`postCreateCommand`** that copies the example `.env` (only when there is no `.env`)
  and installs dependencies, with the same commands `setup` would run (Linux versions).
- **Forwarded ports** for the apps gclone found.
- **Databases** the project needs (PostgreSQL, MySQL, MongoDB, Redis, RabbitMQ) as containers in
  a `docker-compose.yml` next to it. They share one network with your code, so the app reaches
  them on `localhost` and the usual port. Their logins are the image defaults, which gclone
  tells you about, so the variables in your `.env` have to match.
- A few recommended VS Code extensions.

Limits: Flutter is not exported (there is no official feature for it), and the output has
**not been tried against a real Docker engine or Codespaces by gclone**: open it once and check
that it builds. These are ordinary project files, so review and commit them if you want
others to use them.
