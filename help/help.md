# Shortcuts

## Global

| Key | Action |
| --- | ------ |
| Ctrl+D | Exit the program |
| Ctrl+P | Manage payloads |
| Ctrl+T | Manage topics |
| Alt+R | Manage traces |
| Ctrl+B | Open broker manager |
| Ctrl+X | Disconnect from broker after confirmation; offers immediate reconnect or opens broker manager |
| Ctrl+Enter / macOS: Cmd+Enter | Publish message (client Message editor focused) |
| Ctrl+Shift+Enter / macOS: Cmd+Shift+Enter | Publish retained message (client Message editor focused) |
| Ctrl+L | Open log viewer |
| Ctrl+Shift+Up / Ctrl+Shift+Down | Resize panels |
| Ctrl+R | Reset focused client panel height |
| Ctrl+Up / Ctrl+Down or Ctrl+K / Ctrl+J | Scroll current view |

## Navigation

| Key | Action |
| --- | ------ |
| Esc | Back |
| Tab / Shift+Tab | Cycle focus |
| Up/Down or j/k | Move in focused lists (outside text entry) |
| Left / Right | Switch pane |

The global shortcut header occupies one row at every width. On the client
screen, two context-help rows stay above the scrollable content and describe
the hovered area or keyboard focus. Hover does not change colors.

## Panel heights

- Drag the bottom border of Topics, Message or History with the left mouse button.
  Release applies the height; `Esc` restores the starting height and scroll position.
- `Ctrl+Shift+Up/Down` resizes the focused panel; `Ctrl+R` resets its height.
  Topics grows in chip-row steps. Bounds keep content and local help readable.
- View changes preserve accepted heights. Changing views or resizing the terminal
  during a drag cancels it; the new terminal size clamps heights to fit.
- Dragging preserves focus, drafts, selections and filters. Editing and publish
  shortcuts are suspended until the drag ends; `Ctrl+D` still exits.

## Publishing

- With the client Message editor focused, `Ctrl+Enter` publishes or
  `Ctrl+Shift+Enter` publishes retained. On macOS, use `Cmd+Enter` and
  `Cmd+Shift+Enter`; the Ctrl combinations also work. Publishing goes to all
  marked targets or, if none are marked, the selected topic. Plain `Enter`
  inserts a newline. `Ctrl+S` and `Ctrl+E` no longer publish.
- The Message footer keeps publish, retained and newline shortcuts visible,
  even while another area is focused. It wraps on narrow terminals without
  reducing the configured number of editor rows.
- Publishing is asynchronous. Pending feedback names the original targets;
  repeat sends are ignored until the current batch completes. Keep editing:
  each request uses its original payload, target, retain flag and connection,
  and completion never clears or replaces the draft.
- Only successful MQTT API results create publish history, saved payloads and
  success pulses. Errors appear in history and message context; partial batches
  record only successful targets. Old-connection results are ignored.
- QoS 0 API success does not confirm delivery to subscribers. Errors remain
  available in message context after pending work finishes.
- Dates and empty-input hints use adaptive medium gray; message text and help
  remain higher contrast.

### Terminal shortcut compatibility

Modified Enter is the only publishing shortcut. The input adapter supports
Linux and macOS TTY input; other input platforms currently cannot publish via
these shortcuts. The terminal must send distinct sequences:

| Shortcut | CSI-u | modifyOtherKeys |
| --- | --- | --- |
| Ctrl+Enter | `CSI 13;5u` | `CSI 27;5;13~` |
| Ctrl+Shift+Enter (retained) | `CSI 13;6u` | `CSI 27;6;13~` |
| Cmd+Enter (macOS) | `CSI 13;9u` | - |
| Cmd+Shift+Enter (macOS, retained) | `CSI 13;10u` | - |

CSI-u press events with `:1` also work; repeat (`:2`) and release (`:3`) events
do not send another message. Plain `Enter`, pasted text and unrelated keys
never become publish commands. If a terminal sends the same bytes for ordinary
and modified Enter, both insert a newline until a distinct key mapping is set.
`Ctrl+S` and `Ctrl+E` no longer publish and are not fallbacks.

The app leaves keyboard protocol modes unchanged. A terminal-specific mapping
can emit a distinct key without changing other keys. For example, Kitty:

```conf
map ctrl+enter send_text normal,application \x1b[13;5u
map ctrl+shift+enter send_text normal,application \x1b[13;6u
# macOS equivalents:
map cmd+enter send_text normal,application \x1b[13;9u
map cmd+shift+enter send_text normal,application \x1b[13;10u
```

The encoding follows the [Kitty keyboard protocol](https://sw.kovidgoyal.net/kitty/keyboard-protocol/);
mapping syntax is documented in [Kitty's send_text reference](https://sw.kovidgoyal.net/kitty/conf/#shortcut-kitty.SendText).
This optional terminal mapping is not installed by Emqutiti.

In iTerm2, profile key mappings can use **Send Hex Code** for `Cmd+Enter`
(`0x1b 0x5b 0x31 0x33 0x3b 0x39 0x75`) and `Cmd+Shift+Enter`
(`0x1b 0x5b 0x31 0x33 0x3b 0x31 0x30 0x75`). Replace conflicting terminal
shortcuts if needed; see [iTerm2 key mapping documentation](https://iterm2.com/documentation-preferences-profiles-keys.html).

## Broker Manager

| Key | Action |
| --- | ------ |
| Enter | Connect or open client |
| Ctrl+X | Disconnect selected profile |
| a | Add profile |
| e | Edit selected profile |
| Delete | Remove selected profile |
| Ctrl+O | Toggle default profile |

Broker forms scroll with `Ctrl+Up` / `Ctrl+Down`, `Ctrl+K` / `Ctrl+J` or the
mouse wheel. `Tab` / `Shift+Tab` reveals each field. `Enter` saves and `Esc`
cancels; the footer stays visible. Narrow layouts stack labels and inputs.

## Topic input and chips

- Compact hints put keyboard shortcuts in square brackets, such as `[Enter]`,
  `[p]` and `[Del]`; these are keys, not words to type into the topic input.
- Topic input: type a new topic; `Enter` adds and subscribes. Ordinary letters
  remain text, including `p`. Existing-topic help does not promise an add action.
- Topic chips: `Enter` toggles subscription, `p` toggles the publish target,
  and `Delete` confirms removal. Left-click selects a chip; right-click confirms removal.
- Subscribed names have a cyan underline on capable ANSI terminals;
  chips with a pink-filled interior and dark
  text are the actual publish targets, including the selected fallback when no
  targets are marked with `p`. Fills extend through the inner frame halves;
  outer halves remain unfilled. The selected unfilled chip has a pink outline;
  other unfilled borders are neutral. The legend uses consistently readable text.
  Subscribe underlines and publish fills remain visible
  during border pulses. No-color suffixes are `[sub]`, `[pub]`, `[sub,pub]`
  and `[off]`, including the fallback's effective publish state.
- The Message title distinguishes `publish to (selected)` from `publish to (marked)`.
  Marking the fallback with `p` keeps its fill but switches to marked-target mode;
  changing selection then leaves those marked destinations unchanged.
- Subscribe/unsubscribe results and list refreshes preserve the selected chip,
  including a newer selection made while the MQTT request was pending.
- `Topics: N | subscribed: S` shows the total number of topics and subscriptions,
  not the selection position or number of publish targets.

## Topics manager

| Key | Action |
| --- | ------ |
| Enter / Space | Toggle subscription |
| p | Toggle publish target |
| Delete | Delete topic |

While a manager filter is being edited, typing, `Delete` and `Enter` belong
to the filter rather than item actions. `Esc` clears the filter before leaving.

## Payloads manager

| Key | Action |
| --- | ------ |
| Enter | Load payload |
| Delete | Confirm deletion of selected payload |

Left-click loads the clicked row; right-click confirms deletion of that row.
Blank clicks do nothing. The confirmation keeps the entry's identity if rows
change, so a replacement or a newly added payload cannot be deleted instead.

## History

| Key | Action |
| --- | ------ |
| Space / Shift+Space | Toggle current entry without clearing other marks |
| Shift+Up / Shift+Down | Extend selection |
| Ctrl+A | Select all |
| Ctrl+C | Copy selected history entries |
| a | Archive selected messages |
| Delete | Remove selected messages |
| / | Filter messages |
| Ctrl+F | Clear all history filters |
| Enter | View full message |

With History focused, `Shift+Up` / `Shift+Down` selects a range of entries;
`Shift+Click` adds a range without clearing existing marks. `Space` /
`Shift+Space` toggles just the current entry. `Ctrl+C` copies selected
entries, or the current entry when nothing is selected. MQTT entries include
the topic and full payload; multiple entries are separated by newlines.
Range selection is unavailable in archived history. The client context hint
shows the selection shortcuts when History is hovered or focused.

Most terminals send the same space for `Space` and `Shift+Space`; both toggle
the current History row. Linux/macOS TTY input also accepts the distinct CSI-u
`CSI 32;2u` Shift+Space event. Editing fields keep ordinary spaces as text.

Retained messages are labeled "(retained)".
History dates use adaptive medium gray. Log messages and shortcut hints retain
higher contrast on light and dark backgrounds, including selected entries.

History filters fit the terminal: `Tab` / `Shift+Tab` moves between fields,
`PgUp` / `PgDown`, the mouse wheel or `Ctrl+Up` / `Ctrl+Down` scrolls;
`Enter` applies and `Esc` cancels. Clicking a suggestion selects it.
Full message details wrap the complete payload and scroll with arrows,
`PgUp` / `PgDown` or the mouse wheel. `Ctrl+C` copies the full payload;
`Esc` returns. Focus brings the active history entry into view.

## Confirmations

Long prompts wrap and scroll with `PgUp` / `PgDown`, the mouse wheel or
`Ctrl+Up` / `Ctrl+Down`. The `y` / `n` choices stay visible and clickable;
`Esc` cancels without applying the action.

## Traces manager

| Key | Action |
| --- | ------ |
| a | Add trace |
| Enter | Start or stop trace |
| v | View trace messages |
| Delete | Remove trace |

## Tips

- Set `EMQUTITI_DEFAULT_PASSWORD` to override profile passwords when not loading from env.

## CLI Flags

**General**

- `-i, --import FILE` Launch CSV import wizard with optional file path (e.g., `-i data.csv`)
- `-p, --profile NAME` Connection profile name to use (e.g., `-p local`)
- `-l, --list-profiles` List available connection profiles and exit

**Trace**

- `--trace KEY` Trace key name to store messages (e.g., `--trace run1`)
- `--topics LIST` Comma-separated topics to trace (e.g., `--topics "sensors/#"`)
- `--start TIME` Optional RFC3339 start time (e.g., `--start "2025-08-05T11:47:00Z"`)
- `--end TIME` Optional RFC3339 end time (e.g., `--end "2025-08-05T11:49:00Z"`)
- Omit `-p/--profile` when tracing to pick a connection interactively before starting
