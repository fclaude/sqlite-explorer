#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $0 VERSION [GIT_REF]" >&2
  exit 2
fi

version="${1#v}"
git_ref="${2:-HEAD}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid source archive version: $version" >&2
  exit 2
fi
if ! git cat-file -e "${git_ref}^{commit}" 2>/dev/null; then
  echo "git ref does not resolve to a commit: $git_ref" >&2
  exit 1
fi

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="$project_root/build/release"
archive_prefix="sqlite-explorer-${version}/"
tar_path="$output_dir/sqlite-explorer-${version}.src.tar.gz"
zip_path="$output_dir/sqlite-explorer-${version}.src.zip"

mkdir -p "$output_dir"
git -C "$project_root" archive \
  --format=tar \
  --prefix="$archive_prefix" \
  "$git_ref" | gzip -n > "$tar_path"
git -C "$project_root" archive \
  --format=zip \
  --prefix="$archive_prefix" \
  --output="$zip_path" \
  "$git_ref"

echo "Created $tar_path"
echo "Created $zip_path"
