# Repo Guidelines

## Quick Reference
- **Scope:** This file applies to the entire repository. A nested
  `AGENTS.md` overrides these rules for files in its directory tree.
- **Formatting:** Run `gofmt -w` on modified Go files and favor idiomatic Go
  patterns.
- **Checks:** Execute `go vet ./...` and `go test ./...` before committing.
  Run `go mod tidy` when dependencies change.
- **Release automation:** Tags matching `v*` run `.github/workflows/release.yml`
  (GoReleaser + optional Flatpak) and `.github/workflows/aur.yml`. For the
  Codex how-to used to add this setup, see `docs/howto-codex-goreleaser.md`.
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

## Response Efficiency
- Prefer concise responses by default: 3-6 lines unless the user asks for more.
- Use brief progress updates only for long-running tasks or blockers.
- Avoid repeating command output verbatim; summarize key results.
- In final updates, list only: what changed, what was run, and next action.
- Ask at most one clarifying question when required; otherwise proceed.

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
- Mark keys with `[key]` in compact client hints; present shortcuts consistently
  across views and ensure they behave the same everywhere.
- Keep functions small and comment any exported ones for clarity.

### TUI Contracts
- For publishing, topic actions, payload removal or scrolling changes, read
  `help/help.md` for the user-facing workflow and keep its shortcuts accurate.
- Run MQTT publish/subscribe/unsubscribe waits in `tea.Cmd`; apply typed
  results in the UI update loop. Publish history, saved payloads and success
  pulses follow successful API results per target, not dispatch. QoS 0 API
  success does not establish subscriber delivery.
- Keep request payload/target/retain and broker/client snapshots stable while
  pending; preserve the editable draft on success or failure. Ignore stale
  connection results and suppress repeated current-broker publishes while pending.
- Bind payload-delete confirmations to stable entry IDs and hit-test rendered
  rows, including filtering and pagination; blank clicks leave entries unchanged.
- Route text entry before contextual letter actions. Topic input help describes
  adding/subscribing; chip help describes subscription/publish/removal actions.
  Hover updates context without recoloring; focus remains a separate state.
  Keep range-selection hints scoped to non-archived History, not topic chips.
- `OverlayHelp` adds exactly one global header row at every width. Budget it,
  borders and local footers before sizing content. Keep client context help
  two rows above the scrollable body; wrap and scroll long dialog/detail content
  while keeping decisions reachable.
- Preserve chip border geometry and spacing, with no reserved animation column
  or `r`/`rw` prefixes. Cyan-underline subscribed names independently of complete
  pink publish fill and pink selection borders. Fill effective publish targets,
  including the selected fallback; distinguish selected/marked mode in the
  Message title. Use `[sub]`, `[pub]`, `[sub,pub]`, `[off]` for those effective
  states in no-color mode. List rebuilds preserve client selection by identity
  while managers keep their own pane selection.
  Match the Topics legend to those cues; include wrapped rows in its height
  budget. Other section boxes retain the existing pink keyboard-focus style.
  History dates and input hints use adaptive medium gray; log text and help
  keys/descriptions stay higher contrast on light and dark backgrounds.
  Use the shared panel resize controller for bottom-border dragging and keyboard
  resizing/reset; preserve drafts/focus and suspend editing/publishing during drags.
  Modified-key adapters accept only distinct Linux TTY encodings; keep legacy
  Enter/paste unchanged and Ctrl+S portable. History Shift-click adds ranges;
  Space/Shift+Space toggles one row without clearing other marks.
  See `.plan/lab-226-implementation.md` for the comparison and validation evidence.

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
Dependabot configuration lives in `.github/dependabot.yml`; see the dependency
updates section in `README.md` for the update and review policy.

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
