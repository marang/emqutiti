# Shortcuts

## Global

| Key | Action |
| --- | ------ |
| Ctrl+D | Exit the program |
| Ctrl+P | Manage payloads |
| Ctrl+T | Manage topics |
| Ctrl+R | Manage traces |
| Ctrl+B | Open broker manager |
| Ctrl+X | Disconnect from broker after confirmation; offers immediate reconnect or opens broker manager |
| Ctrl+S | Publish message |
| Ctrl+E | Publish retained message |
| Ctrl+L | Open log viewer |
| Ctrl+Shift+Up / Ctrl+Shift+Down | Resize panels |
| Ctrl+1 | Toggle topics pane |
| Ctrl+2 | Toggle message pane |
| Ctrl+3 | Toggle history pane |

## Navigation

| Key | Action |
| --- | ------ |
| Esc | Back |
| Tab / Shift+Tab | Cycle focus |
| Up/Down or j/k | Scroll view |
| Left / Right | Switch pane |

## Broker Manager

| Key | Action |
| --- | ------ |
| Enter | Connect or open client |
| Ctrl+X | Disconnect selected profile |
| a | Add profile |
| e | Edit selected profile |
| Delete | Remove selected profile |
| Ctrl+O | Toggle default profile |

## Topics manager

| Key | Action |
| --- | ------ |
| Enter / Space | Toggle subscription |
| p | Toggle publish highlight |
| Delete | Delete topic |

## Payloads manager

| Key | Action |
| --- | ------ |
| Enter | Load payload |
| Delete | Delete payload |

## History

| Key | Action |
| --- | ------ |
| Space | Toggle selection |
| Shift+Up / Shift+Down | Extend selection |
| Ctrl+A | Select all |
| Ctrl+C | Copy selected history entries |
| a | Archive selected messages |
| Delete | Remove selected messages |
| / | Filter messages |
| Ctrl+F | Clear all history filters |
| Enter | View full message |

Retained messages are labeled "(retained)".

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

**Trace**

- `--trace KEY` Trace key name to store messages (e.g., `--trace run1`)
- `--topics LIST` Comma-separated topics to trace (e.g., `--topics "sensors/#"`)
- `--start TIME` Optional RFC3339 start time (e.g., `--start "2025-08-05T11:47:00Z"`)
- `--end TIME` Optional RFC3339 end time (e.g., `--end "2025-08-05T11:49:00Z"`)
