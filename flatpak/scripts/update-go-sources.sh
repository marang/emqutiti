#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

cd "$tmp_dir"
go run github.com/dennwc/flatpak-go-mod@latest "$repo_root"

mv go.mod.yml "$repo_root/flatpak/go-mod-sources.yml"
mv modules.txt "$repo_root/flatpak/modules.txt"
