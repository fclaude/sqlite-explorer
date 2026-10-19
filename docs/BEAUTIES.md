# Private workstation packages

Gitea's private `beauties/packages` publisher monitors this repository's GitHub
`main` every five minutes. It pins a source commit, validates it, and publishes
signed Fedora RPMs plus a checksum-pinned native Homebrew source formula.
Existing tagged GitHub releases remain separate.

The package embeds its source commit and increasing main-build revision in
`--version`; `beauties-build.json` records source and recipe provenance. Routine
Fedora updates and the managed daily macOS Beauties updater install new builds.
Configuration, backend trust, databases and user sessions stay user-owned.

For operation, acceptance and rollback, see `homelab/docs/beauties.md` in infra.

## Updating and checking an installed build

Use `sudo dnf upgrade sqlite-explorer` on Fedora or
`brew upgrade beauties/tap/sqlite-explorer` on macOS. The managed routine jobs
do this automatically. `sqlite-explorer --version` reports the full source SHA
and main-build revision without opening a desktop window or database.

The Mac app is linked at `~/Applications/SQLite Explorer.app`. Its application
payload is managed by Homebrew; databases, settings and permissions remain
user-owned. Restart an active window normally to use an installed new build.
