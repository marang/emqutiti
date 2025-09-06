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
- [ ] Debian/Ubuntu package
- [ ] Fedora RPM
- [ ] Homebrew formula for macOS users

## Documentation
- [x] Include a VHS GIF in the README
- [x] Document GIF generation using `vhs`
- [x] Provide a Dockerfile for tape recording to avoid host installs
- [x] Add screenshots to the README

Remember to update this file as tasks are completed.
