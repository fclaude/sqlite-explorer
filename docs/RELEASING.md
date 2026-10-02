# Releasing

Pushing a tag `vX.Y.Z` runs [`.github/workflows/release.yml`](../.github/workflows/release.yml). It builds the Linux and macOS packages, writes `SHA256SUMS`, and publishes a GitHub release with the notes from `docs/release-notes/vX.Y.Z.md`. It stops before publishing if the tag does not match the app version, the notes file is missing, or a package fails validation. No repository secrets are needed.

## Checklist

1. Set the version in `frontend/package.json`, `frontend/package-lock.json`, `wails.json`, and `frontend/package.json.md5` (Wails compares this hash to decide whether to reinstall dependencies):

   ```bash
   (cd frontend && npm version X.Y.Z --no-git-tag-version)
   jq '.info.productVersion = "X.Y.Z"' wails.json > wails.json.tmp && mv wails.json.tmp wails.json
   printf %s "$(md5sum frontend/package.json | cut -d' ' -f1)" > frontend/package.json.md5
   ```

2. Write `docs/release-notes/vX.Y.Z.md`: what users will notice, grouped under **New**, **Fixed**, and **Other**. The workflow publishes this file as the release notes, and CI fails while the current version has none.
3. Update the README for any user-facing change.
4. Run the checks locally and confirm CI passes on `main`: `make test frontend-test vet audit`, plus `go test ./... -race -count=1`.
5. Confirm no secrets are in the repository or its history. A secret that was ever committed stays compromised until it is rotated.
6. Commit, then tag and push:

   ```bash
   git tag -a vX.Y.Z -m "SQLite Explorer vX.Y.Z"
   git push origin main vX.Y.Z
   ```

## Artifacts

| File | Contents |
|---|---|
| `sqlite-explorer_VERSION_amd64.deb` | Debian 12 / Ubuntu 22.04+ package; depends on GTK3 and WebKit2GTK 4.1 |
| `sqlite-explorer-VERSION-1.x86_64.rpm` | Fedora package; depends on `gtk3` and `webkit2gtk4.1` |
| `sqlite-explorer-VERSION-linux-amd64.tar.gz` | Portable Linux binary |
| `SQLite-Explorer-VERSION-universal-unsigned.dmg` | macOS 12+ app for Intel and Apple Silicon, ad-hoc signed, not notarized |
| `sqlite-explorer-VERSION.src.tar.gz`, `.src.zip` | Source of the tagged commit |
| `SHA256SUMS` | Checksums of the files above |

## macOS signing

An ad-hoc signature seals the app but does not identify a trusted Apple developer, so macOS may block the first launch until the user chooses **Open Anyway** under **System Settings → Privacy & Security**. Do not describe the app as Developer ID-signed or notarized. If a Developer ID certificate becomes available, add certificate signing and notarization to the workflow before dropping `unsigned` from the file name.
