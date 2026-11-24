# Roadmap

Priorities, in order. Dates are deliberately absent.

**Next**
- Check the Flutter adapter against the real SDK
- Deeper Java checks: a real end-to-end run of Maven and Gradle builds

**Then**
- Rust: start servers (find the port), and hint at system libraries a crate needs (OpenSSL and similar)
- Read versions from `mise.toml`, and reuse an existing Dev Container

**Later, if there is demand**
- An optional LLM planner that only *suggests* steps, always validated and approved by you
- Analyze a repository before cloning it
- Multi-repository workspaces
- A migration report: `gclone analyze --port-to <language>` would list what a project uses (routes, database, packages) and the usual equivalent in another stack, as a checklist. It would not translate code

The things you can help with right now are in the [good first issues](GOOD_FIRST_ISSUES.md).
