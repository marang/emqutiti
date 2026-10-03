# TUI Tickets And Release

## Scope

- LAB-226: finish and review the existing TUI corrections and chip/contrast follow-ups.
- LAB-232: shared bottom-border dragging for Topics, Message and History,
  bounded heights, Escape rollback, keyboard resize/reset and preserved drafts/focus.
- LAB-233: unequivocal Ctrl+Enter publishing, retaining Enter and Ctrl+S semantics.
- Review, merge to main, then tag and verify the automated GitHub release.

## Checklist

- [x] LAB-226 corrections implemented and validated locally.
- [x] LAB-232 implementation and interaction/layout tests.
- [x] LAB-233 implementation, protocol tests and retained adapter review fixes.
- [x] Update shared docs and complete independent review.
- [x] Run vet, full tests, race suite, build and visual/live terminal checks.
- [x] Commit and merge the reviewed work into main; push main.
- [x] Close the three Linear tickets with implementation/verification evidence.
- [x] Create release tag and verify GitHub release workflows/assets.

## Resize Decisions

- Shared minimum/maximum height constraints apply to dragging, keyboard resizing,
  restored layouts and terminal resize. Topics keeps its existing chip-row unit.
- Escape, mode changes and terminal resizing cancel the captured drag; accepted
  heights survive view changes and use the existing per-profile persistence.
- Input and publish shortcuts are captured while dragging. Ctrl+D still exits.
- Current focused resize/reset keys: Ctrl+Shift+Up/Down and Ctrl+R.

## Chip Follow-Up

- Complete pink fill includes the publish chip border cells, with unchanged geometry.
- Selection uses a pink outline; subscribe uses an independent cyan underline.
- Terminals without underline-color support still retain a normal underline;
  no-color mode keeps the existing semantic suffixes.

## Modified Keys

- Linux TTY adapter frames input before Bubble Tea 1.3.10, then converts only
  exact Ctrl+Enter CSI-u/modifyOtherKeys and Shift+Space CSI-u signatures.
- Ordinary Enter, text, bracketed paste, unrelated keys and native non-Linux
  input remain unchanged. Protocol modes and terminal configuration stay untouched.
- Ctrl+S remains portable; explicit compatibility and optional terminal mapping
  live in `help/help.md`. Repeat/release events never publish again.
- Tests cover the actual Tea parser, every read split, negative encodings,
  paste, input ordering, Linux PTY/raw restoration and root dispatch/drag capture.
- Independent review must clear legacy-key framing and malformed-prefix paste
  regressions before merge. Pasted key encodings must never publish.

## History Follow-Up

- Shift+Click adds ranges without clearing existing marks; Space/Shift+Space
  toggles the current row independently.
- Keyboard range contraction preserves earlier marks. Filtered incoming items
  and copy-log refreshes retain the selection by item identity.
- Explicit Shift+Space input still inserts ordinary spaces in editing fields.
- Independent interaction review and focused normal/race regressions passed.

## Review Fixes

- Restore original-height viewport content before Escape reapplies the scroll anchor.
- Reassert History's minimum when expanded help/filter chrome changes.
- Persist resolved history log text so MQTT failures survive filters and reloads.
- Consume headless trace reports and retain the first failure, without mistaking
  worker startup for completion. Cancellation/end deadlines unblock stalled
  subscriptions and drain reports before returning.
- Headless lifecycle fixes passed independent review, loopback integration tests
  and repeated race runs.
- Fix GoReleaser/Flatpak version injection to use the actual cmd package symbol.
  The previous default flags reproduced `--version=dev`; explicit package flags
  are checked with a release-version build before publishing.
- Preserve SS3, legacy CSI-O/Linux-console keys and their Alt forms; recognize
  paste boundaries independently of malformed control prefixes. Keep UTF-8
  lookahead/continuations intact, including timeout boundaries.

## Integrated Validation

- Final stable adapter SHA-256:
  `4d5bc8d584f0ade9dea561ca7c900105b62d3feecef621932c7d7c0746e1b39e`.
- `make test` passed: vet and all packages; root regressions took 41.706s.
- `go test -race ./...` passed; root regressions took 54.175s.
- Formatting, `git diff --check` and `go mod tidy` passed. Only existing
  dependencies were promoted to direct use in this initial integrated pass.
- `make build`, Darwin arm64 and Windows amd64 cross-builds passed.
- Live isolated PTY smoke passed on the final build: broker list/form, typing,
  Tab, 80x24 -> 40x16 resize with draft/footer intact, Esc and Ctrl+D; exit 0.
- 287 local ANSI captures across 120x40, 80x24, 60x20 and 40x16; themed
  dark/light/no-color views plus resize/history/chip fixtures. Inspected raster
  samples include complete pink chip fill, cyan underlines and medium-gray dates.
  34 local rasterizations; no image or binary artifacts committed.
- Independent final packaging review passed, with release/rc version linker probes.
- Independent UI/History, core/headless and exact-hash input-adapter reviews
  cleared all retained P1/P2 findings. Adapter verification additionally passed
  the pinned 142-key Tea corpus plus Alt variants, every read split, 5,440
  malformed-prefix paste-boundary cases, root publish-safety checks and race x3.
- Live external brokers and manual system-keyring examples were not exercised;
  MQTT/proxy/TLS integration uses isolated loopback fixtures. Linux PTY tests
  validate emitted bytes/raw restoration, not a physical terminal's key mapping.

## Pre-Release Security Gate

- The first main push reported five existing GitHub dependency alerts. Upgrade
  gRPC 1.80.0 -> 1.83.2 and its required dependency closure (including x/net
  0.58.0), without a broad dependency refresh or TUI-library migration.
- Replace both deprecated grpc.Dial calls with grpc.NewClient; validate real
  proxy RPCs and status logging, including lazy connection/cancellation behavior.
- A scan with the previous release toolchain (Go 1.26.2) found eight reachable
  standard-library vulnerabilities. Raise the minimum/release Go patch to
  1.26.8, retaining the same Go feature series.
- Patch the additional imported S2 dictionary issue with compress 1.18.7.
  The vulnerable function was not reachable in the scan, but the patch removes
  the remaining package-level finding as well.
- Repeat full checks, independent focused review and vulnerability scanning
  using the exact patched release toolchain before tagging.
- Independent dependency/API review passed. Real proxy status RPC success and
  stalled-handshake deadline tests fill the status-logging coverage gap.
- `GOTOOLCHAIN=go1.26.8 govulncheck ./...` passed with no vulnerabilities found
  at symbol, package or module level; final full normal tests passed (root 43.197s).
- Final exact-toolchain full race tests passed (root 53.552s), as did the build,
  Darwin/Windows cross-builds and isolated live PTY smoke on the updated binary.

## Publication

- TUI implementation: `92c127463d7115577a157f53844ae50ef5451f84`.
- Final reviewed security/release code: `f617a86d692578c68da7eccb6b4846081ed88f3c`.
- Both topic branches were fast-forward merged to main and pushed.
- [Final main CI](https://github.com/marang/emqutiti/actions/runs/37115300521)
  passed; GitHub reports zero open Dependabot alerts.
- LAB-226, LAB-232 and LAB-233 are Done, with implementation/review/test evidence
  recorded in Linear before the release tag was created.
- Annotated tag `v0.7.12` points to the final reviewed code commit.
- [Release workflow](https://github.com/marang/emqutiti/actions/runs/37115520564)
  passed, including GoReleaser and Flatpak; the
  [AUR workflow](https://github.com/marang/emqutiti/actions/runs/37115520409) passed.
- [Published release](https://github.com/marang/emqutiti/releases/tag/v0.7.12)
  is public, non-draft/non-prerelease, with twelve assets and curated notes.
- Downloaded all ten archives/Linux packages and verified every published
  checksum. The Flatpak bundle additionally matches its GitHub SHA-256 digest.
- All six archived platform binaries carry Go 1.26.8, the correct release
  version flag and patched gRPC/x/net/compress dependencies. The downloaded
  Linux amd64 binary prints `0.7.12`; its binary-mode govulncheck is also clean.
- This completion record is a documentation-only follow-up after publication;
  the release tag stays on the reviewed code commit above.
