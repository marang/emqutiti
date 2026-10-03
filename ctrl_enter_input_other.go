//go:build !linux

package emqutiti

import tea "github.com/charmbracelet/bubbletea"

// The framing adapter is verified on Linux only. In particular, preserve
// Windows' native console reader; Tea 1.3.10 drops Ctrl on its Enter KeyMsg.
func ctrlEnterInputOptions() ([]tea.ProgramOption, func()) { return nil, func() {} }
