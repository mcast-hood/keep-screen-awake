//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/mcast-hood/keep-screen-awake/internal/config"
	"github.com/mcast-hood/keep-screen-awake/internal/transport"
)

const (
	serviceName        = "KeepScreenAwake"
	serviceDisplayName = "KeepScreenAwake \u2014 Keep Screen Awake Daemon"
	serviceDescription = "Prevents the display and system from sleeping."
)

// newServer returns a Windows named-pipe Server.
func newServer(cfg *config.Config) transport.Server {
	return transport.NewPipeServer(cfg.IPC.PipeName)
}

// ksaService implements svc.Handler.
type ksaService struct {
	cfg *config.Config
}

func (s *ksaService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runDaemon(ctx, s.cfg)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case <-errCh:
				case <-time.After(10 * time.Second):
				}
				changes <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		case err := <-errCh:
			if err != nil {
				log.Printf("ksad service: daemon error: %v", err)
			}
			changes <- svc.Status{State: svc.Stopped}
			return false, 0
		}
	}
}

// maybeRunAsService checks whether ksad is running as a Windows service and, if
// so, enters the service dispatcher (blocking until stopped).
func maybeRunAsService() {
	inSvc, err := svc.IsWindowsService()
	if err != nil || !inSvc {
		return
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("ksad: load config: %v", err)
	}

	if err := svc.Run(serviceName, &ksaService{cfg: cfg}); err != nil {
		log.Fatalf("ksad: service run: %v", err)
	}
	os.Exit(0)
}

// addServiceCommands wires install/uninstall/start/stop sub-commands.
func addServiceCommands(root *cobra.Command) {
	root.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "Install the Windows service",
			RunE:  serviceInstall,
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "Uninstall the Windows service",
			RunE:  serviceUninstall,
		},
		&cobra.Command{
			Use:   "start",
			Short: "Start the Windows service",
			RunE:  serviceStart,
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop the Windows service",
			RunE:  serviceStop,
		},
	)
}

func openSCM() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("connect to service manager: %w", err)
	}
	return m, nil
}

func serviceInstall(_ *cobra.Command, _ []string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	m, err := openSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.CreateService(
		serviceName,
		exePath,
		mgr.Config{
			StartType:   mgr.StartAutomatic,
			DisplayName: serviceDisplayName,
			Description: serviceDescription,
		},
		"run",
	)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer s.Close()

	fmt.Printf("Service %q installed.\n", serviceName)
	return nil
}

func serviceUninstall(_ *cobra.Command, _ []string) error {
	m, err := openSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("open service %q: %w", serviceName, err)
	}
	defer s.Close()

	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service %q: %w", serviceName, err)
	}

	fmt.Printf("Service %q uninstalled.\n", serviceName)
	return nil
}

func serviceStart(_ *cobra.Command, _ []string) error {
	m, err := openSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("open service %q: %w", serviceName, err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("start service %q: %w", serviceName, err)
	}

	fmt.Printf("Service %q started.\n", serviceName)
	return nil
}

func serviceStop(_ *cobra.Command, _ []string) error {
	m, err := openSCM()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("open service %q: %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("stop service %q: %w", serviceName, err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for service %q to stop", serviceName)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("query service %q: %w", serviceName, err)
		}
	}

	fmt.Printf("Service %q stopped.\n", serviceName)
	return nil
}
