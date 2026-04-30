package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/mcast-hood/keep-screen-awake/internal/awake"
	"github.com/mcast-hood/keep-screen-awake/internal/config"
	"github.com/mcast-hood/keep-screen-awake/internal/transport"
)

// daemonState holds all mutable state used by the running daemon.
type daemonState struct {
	mu          sync.RWMutex
	mode        string
	awakeActive bool
	displayOnly bool
	schedule    []config.Window
	logBuf      []string
	awake       *awake.Manager
}

func (d *daemonState) appendLog(msg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ts := time.Now().Format("2006-01-02T15:04:05")
	d.logBuf = append(d.logBuf, fmt.Sprintf("%s %s", ts, msg))
	const maxLines = 1000
	if len(d.logBuf) > maxLines {
		d.logBuf = d.logBuf[len(d.logBuf)-maxLines:]
	}
}

var cfgPath string

func main() {
	// On Windows this may block inside the service dispatcher; on other
	// platforms it is a no-op.
	maybeRunAsService()

	root := &cobra.Command{
		Use:   "ksad",
		Short: "Keep Screen Awake Daemon",
	}

	root.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "path to config file")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the daemon in the foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return runDaemon(ctx, cfg)
		},
	}

	root.AddCommand(runCmd)
	addServiceCommands(root)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// runDaemon is the platform-independent daemon loop.
func runDaemon(ctx context.Context, cfg *config.Config) error {
	log.SetFlags(log.LstdFlags)
	if cfg.Log.File != "" {
		f, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	mgr := awake.New(cfg.DisplayOnly)
	defer mgr.Close()

	state := &daemonState{
		mode:        cfg.Mode,
		displayOnly: cfg.DisplayOnly,
		schedule:    cfg.Schedule,
		awake:       mgr,
	}

	state.appendLog(fmt.Sprintf("daemon starting, mode=%s displayOnly=%v", cfg.Mode, cfg.DisplayOnly))
	log.Printf("ksad: starting, mode=%s displayOnly=%v", cfg.Mode, cfg.DisplayOnly)

	switch cfg.Mode {
	case "always":
		if err := mgr.Enable(); err != nil {
			return fmt.Errorf("awake enable: %w", err)
		}
		state.mu.Lock()
		state.awakeActive = true
		state.mu.Unlock()
		state.appendLog("awake enabled (always mode)")

	case "toggle":
		if err := mgr.Enable(); err != nil {
			return fmt.Errorf("awake enable: %w", err)
		}
		state.mu.Lock()
		state.awakeActive = true
		state.mu.Unlock()
		state.appendLog("awake enabled (toggle mode, starts on)")

	case "schedule":
		go runScheduler(ctx, state)
	}

	srv := newServer(cfg)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx, makeHandler(state))
	}()

	select {
	case <-ctx.Done():
		log.Println("ksad: shutting down")
		state.appendLog("daemon shutting down")
		return nil
	case err := <-errCh:
		return err
	}
}

// runScheduler checks the schedule every 30 seconds and enables/disables awake.
func runScheduler(ctx context.Context, state *daemonState) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	apply := func() {
		now := time.Now()
		active := inSchedule(now, state.schedule)
		state.mu.Lock()
		changed := active != state.awakeActive
		state.awakeActive = active
		state.mu.Unlock()

		if changed {
			if active {
				if err := state.awake.Enable(); err != nil {
					log.Printf("scheduler: enable awake: %v", err)
				}
				state.appendLog("schedule: awake enabled")
			} else {
				state.awake.Disable()
				state.appendLog("schedule: awake disabled")
			}
		}
	}

	apply()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			apply()
		}
	}
}

// inSchedule returns true if t falls within any schedule window.
func inSchedule(t time.Time, windows []config.Window) bool {
	dayName := t.Weekday().String()[:3] // "Mon", "Tue", etc.
	hhmm := t.Hour()*60 + t.Minute()

	for _, w := range windows {
		if !containsDay(w.Days, dayName) {
			continue
		}
		start, ok1 := parseHHMM(w.Start)
		end, ok2 := parseHHMM(w.End)
		if !ok1 || !ok2 {
			continue
		}
		if hhmm >= start && hhmm < end {
			return true
		}
	}
	return false
}

func containsDay(days []string, day string) bool {
	for _, d := range days {
		if strings.EqualFold(d, day) {
			return true
		}
	}
	return false
}

func parseHHMM(s string) (int, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return h*60 + m, true
}

// makeHandler returns the IPC HandlerFunc for the given daemon state.
func makeHandler(state *daemonState) transport.HandlerFunc {
	return func(req transport.Request) transport.Response {
		switch req.Command {
		case transport.CmdStatus:
			state.mu.RLock()
			defer state.mu.RUnlock()
			windows := make([]transport.Window, len(state.schedule))
			for i, w := range state.schedule {
				windows[i] = transport.Window{Start: w.Start, End: w.End, Days: w.Days}
			}
			return transport.Response{
				OK: true,
				Data: transport.StatusData{
					Mode:        state.mode,
					AwakeActive: state.awakeActive,
					DisplayOnly: state.displayOnly,
					Schedule:    windows,
				},
			}

		case transport.CmdOn:
			state.mu.Lock()
			state.awakeActive = true
			state.mu.Unlock()
			if err := state.awake.Enable(); err != nil {
				return transport.Response{OK: false, Error: err.Error()}
			}
			state.appendLog("awake enabled via IPC")
			return transport.Response{OK: true}

		case transport.CmdOff:
			state.mu.Lock()
			state.awakeActive = false
			state.mu.Unlock()
			state.awake.Disable()
			state.appendLog("awake disabled via IPC")
			return transport.Response{OK: true}

		case transport.CmdMode:
			newMode := req.Mode
			switch newMode {
			case "always", "toggle", "schedule":
			default:
				return transport.Response{OK: false, Error: fmt.Sprintf("unknown mode %q", newMode)}
			}
			state.mu.Lock()
			state.mode = newMode
			state.mu.Unlock()
			state.appendLog(fmt.Sprintf("mode changed to %s via IPC", newMode))
			return transport.Response{OK: true}

		case transport.CmdLogs:
			state.mu.RLock()
			defer state.mu.RUnlock()
			lines := req.Lines
			if lines <= 0 {
				lines = 50
			}
			buf := state.logBuf
			if len(buf) > lines {
				buf = buf[len(buf)-lines:]
			}
			out := make([]string, len(buf))
			copy(out, buf)
			return transport.Response{OK: true, Data: transport.LogsData{Lines: out}}

		default:
			return transport.Response{OK: false, Error: fmt.Sprintf("unknown command %q", req.Command)}
		}
	}
}

