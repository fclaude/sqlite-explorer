#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 3 ]]; then
  echo "usage: $0 VERSION [ARCH] [BINARY]" >&2
  exit 2
fi

version="${1#v}"
arch="${2:-amd64}"
binary="${3:-build/bin/sqlite-explorer}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([~-][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid package version: $version" >&2
  exit 2
fi
case "$arch" in
  amd64|arm64) ;;
  *)
    echo "unsupported Debian architecture: $arch" >&2
    exit 2
    ;;
esac
if [[ ! -x "$binary" ]]; then
  echo "application binary is missing or not executable: $binary" >&2
  exit 1
fi
if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "dpkg-deb is required to build the Debian package" >&2
  exit 1
fi

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="$project_root/build/release"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

package_root="$work_dir/sqlite-explorer_${version}_${arch}"
install -Dm755 "$binary" "$package_root/usr/bin/sqlite-explorer"
install -Dm644 "$project_root/build/appicon.png" \
  "$package_root/usr/share/pixmaps/sqlite-explorer.png"
install -Dm644 "$project_root/packaging/linux/sqlite-explorer.desktop" \
  "$package_root/usr/share/applications/sqlite-explorer.desktop"
install -Dm644 "$project_root/LICENSE" \
  "$package_root/usr/share/doc/sqlite-explorer/copyright"

installed_size="$(du -sk "$package_root/usr" | cut -f1)"
mkdir -p "$package_root/DEBIAN"
cat > "$package_root/DEBIAN/control" <<EOF
Package: sqlite-explorer
Version: $version
Section: utils
Priority: optional
Architecture: $arch
Installed-Size: $installed_size
Maintainer: SQLite Explorer maintainers <fclaude@recoded.cl>
Depends: libgtk-3-0, libwebkit2gtk-4.1-0
Description: Local desktop browser and editor for SQLite databases
 Browse schemas and table data, run validated read-only SQL, edit rows,
 inspect table statistics, and export results to CSV.
EOF

mkdir -p "$output_dir"
deb_path="$output_dir/sqlite-explorer_${version}_${arch}.deb"
dpkg-deb --build --root-owner-group "$package_root" "$deb_path"

archive_root="$work_dir/sqlite-explorer-${version}-linux-${arch}"
mkdir -p "$archive_root"
install -m755 "$binary" "$archive_root/sqlite-explorer"
install -m644 "$project_root/README.md" "$archive_root/README.md"
install -m644 "$project_root/LICENSE" "$archive_root/LICENSE"
install -m644 "$project_root/build/appicon.png" "$archive_root/sqlite-explorer.png"
install -m644 "$project_root/packaging/linux/sqlite-explorer.desktop" \
  "$archive_root/sqlite-explorer.desktop"
tar -C "$work_dir" -czf \
  "$output_dir/sqlite-explorer-${version}-linux-${arch}.tar.gz" \
  "$(basename "$archive_root")"

echo "Created $deb_path"
echo "Created $output_dir/sqlite-explorer-${version}-linux-${arch}.tar.gz"
