#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 3 ]]; then
  echo "usage: $0 VERSION [ARCH] [BINARY]" >&2
  exit 2
fi

version="${1#v}"
arch="${2:-amd64}"
binary="${3:-build/bin/sqlite-explorer}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid RPM version: $version" >&2
  exit 2
fi
case "$arch" in
  amd64) rpm_arch="x86_64" ;;
  arm64) rpm_arch="aarch64" ;;
  *)
    echo "unsupported RPM architecture: $arch" >&2
    exit 2
    ;;
esac
if [[ ! -x "$binary" ]]; then
  echo "application binary is missing or not executable: $binary" >&2
  exit 1
fi
if ! command -v rpmbuild >/dev/null 2>&1; then
  echo "rpmbuild is required to build the RPM package" >&2
  exit 1
fi

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="$project_root/build/release"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

top_dir="$work_dir/rpmbuild"
mkdir -p "$top_dir"/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS,TMP}
changelog_date="$(LC_ALL=C date -u '+%a %b %d %Y')"
install -m755 "$binary" "$top_dir/SOURCES/sqlite-explorer"
install -m644 "$project_root/build/appicon.png" \
  "$top_dir/SOURCES/sqlite-explorer.png"
install -m644 "$project_root/packaging/linux/sqlite-explorer.desktop" \
  "$top_dir/SOURCES/sqlite-explorer.desktop"
install -m644 "$project_root/LICENSE" "$top_dir/SOURCES/LICENSE"

spec_path="$top_dir/SPECS/sqlite-explorer.spec"
cat > "$spec_path" <<EOF
Name:           sqlite-explorer
Version:        $version
Release:        1
Summary:        Local desktop browser and editor for SQLite databases
License:        MIT
URL:            https://github.com/fclaude/sqlite-explorer
Source0:        sqlite-explorer
Source1:        sqlite-explorer.png
Source2:        sqlite-explorer.desktop
Source3:        LICENSE
Requires:       gtk3
Requires:       webkit2gtk4.1
ExclusiveArch:  x86_64 aarch64

%description
Browse schemas and table data, run validated read-only SQL, edit rows,
inspect table statistics, and export results to CSV.

%prep

%build

%install
install -Dm0755 %{SOURCE0} %{buildroot}%{_bindir}/sqlite-explorer
install -Dm0644 %{SOURCE1} %{buildroot}%{_datadir}/pixmaps/sqlite-explorer.png
install -Dm0644 %{SOURCE2} %{buildroot}%{_datadir}/applications/sqlite-explorer.desktop
install -Dm0644 %{SOURCE3} %{buildroot}%{_datadir}/licenses/%{name}/LICENSE

%files
%{_bindir}/sqlite-explorer
%{_datadir}/pixmaps/sqlite-explorer.png
%{_datadir}/applications/sqlite-explorer.desktop
%license %{_datadir}/licenses/%{name}/LICENSE

%changelog
* $changelog_date SQLite Explorer maintainers <fclaude@recoded.cl> - $version-1
- Package the upstream desktop application.
EOF

rpmbuild -bb \
  --define "_topdir $top_dir" \
  --define "_tmppath $top_dir/TMP" \
  --target "$rpm_arch" \
  "$spec_path"

rpm_path="$top_dir/RPMS/$rpm_arch/sqlite-explorer-${version}-1.${rpm_arch}.rpm"
if [[ ! -f "$rpm_path" ]]; then
  echo "expected RPM was not created: $rpm_path" >&2
  exit 1
fi

mkdir -p "$output_dir"
install -m644 "$rpm_path" "$output_dir/$(basename "$rpm_path")"
echo "Created $output_dir/$(basename "$rpm_path")"
