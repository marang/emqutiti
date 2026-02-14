# How-to: GoReleaser for emqutiti with Debian, Fedora, and optional Flatpak

This guide is written so you can paste it directly into Codex to prepare a
release setup for **this TUI project** (not a system service).

## Goal

When you push a tag such as `v0.9.0`, CI should:

- build binaries for multiple platforms,
- create `deb` and `rpm` packages,
- publish a GitHub release with assets,
- optionally build a Flatpak bundle artifact.

## Why this differs for a non-service TUI

Since emqutiti is a TUI/CLI app, you usually do **not** need a `systemd`
unit in `deb`/`rpm` packages. Focus on:

- installing the binary to `/usr/bin/emqutiti`,
- optionally shipping README/LICENSE under `/usr/share/doc/emqutiti/`.

## 1) Create `.goreleaser.yaml`

Create `.goreleaser.yaml` in the repository root with this starting point:

```yaml
project_name: emqutiti

before:
  hooks:
    - go mod tidy

builds:
  - id: emqutiti
    main: ./cmd/emqutiti
    binary: emqutiti
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - formats: [tar.gz]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - README.md

checksum:
  name_template: checksums.txt

nfpms:
  - id: linux-packages
    package_name: emqutiti
    builds: [emqutiti]
    formats: [deb, rpm]
    maintainer: "Your Name <you@example.com>"
    description: "Terminal UI for MQTT workflows"
    homepage: "https://github.com/<ORG>/emqutiti"
    license: MIT
    contents:
      - src: ./dist/emqutiti_linux_{{ .Arch }}/emqutiti
        dst: /usr/bin/emqutiti
        file_info:
          mode: 0755
      - src: ./README.md
        dst: /usr/share/doc/emqutiti/README.md
      - src: ./LICENSE
        dst: /usr/share/doc/emqutiti/LICENSE

release:
  github:
    owner: <ORG>
    name: emqutiti
```

### Validate locally

```bash
goreleaser release --snapshot --clean
```

Useful package checks:

```bash
dpkg -c dist/*.deb
rpm -qpl dist/*.rpm
```

## 2) Add a GitHub Actions release workflow

Create `.github/workflows/release.yml`:

```yaml
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Release
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## 3) Optional: Flatpak as an extra CI job

Flatpak is not usually the primary channel for CLI/TUI apps, but you can add
it as an extra CI output or for a later Flathub submission.

For this repository:
- `flatpak/io.github.marang.Emqutiti.yml` is used by CI to bundle a local binary.
- `flatpak/io.github.marang.Emqutiti.flathub.yml` is the Flathub-oriented
  source-build manifest.
- Regenerate Go module sources for Flathub with
  `flatpak/scripts/update-go-sources.sh` (updates `flatpak/go-mod-sources.yml`
  and `flatpak/modules.txt`).

### Example manifest `flatpak/io.github.marang.Emqutiti.yml`

```yaml
app-id: io.github.marang.Emqutiti
runtime: org.freedesktop.Platform
runtime-version: "24.08"
sdk: org.freedesktop.Sdk
command: emqutiti

finish-args:
  - --share=network
  - --filesystem=home

modules:
  - name: emqutiti
    buildsystem: simple
    build-commands:
      - install -D emqutiti /app/bin/emqutiti
    sources:
      - type: dir
        path: .
```

### Optional Flatpak job for `.github/workflows/release.yml`

```yaml
  flatpak:
    runs-on: ubuntu-latest
    needs: goreleaser
    steps:
      - uses: actions/checkout@v5

      - name: Install tools
        run: |
          sudo apt-get update
          sudo apt-get install -y flatpak flatpak-builder
          flatpak remote-add --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo
          flatpak install -y --noninteractive flathub org.freedesktop.Platform//24.08 org.freedesktop.Sdk//24.08

      - name: Build
        run: |
          flatpak-builder --force-clean --repo=repo builddir flatpak/io.github.marang.Emqutiti.yml

      - name: Bundle
        run: |
          flatpak build-bundle repo emqutiti.flatpak io.github.marang.Emqutiti --runtime-repo=https://flathub.org/repo/flathub.flatpakrepo

      - uses: actions/upload-artifact@v4
        with:
          name: emqutiti-flatpak
          path: emqutiti.flatpak

      - name: Upload Flatpak to GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: emqutiti.flatpak
```

## 4) Codex prompt you can reuse

Use this prompt in Codex when you want it to generate the setup automatically:

```text
Please set up GoReleaser in this repository for emqutiti.
Requirements:
- TUI/CLI app, no systemd service
- Builds for linux/darwin/windows (amd64/arm64)
- nFPM packages: deb and rpm
- GitHub Release on v* tags
- Optional Flatpak CI job as an artifact

Please:
1) Create .goreleaser.yaml at repo root,
2) Create .github/workflows/release.yml,
3) Use cmd/emqutiti as the build entrypoint,
4) Install binary in deb/rpm to /usr/bin/emqutiti,
5) Briefly update the README releasing section,
6) Run go test ./... and go vet ./...,
7) End with a summary of changed files.
```

## 5) Practical notes

- Existing AUR automation can remain in place.
- Flatpak is optional; for CLI/TUI apps, `deb`, `rpm`, `go install`, and AUR
  are often the more practical distribution channels.
