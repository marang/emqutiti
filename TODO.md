# Project Roadmap

This file tracks planned improvements for Emqutiti.

## UI
- [x] Split view logic into multiple files for easier maintenance
- [x] Responsive layout via `tea.WindowSizeMsg` and `lipgloss`
- [x] LAB-226: asynchronous MQTT results, pending/error feedback and draft snapshots
- [x] LAB-226: stable confirmed payload deletion and rendered-row mouse targets
- [x] LAB-226: scrollable broker/filter/detail/confirmation views and corrected sizing
- [x] LAB-226: separate input/chip context actions and a one-row global header
- [x] Compare and validate ANSI underline/no-color suffix chips without geometry changes
- [x] Match the Topics legend to state styling and explain subscription counts
- [x] Mark client shortcut hints with square brackets and keep narrow layouts readable
- [x] Explain History Shift range selection and selected/current-entry copying in context help
- [x] Preserve chip selection across MQTT results
- [x] Use interior-only pink publish fill, pink selection borders, cyan subscribe underlines and selected/marked titles
- [x] Keep Topics legend and shortcut text at a consistent readable contrast
- [x] Use readable medium-gray dates/input hints and high-contrast History text/help
- [x] LAB-232: drag client panel bottom borders, shared height bounds and keyboard reset
- [x] LAB-233: distinct terminal Ctrl+Enter publish input and input framing
- [x] Publish only on modified Enter: Ctrl normally, Ctrl+Shift retained; macOS Cmd equivalents
- [x] Update publish hints, help and terminal compatibility; remove old send bindings
- [x] Keep publish/retained/newline shortcuts in a responsive Message footer
- [x] Preserve additive History mouse/Space selection and marks during filter refreshes
- [ ] Refine vertical stacking on very narrow terminals

## Connection Management
- [x] Secure credentials using the OS keyring
- [x] Full CRUD operations for broker profiles
 - [x] TLS/SSL certificate management

## Importer
- [x] Interactive wizard for publishing CSV files
- [ ] Persist import wizard settings for reuse

## Testing
- [x] Run vet and unit tests in CI for pull requests and pushes to main
- [ ] Verify layout across a wide range of terminal sizes
- [x] Complete LAB-226 review, visual checks and full validation; track status in
      [.plan/lab-226-implementation.md](.plan/lab-226-implementation.md)
- [x] Complete integrated LAB-232/LAB-233 review and terminal-input validation;
      track release gates in
      [.plan/lab-232-233-release.md](.plan/lab-232-233-release.md)

## Maintenance
- [x] Automate Go module and GitHub Actions update pull requests with Dependabot
- [x] Patch pre-release dependency alerts and validate with the patched Go toolchain

## Packaging
- [x] Provide a `PKGBUILD` for Arch Linux
- [x] Debian/Ubuntu package (`.deb` via GoReleaser)
- [x] Fedora RPM (`.rpm` via GoReleaser)
- [ ] Homebrew formula for macOS users
- [x] Flatpak package

## Documentation
- [x] Include a VHS GIF in the README
- [x] Document GIF generation using `vhs`
- [x] Provide a Dockerfile for tape recording to avoid host installs
- [x] Add screenshots to the README
- [x] Add Codex how-to for GoReleaser + optional Flatpak

## Storage
- [x] Reduce BadgerDB's initial footprint from ~2GB to a maximum of 10MB while
      still allowing the database to grow as needed

Remember to update this file as tasks are completed.
