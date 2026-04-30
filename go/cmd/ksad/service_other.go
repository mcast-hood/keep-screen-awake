//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/mcast-hood/keep-screen-awake/internal/config"
	"github.com/mcast-hood/keep-screen-awake/internal/transport"
)

const (
	launchdLabel    = "com.mcast-hood.ksad"
	plistFilename   = "com.mcast-hood.ksad.plist"
	launchAgentsDir = "Library/LaunchAgents"
	logsDir         = "Library/Logs"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>{{.Label}}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{.ExePath}}</string>
    <string>run</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>{{.LogDir}}/ksad.log</string>
  <key>StandardErrorPath</key>
  <string>{{.LogDir}}/ksad.error.log</string>
</dict>
</plist>
`

// newServer returns an HTTP Server for non-Windows platforms.
func newServer(cfg *config.Config) transport.Server {
	return transport.NewHTTPServer(cfg.IPC.HTTPPort)
}

// maybeRunAsService is a no-op on non-Windows platforms.
func maybeRunAsService() {}

// addServiceCommands wires launchd install/uninstall/start/stop sub-commands.
func addServiceCommands(root *cobra.Command) {
	root.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "Install the launchd agent plist",
			RunE:  launchdInstall,
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "Uninstall the launchd agent plist",
			RunE:  launchdUninstall,
		},
		&cobra.Command{
			Use:   "start",
			Short: "Start the launchd agent",
			RunE:  launchdStart,
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop the launchd agent",
			RunE:  launchdStop,
		},
	)
}

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, launchAgentsDir, plistFilename), nil
}

func logDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, logsDir), nil
}

func launchdInstall(_ *cobra.Command, _ []string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	ld, err := logDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ld, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	pp, err := plistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(pp), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("parse plist template: %w", err)
	}

	f, err := os.Create(pp)
	if err != nil {
		return fmt.Errorf("create plist %q: %w", pp, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, struct {
		Label   string
		ExePath string
		LogDir  string
	}{
		Label:   launchdLabel,
		ExePath: exePath,
		LogDir:  ld,
	}); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	fmt.Printf("Installed plist to %s\n", pp)
	fmt.Printf("Run 'ksad start' to load the agent.\n")
	return nil
}

func launchdUninstall(_ *cobra.Command, _ []string) error {
	pp, err := plistPath()
	if err != nil {
		return err
	}

	// Attempt to unload first; ignore errors.
	_ = exec.Command("launchctl", "unload", pp).Run()

	if err := os.Remove(pp); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist %q: %w", pp, err)
	}

	fmt.Printf("Uninstalled plist %s\n", pp)
	return nil
}

func launchdStart(_ *cobra.Command, _ []string) error {
	pp, err := plistPath()
	if err != nil {
		return err
	}
	out, err := exec.Command("launchctl", "load", "-w", pp).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	fmt.Printf("Agent %s loaded.\n", launchdLabel)
	return nil
}

func launchdStop(_ *cobra.Command, _ []string) error {
	pp, err := plistPath()
	if err != nil {
		return err
	}
	out, err := exec.Command("launchctl", "unload", pp).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl unload: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	fmt.Printf("Agent %s unloaded.\n", launchdLabel)
	return nil
}
