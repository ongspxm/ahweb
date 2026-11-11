//go:build linux

package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const serviceName = "ahweb.service"

func serviceInstall(port int) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find ahweb executable: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return fmt.Errorf("create systemd user directory: %w", err)
	}
	path := os.Getenv("PATH")
	fmt.Println("Copying over PATH to systemd service")
	_systemdQuote := func(value string) string { return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"` }
	unit := fmt.Sprintf(`[Unit]
Description=ahweb
After=network.target

[Service]
Environment=%s
ExecStart=%s --port %d
Restart=on-failure

[Install]
WantedBy=default.target
`, _systemdQuote("PATH="+path), _systemdQuote(executable), port)
	unitPath := filepath.Join(unitDir, serviceName)
	if err := os.WriteFile(unitPath, []byte(unit), 0644); err != nil {
		return fmt.Errorf("write systemd user service: %w", err)
	}
	fmt.Printf("Wrote systemd service to %s\n", unitPath)

	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("find current user: %w", err)
	}
	if err := runServiceManager("loginctl", "enable-linger", current.Username); err != nil {
		return fmt.Errorf("enable user lingering: %w", err)
	}
	if err := runServiceManager("systemctl", "--user", "daemon-reload"); err != nil {
		return err
	}
	return runServiceManager("systemctl", "--user", "enable", "--now", serviceName)
}

func serviceStatus() error {
	return runServiceManager("systemctl", "--user", "status", serviceName, "--no-pager")
}

func serviceLogs() error {
	return runServiceManager("journalctl", "--user", "-u", serviceName, "-n", "100", "-f", "--no-pager")
}

func serviceRestart() error {
	return runServiceManager("systemctl", "--user", "restart", serviceName)
}

func serviceUninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	unitPath := filepath.Join(home, ".config", "systemd", "user", serviceName)
	if _, err := os.Stat(unitPath); os.IsNotExist(err) {
		fmt.Println("ahweb service is not installed")
		return nil
	} else if err != nil {
		return err
	}
	if err := runServiceManager("systemctl", "--user", "disable", "--now", serviceName); err != nil {
		return err
	}
	if err := os.Remove(unitPath); err != nil {
		return fmt.Errorf("remove systemd user service: %w", err)
	}
	return runServiceManager("systemctl", "--user", "daemon-reload")
}
