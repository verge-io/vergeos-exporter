//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceName        = "VergeOSExporter"
	serviceDisplayName = "VergeOS Exporter"
	serviceDescription = "Prometheus exporter for VergeOS metrics"
)

// exporterService adapts runExporter to the Windows Service Control Manager.
type exporterService struct{}

func (s *exporterService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- runExporter(stop) }()

	changes <- svc.Status{State: svc.Running, Accepts: accepted}
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				close(stop)
				<-done
				changes <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		case err := <-done:
			// The exporter exited on its own (startup failure or server error).
			if err != nil {
				return false, 1
			}
			changes <- svc.Status{State: svc.Stopped}
			return false, 0
		}
	}
}

// runWindowsService handles service management actions and hosting under the SCM.
// It returns handled=true when it has taken over execution (so main() should
// return), or handled=false to fall through to a normal foreground run.
func runWindowsService(action string) (handled bool, err error) {
	switch strings.ToLower(action) {
	case "install":
		return true, installService()
	case "uninstall", "remove":
		return true, removeService()
	case "start":
		return true, startService()
	case "stop":
		return true, stopService()
	case "run":
		return true, svc.Run(serviceName, &exporterService{})
	case "":
		// Auto-detect: if the SCM launched us, host the service; otherwise run
		// in the foreground exactly like on any other platform.
		isService, derr := svc.IsWindowsService()
		if derr != nil {
			return false, fmt.Errorf("failed to determine service context: %w", derr)
		}
		if isService {
			return true, svc.Run(serviceName, &exporterService{})
		}
		return false, nil
	default:
		return true, fmt.Errorf("unknown -service action %q (install|uninstall|start|stop|run)", action)
	}
}

// serviceArgs reconstructs the flags the operator passed at install time so the
// SCM launches the service with the same configuration. The -service flag
// itself is replaced with "-service=run".
func serviceArgs() []string {
	args := []string{"-service=run"}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "service" {
			return
		}
		args = append(args, "-"+f.Name+"="+f.Value.String())
	})
	return args
}

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager (run as Administrator): %w", err)
	}
	defer m.Disconnect()

	if s, err := m.OpenService(serviceName); err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	s, err := m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName:  serviceDisplayName,
		Description:  serviceDescription,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	}, serviceArgs()...)
	if err != nil {
		return fmt.Errorf("could not create service: %w", err)
	}
	defer s.Close()

	fmt.Printf("Installed service %q. Start it with:\n  %s -service=start\n", serviceName, filepath.Base(exePath))
	return nil
}

func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager (run as Administrator): %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed: %w", serviceName, err)
	}
	defer s.Close()

	if err := s.Delete(); err != nil {
		return fmt.Errorf("could not remove service: %w", err)
	}
	fmt.Printf("Removed service %q.\n", serviceName)
	return nil
}

func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager (run as Administrator): %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed: %w", serviceName, err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("could not start service: %w", err)
	}
	fmt.Printf("Started service %q.\n", serviceName)
	return nil
}

func stopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager (run as Administrator): %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed: %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("could not stop service: %w", err)
	}

	// Wait up to 15s for the service to report Stopped.
	deadline := 30
	for status.State != svc.Stopped && deadline > 0 {
		time.Sleep(500 * time.Millisecond)
		deadline--
		if status, err = s.Query(); err != nil {
			return fmt.Errorf("could not query service status: %w", err)
		}
	}
	fmt.Printf("Stopped service %q.\n", serviceName)
	return nil
}
