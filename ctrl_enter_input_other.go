//go:build !linux && !darwin

package emqutiti

import tea "github.com/charmbracelet/bubbletea"

// Platforms outside Linux/macOS keep Tea's default input. In particular, preserve
// Windows' native console reader; Tea 1.3.10 drops Ctrl on its Enter KeyMsg.
func ctrlEnterInputOptions() ([]tea.ProgramOption, func()) { return nil, func() {} }
