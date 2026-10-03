# LAB-226 Implementation

- Issue: LAB-226, TUI/design corrections.
- Branch: `feature/lab-226-tui-corrections`.
- Historical findings: [tui-design-review.md](tui-design-review.md), preserved unchanged.
- Scope: bounded behavior/layout corrections using existing dependencies; no redesign.

## Checklist

- [x] Implement asynchronous publish/subscribe/unsubscribe commands and typed results.
- [x] Report publish pending/error state; record per-target API success only.
- [x] Preserve editable drafts and request payload/target/retain/connection snapshots;
      ignore stale connection results and suppress duplicate pending sends.
- [x] Confirm payload deletion against stable IDs and use rendered list-row hitboxes.
- [x] Separate list-filter text entry from item actions and topic-input help from chip actions.
- [x] Fit and scroll broker forms, history filters/details and long confirmations;
      keep focused content and dialog decisions reachable.
- [x] Reserve exactly one global header row at all widths and two client context-help rows.
- [x] Route history failures through application logs/history and trace runtime
      reports through typed results consumed globally, including after mode changes.
- [x] Add focused regression tests and update README, TODO, AGENTS and in-app help.
- [x] Complete independent review of the integrated changes; reproduce and close its findings.
- [x] Complete constrained-terminal visual checks, including ANSI and no-color output.
- [x] Approve the chip experiment after visual comparison.
- [x] Run and record full validation: `go vet ./...`, `go test ./...` and the race suite.

## Behavior Contracts

- The TUI publishes at QoS 0. Token/API success is not subscriber-delivery confirmation.
  Failed targets produce errors, not publish history or success pulses; successful
  targets in a partially failed batch remain recorded.
- Pending feedback uses dispatch-time targets even when selection or draft changes.
  MQTT workers return results without mutating UI state; draft edits survive completion.
- Delete and right-click payload actions confirm the chosen stable entry, not a later
  index. Filter editing and blank clicks cannot execute item actions.
- Broker forms reveal focused fields and keep save/cancel visible. History filter
  hitboxes include centering, border/padding, blank/suggestion rows and scroll offsets.
  Full detail payloads and long confirmation text wrap and remain scrollable.
- `OverlayHelp` contributes one header row, including narrow widths. Client context
  help remains two lines above the scrolling body; input hints describe add/subscribe,
  while chip hints describe toggling subscription/publish and removal.

## Chip Experiment

- Chosen implementation: underline subscribed topic names on ANSI terminals;
  no-color suffixes `[sub]`, `[pub]`, `[sub,pub]`, `[off]` distinguish independent states.
- Compared underline and a mark in the existing top border. Focus overrides the
  border cue, and pulse phases make publish-only/both identical. Underline remains
  distinguishable without changing geometry. Apply it on the chip's outer style
  so nested ANSI resets cannot remove the publish background.
- Preserve existing chip border geometry and spacing. No hover recoloring, `r`/`rw`
  prefixes, reserved animation column or new chip border shapes.

### Approved Chip-State Follow-Up

- Cyan underline means subscribed. Complete pink fill means an effective publish target,
  including the selected fallback when no explicit `p` marks exist.
- Neutral borders carry no subscription/publish state; selection has a pink
  outline. Border flashes preserve fill and underline geometry.
- Remove `pink=focus` from the chip legend. The Message title distinguishes
  selected/marked targets; chip hover uses the same effective destinations as sending.
- No-color suffixes include the fallback's effective state without persisting a
  publish mark. Other section boxes retain the existing pink focus style.
- History timestamps and input hints use adaptive medium gray; logs and help
  retain higher contrast. Selected rows adapt their background as well.

## Integration Review

- Independent review found reopening details bypassed wrapping, trace reports lacked
  a UI consumer, and root Tab routing discarded topic searches. Fixed all three with
  root navigation/detail tests and trace subscription-failure/report lifecycle tests.
- Final review found a second history mouse path bypassed the validated hit test;
  removed its unconditional selection and added root border/Shift-click range tests.
- Broker searches also own Enter/Tab/Esc; filtered actions resolve the original
  profile, and manager actions wrap within the same terminal budget.
- Root topic-delete confirmations resolve the confirmed name at acceptance,
  so an asynchronous subscription-result resort cannot remove a different topic.
- Removed obsolete topic/detail size helpers after moving sizing into components.

## Initial LAB-226 Validation

This records the initial LAB-226 pass before the resize, modified-key and History
follow-ups. Final integrated review and release evidence belongs in
`lab-232-233-release.md`.

- Final `make test` and `go test -race ./...` passed, including the root mouse
  routing and asynchronous topic-confirmation resort regressions. One race process
  ended with SIGTERM without a diagnostic; its isolated full rerun passed.
- All independent review findings closed; root regression covers the previously
  bypassed selection path. No further P1/P2 findings in the final targeted pass.
- Build passed: `go build -o /tmp/emqutiti-lab226-qbwce8/emqutiti ./cmd/emqutiti`.
- Live isolated PTY smoke passed: broker list, add form, text entry, Tab, resize
  80x24 -> 40x16 with draft/footer preserved, Esc and Ctrl+D; exit 0.
- 217 offline ANSI captures, including populated running/planned/stopped trace
  fixtures, filtering, expanded selects and validation errors. Four sizes: 120x40,
  80x24, 60x20, 40x16; dark/light/no-color. Opt-in capture tests use
  `EMQUTITI_REVIEW_CAPTURE_DIR` with `TestLAB226` and `TestTraceAcceptanceLayout`.
- Captures/rasterizations are local under `/tmp/emqutiti-lab226-qbwce8`, not committed.
  Raster samples inspected with pyte/Pillow and JetBrainsMono. Original joint-review
  captures remain unchanged. No live external broker or manual system-keyring test.
- At this stage, Dependabot changes remained separate; no commit, push or release
  had been made.

## API References

- Existing termenv version unchanged, promoted to a direct dependency for no-color
  profile checks. [termenv profiles](https://github.com/muesli/termenv#usage).
- [Lip Gloss UnderlineSpaces](https://pkg.go.dev/github.com/charmbracelet/lipgloss#Style.UnderlineSpaces)
  preserves unadorned padding while underlining the topic name.
