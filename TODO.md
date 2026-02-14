# Project Roadmap

This file tracks planned improvements for Emqutiti.

## UI
- [x] Split view logic into multiple files for easier maintenance
- [x] Responsive layout via `tea.WindowSizeMsg` and `lipgloss`
- [ ] Refine vertical stacking on very narrow terminals

## Connection Management
- [x] Secure credentials using the OS keyring
- [x] Full CRUD operations for broker profiles
 - [x] TLS/SSL certificate management

## Importer
- [x] Interactive wizard for publishing CSV files
- [ ] Persist import wizard settings for reuse

## Testing
- [ ] Verify layout across a wide range of terminal sizes

## Packaging
- [x] Provide a `PKGBUILD` for Arch Linux
- [x] Debian/Ubuntu package (`.deb` via GoReleaser)
- [x] Fedora RPM (`.rpm` via GoReleaser)
- [ ] Homebrew formula for macOS users
- [ ] Flatpak package

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
