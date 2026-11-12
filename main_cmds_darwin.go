//go:build darwin

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

const serviceLabel = "com.ongspxm.ahweb"

func serviceInstall(port int) error {
	account, err := installUser()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find ahweb executable: %w", err)
	}
	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return fmt.Errorf("parse installing user id: %w", err)
	}
	gui := "gui/" + strconv.Itoa(uid)
	agentDir := filepath.Join(account.HomeDir, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return fmt.Errorf("create LaunchAgents directory: %w", err)
	}
	logDir := filepath.Join(account.HomeDir, "Library", "Logs", "ahweb")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create ahweb log directory: %w", err)
	}
	servicePlist := filepath.Join(agentDir, serviceLabel+".plist")
	path := os.Getenv("PATH")
	fmt.Println("Copying over PATH to launchd service")
	if err := os.WriteFile(servicePlist, launchdPlist(executable, account.HomeDir, logDir, path, port), 0644); err != nil {
		return fmt.Errorf("write launch agent: %w", err)
	}
	// Reinstalling may leave the previous agent loaded.
	_ = runServiceManager("launchctl", "bootout", gui, servicePlist)
	fmt.Printf("Wrote launch agent to %s\n", servicePlist)
	return runServiceManager("launchctl", "bootstrap", gui, servicePlist)
}

func serviceStatus() error {
	account, err := installUser()
	if err != nil {
		return err
	}
	return runServiceManager("launchctl", "print", "gui/"+account.Uid+"/"+serviceLabel)
}

func serviceLogs() error {
	account, err := installUser()
	if err != nil {
		return err
	}
	logDir := filepath.Join(account.HomeDir, "Library", "Logs", "ahweb")
	return runServiceManager("tail", "-n", "100", "-f", filepath.Join(logDir, "stdout.log"), filepath.Join(logDir, "stderr.log"))
}

func serviceRestart() error {
	account, err := installUser()
	if err != nil {
		return err
	}
	return runServiceManager("launchctl", "kickstart", "-k", "gui/"+account.Uid+"/"+serviceLabel)
}

func serviceUninstall() error {
	account, err := installUser()
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return fmt.Errorf("parse installing user id: %w", err)
	}
	servicePlist := filepath.Join(account.HomeDir, "Library", "LaunchAgents", serviceLabel+".plist")
	if _, err := os.Stat(servicePlist); os.IsNotExist(err) {
		fmt.Println("ahweb service is not installed")
		return nil
	} else if err != nil {
		return err
	}
	if err := runServiceManager("launchctl", "bootout", "gui/"+strconv.Itoa(uid), servicePlist); err != nil {
		return err
	}
	if err := os.Remove(servicePlist); err != nil {
		return fmt.Errorf("remove launch agent: %w", err)
	}
	return nil
}

func installUser() (*user.User, error) {
	if os.Geteuid() == 0 {
		return nil, fmt.Errorf("run this command as your logged-in user, without sudo")
	}
	account, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("find current user: %w", err)
	}
	return account, nil
}

func launchdPlist(executable, home, logDir, path string, port int) []byte {
	var b bytes.Buffer
	write := func(value string) {
		_ = xml.EscapeText(&b, []byte(value))
	}
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>`)
	write(serviceLabel)
	b.WriteString(`</string><key>ProgramArguments</key><array><string>`)
	write(executable)
	b.WriteString(`</string><string>--port</string><string>`)
	write(strconv.Itoa(port))
	b.WriteString(`</string></array><key>EnvironmentVariables</key><dict><key>HOME</key><string>`)
	write(home)
	b.WriteString(`</string><key>PATH</key><string>`)
	write(path)
	b.WriteString(`</string><key>TERM</key><string>xterm</string></dict><key>RunAtLoad</key><true/><key>KeepAlive</key><true/><key>StandardOutPath</key><string>`)
	write(filepath.Join(logDir, "stdout.log"))
	b.WriteString(`</string><key>StandardErrorPath</key><string>`)
	write(filepath.Join(logDir, "stderr.log"))
	b.WriteString(`</string></dict></plist>`)
	return b.Bytes()
}
