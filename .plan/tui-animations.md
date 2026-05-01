# TUI animations plan

## Summary

- Keep animations subtle, short, and ASCII-only.
- Do not add a global config or environment flag in this step.
- Add one central animation tick path instead of scattered timers.
- Animate topic chips on state changes and publish actions.
- Smooth the History scroll indicator for keyboard/programmatic movement;
  mouse wheel scrolling stays direct.

## Key changes

- Add root-level animation state and an internal `animationTickMsg`.
- Schedule ticks only while an animation is active.
- Add topic chip style pulses so hitboxes do not shift.
- Add short publish/subscribe/inactive pulses for the affected topic chips.
- Add a short History title pulse for new published or received messages.
- Smooth the History box scroll indicator for keyboard movement.
- Add a small ASCII connecting frame to the client status line.

## Test plan

- Topic pulse starts for topic toggle and publish targets.
- Publish fallback target pulses when no explicit publish topic exists.
- Topic chip labels keep topic names readable and do not use `r`, `w`, `rw`,
  or `x` prefixes.
- History keyboard scrolling starts smooth scroll state.
- Mouse wheel history scrolling does not start smooth scroll state.
- Animation ticks expire active pulses.
- `go vet ./...` and `go test ./...` pass.

## Assumptions

- Defaults chosen: subtle, ASCII, topic animation on change, keyboard-only
  smooth scroll, no reduced-motion option yet.
- Animation is visual only and must not change MQTT behavior, persistence, or
  history data semantics.
