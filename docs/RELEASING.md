# Releasing SQLite Explorer

The release workflow builds a Linux amd64 tarball, Debian package, and Fedora-compatible RPM, plus a universal macOS application for Intel and Apple Silicon. The macOS application uses an ad-hoc signature and is placed in a DMG without Apple notarization. No repository secrets are required.

## macOS trust model

An ad-hoc signature seals the application but does not identify a trusted Apple developer. A downloaded build may therefore be blocked on first launch. Users who trust the release can try to open it once, then choose **Open Anyway** under **System Settings → Privacy & Security**.

Do not describe this artifact as Developer ID-signed or notarized. If a Developer ID certificate becomes available later, restore certificate signing and Apple notarization before removing the `unsigned` suffix from its filename.

## Release checklist

1. Confirm CI passes on Linux and macOS.
2. Run `go test ./... -race -count=1`, `go vet ./...`, `govulncheck ./...`, `npm test -- --run`, `npm run build`, and `npm audit --audit-level=low`.
3. Confirm `info.productVersion` in `wails.json` and `version` in `frontend/package.json` are the same release version.
4. Review user-facing behavior and update the README or release notes.
5. Confirm the repository and all reachable Git history pass the secret scan. A deleted secret remains compromised until history is rewritten and the credential is rotated.
6. Create and push an annotated tag matching the configured version, for example `git tag -a v0.1.2 -m "SQLite Explorer v0.1.2"`.

Pushing the tag runs `.github/workflows/release.yml`. The workflow deliberately fails before publishing when the tag/version mismatch or a package cannot be validated.

## Artifact expectations

- `sqlite-explorer_VERSION_amd64.deb`: Debian 12 / Ubuntu 22.04+ package with GTK3 and WebKit2GTK 4.1 runtime dependencies.
- `sqlite-explorer-VERSION-1.x86_64.rpm`: Fedora-compatible RPM with `gtk3` and `webkit2gtk4.1` runtime dependencies.
- `sqlite-explorer-VERSION-linux-amd64.tar.gz`: portable binary archive for modern Linux distributions with compatible runtime libraries installed.
- `sqlite-explorer-VERSION.src.tar.gz` and `sqlite-explorer-VERSION.src.zip`: snapshots generated from the exact tagged Git tree.
- `SQLite-Explorer-VERSION-universal-unsigned.dmg`: ad-hoc-signed, unnotarized macOS 12+ application for Intel and Apple Silicon.
- `SHA256SUMS`: checksums generated from the exact uploaded artifacts.
