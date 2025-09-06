# Repo Guidelines

## Quick Reference
- **Scope:** This file applies to the entire repository. A nested
  `AGENTS.md` overrides these rules for files in its directory tree.
- **Formatting:** Run `gofmt -w` on modified Go files and favor idiomatic Go
  patterns.
- **Checks:** Execute `go vet ./...` and `go test ./...` before committing.
  Run `go mod tidy` when dependencies change.
- **Tasks:** Use the `Makefile` for common workflows:
  `make build` compiles the app, `make test` runs vet and tests,
  `make proto` regenerates gRPC code, and `make tape` records demo
  sessions.
- **Artifacts:** Avoid committing binary files such as GIFs. Generate them
  locally from `.tape` recordings instead.
- **Commits:** Keep messages short, wrap lines at 72 characters, and summarize
  changes and test results in pull requests.
- **Docs:** Keep `README.md`, `TODO.md`, `AGENTS.md`, and `help/help.md` in
  sync. Docs live under `docs/`; use short sections and bullet lists so
  they are easy to skim and mention key shortcuts.
- **Key directories:** `cmd/` contains the CLI entry point, `ui/` holds TUI
  components, and `docs/` stores user docs.
- **Pitfalls:** Keyboard shortcuts bound to letters can interfere with text
  entry—prefer `Ctrl` combinations. `ExampleSet_manual` in
  `keyring_util_test.go` requires a real keyring and is skipped by default.

## Agent Notes
The TUI runs fullscreen with colorful borders. Press `Ctrl+B` to open the broker manager to add, edit, or delete MQTT profiles. Passwords are stored securely using the system keyring. Publish messages with `Ctrl+S` or use `Ctrl+E` to retain them, when the message field is focused. History labels retained messages. Use the `--import`/`-i` flag to launch an interactive wizard for CSV bulk publishing and select a connection with `--profile` or `-p`. The wizard lets you rename columns when mapping them to JSON fields. Leaving a mapping blank keeps the original column name. The importer code lives in the main package and runs via these flags.
Press `Ctrl+D` from any screen to exit the program.
Press `Ctrl+L` from any screen to open the log viewer; press `Esc` to return.
Scroll with `Ctrl+Up`/`Ctrl+Down` or `Ctrl+K`/`Ctrl+J`. In history,
`a` archives messages and `Delete` removes them.

### Recent Experience
- Keyboard shortcuts bound to plain letters can interfere with text entry. Use `Ctrl` combinations for global actions.
- Provide both keyboard and mouse interaction for lists and chips to keep the UI consistent.
- Favor multi-line text areas where users might paste formatted data.
- Always consider usability and look for ways to improve it.

### UI Guidelines
- Use the `LegendBox` helper for all boxed sections.
- The box helpers have been simplified into a single `LegendBox` function that
  accepts a border color and optional height. Use this function directly rather
  than maintaining multiple wrapper variants.
- Highlight the selected box using the focused style (pink).
- Present keyboard shortcuts consistently across views and ensure they behave the same everywhere.
- Keep functions small and comment any exported ones for clarity.

### Form Utilities
- Shared form behavior lives in `ui/form*.go`.
- Embed the `ui.Form` type to manage focus with `CycleFocus` and `ApplyFocus`.
- `ui.NewTextField`, `ui.NewSelectField`, `ui.NewSuggestField`, and `ui.NewCheckField`
  build common inputs.
- These helpers optionally call `setReadOnly` when values come from the
  environment.
- New UI code should reuse this logic instead of writing custom forms.

## Test Info
`ExampleSet_manual` in `keyring_util_test.go` requires a real keyring. It does not
run during `go test ./...` and can be executed manually if needed:

```bash
go test -run ExampleSet_manual -tags manual
```

## Maintenance
Keep `README.md`, `TODO.md`, `AGENTS.md`, and `help/help.md` in sync when changes are made to the project or development workflow.

## Dependencies
When adding or updating third-party packages, always consult the latest
documentation for each dependency to ensure deprecated APIs are avoided.
Replace outdated calls with the recommended alternatives before committing
changes.

## Contribution Best Practices
- Create topic branches off `main` and keep pull requests focused.
- Describe the problem and solution clearly in commit messages.
- Keep commits small and avoid mixing unrelated changes.
- Record TUI demos with `vhs` and keep the `.tape` files under `docs/`.
- Rebuild GIF previews with `make tape`, which runs `docs/scripts/record_tapes.sh`
  inside a helper container built from `docs/scripts/Dockerfile.vhs`.
- Run the script directly if `vhs` is already on your `PATH`, but do not commit
  generated GIFs.
