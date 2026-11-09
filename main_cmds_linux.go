//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const serviceName = "ahweb.service"

func serviceInstall() error {
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
	pathDirs := []string{}
	seenDirs := make(map[string]bool)
	for _, dir := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if dir != "" && !seenDirs[dir] {
			pathDirs = append(pathDirs, dir)
			seenDirs[dir] = true
		}
	}
	for _, dir := range []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"} {
		if !seenDirs[dir] {
			pathDirs = append(pathDirs, dir)
			seenDirs[dir] = true
		}
	}

	// make sure that ahweb cli can access the harness cli-es
	fmt.Println("Harnesses:")
	harnesses := []struct{ name, command string }{{"pi", "pi"}, {"prime", "prime-agent"}, {"opencode", "opencode"}, {"claude", "claude"}, {"codex", "codex"}, {"cursor", "cursor-agent"}}
	for _, harness := range harnesses {
		path, err := exec.LookPath(harness.command)
		if err != nil {
			fmt.Printf("  %-8s !found\n", harness.name)
			continue
		}
		dir, err := filepath.Abs(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("find %s executable directory: %w", harness.name, err)
		}
		if !seenDirs[dir] {
			pathDirs = append(pathDirs, dir)
			seenDirs[dir] = true
		}
		fmt.Printf("  %-8s =found (%s)\n", harness.name, path)
	}
	_systemdQuote := func(value string) string { return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"` }
	unit := "[Unit]\nDescription=ahweb\nAfter=network.target\n\n[Service]\nEnvironment=" + _systemdQuote("PATH="+strings.Join(pathDirs, string(os.PathListSeparator))) + "\nExecStart=" + _systemdQuote(executable) + "\nRestart=on-failure\n\n[Install]\nWantedBy=default.target\n"
	if err := os.WriteFile(filepath.Join(unitDir, serviceName), []byte(unit), 0644); err != nil {
		return fmt.Errorf("write systemd user service: %w", err)
	}

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
