package traces

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// ReportMsg carries an asynchronous runtime result for one trace instance.
// A nil Err marks completion without a pending error.
type ReportMsg struct {
	Key    string
	Tracer *Tracer
	Err    error
}

func listenTraceReports(key string, tracer *Tracer) tea.Cmd {
	if tracer == nil || tracer.report == nil || tracer.done == nil {
		return nil
	}
	reports, done := tracer.report, tracer.done
	return func() tea.Msg {
		var err error
		select {
		case err = <-reports:
		case <-done:
			// Fatal errors can be queued before completion. Do not let select
			// choosing the completed channel discard that final report.
			select {
			case err = <-reports:
			default:
			}
		}
		return ReportMsg{Key: key, Tracer: tracer, Err: err}
	}
}

// HandleReport logs current-instance runtime errors and waits for the next result.
// The host should route ReportMsg here regardless of the active application mode.
func (t *Component) HandleReport(msg ReportMsg) tea.Cmd {
	index := t.traceIndex(msg.Key)
	if index < 0 || msg.Tracer == nil || t.items[index].tracer != msg.Tracer {
		return nil
	}
	if msg.Err == nil {
		return nil
	}
	text := fmt.Sprintf("trace '%s': %v", msg.Key, msg.Err)
	t.api.LogHistory("", text, "log", false, text)
	return listenTraceReports(msg.Key, msg.Tracer)
}
