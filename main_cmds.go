package main

import (
	"fmt"
	"os"
	"os/exec"
)

func runCommand(args []string, port int) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	commands := map[string]func() error{
		"status":    serviceStatus,
		"logs":      serviceLogs,
		"restart":   serviceRestart,
		"uninstall": serviceUninstall,
		"help": func() error {
			fmt.Println(`Usage: ahweb <command>

  ahweb [--port PORT]    Run the web server directly (default port: 8321)
  install                Install and start the service at boot (use --port before install)
  status                 Show service status
  logs                   Show recent service logs
  restart                Restart the service
  uninstall              Stop and remove the service
  help                   Show this help`)
			return nil
		},
	}
	if len(args) != 1 {
		return true, fmt.Errorf("%s does not accept arguments", args[0])
	}

	if args[0] == "install" {
		return true, serviceInstall(port)
	}

	command, ok := commands[args[0]]
	if !ok {
		return true, fmt.Errorf("unknown command %q", args[0])
	}
	return true, command()
}

func runServiceManager(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %w", name, args, err)
	}
	return nil
}
