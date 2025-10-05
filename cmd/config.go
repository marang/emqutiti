package cmd

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type AppConfig struct {
	ImportFile   string
	ProfileName  string
	TraceKey     string
	TraceTopics  string
	TraceStart   string
	TraceEnd     string
	Timeout      time.Duration
	ListProfiles bool
}

func ParseFlags() AppConfig {
	var cfg AppConfig
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&cfg.ImportFile, "import", "", "Launch import wizard with optional file path")
	fs.StringVar(&cfg.ImportFile, "i", "", "(shorthand)")
	fs.StringVar(&cfg.ProfileName, "profile", "", "Connection profile name to use")
	fs.StringVar(&cfg.ProfileName, "p", "", "(shorthand)")
	fs.StringVar(&cfg.TraceKey, "trace", "", "Trace key name to store messages")
	fs.StringVar(&cfg.TraceTopics, "topics", "", "Comma-separated topics to trace")
	fs.StringVar(&cfg.TraceStart, "start", "", "Optional RFC3339 trace start time")
	fs.StringVar(&cfg.TraceEnd, "end", "", "Optional RFC3339 trace end time")
	fs.DurationVar(&cfg.Timeout, "timeout", 0, "Optional overall runtime limit (e.g., 30s)")
	fs.BoolVar(&cfg.ListProfiles, "list-profiles", false, "List available connection profiles and exit")
	fs.BoolVar(&cfg.ListProfiles, "l", false, "(shorthand)")
	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintln(w, "General:")
		fmt.Fprintln(w, "  -i, --import FILE     Launch import wizard with optional file path (e.g., -i data.csv)")
		fmt.Fprintln(w, "  -p, --profile NAME    Connection profile name to use (e.g., -p local)")
		fmt.Fprintln(w, "  -l, --list-profiles   List available connection profiles and exit")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Trace:")
		fmt.Fprintln(w, "      --trace KEY       Trace key name to store messages (e.g., --trace run1)")
		fmt.Fprintln(w, "      --topics LIST     Comma-separated topics to trace (e.g., --topics \"sensors/#\")")
		fmt.Fprintln(w, "      --start TIME      Optional RFC3339 trace start time (e.g., --start \"2025-08-05T11:47:00Z\")")
		fmt.Fprintln(w, "      --end TIME        Optional RFC3339 trace end time (e.g., --end \"2025-08-05T11:49:00Z\")")
	}
	_ = fs.Parse(os.Args[1:])
	return cfg
}
