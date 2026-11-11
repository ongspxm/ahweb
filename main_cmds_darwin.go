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
const servicePlist = "/Library/LaunchDaemons/com.ongspxm.ahweb.plist"

func serviceInstall(port int) error {
	if err := requireAdmin(); err != nil {
		return err
	}
	account, err := installUser()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find ahweb executable: %w", err)
	}
	logDir := filepath.Join(account.HomeDir, "Library", "Logs", "ahweb")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create ahweb log directory: %w", err)
	}
	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return fmt.Errorf("parse installing user id: %w", err)
	}
	gid, err := strconv.Atoi(account.Gid)
	if err != nil {
		return fmt.Errorf("parse installing group id: %w", err)
	}
	if err := os.Chown(logDir, uid, gid); err != nil {
		return fmt.Errorf("set ahweb log directory owner: %w", err)
	}
	path := os.Getenv("PATH")
	fmt.Println("Copying over PATH to launchd service")
	plist := launchdPlist(executable, account.Username, account.HomeDir, logDir, path, port)
	if err := os.WriteFile(servicePlist, plist, 0644); err != nil {
		return fmt.Errorf("write launchd service (run with sudo): %w", err)
	}
	fmt.Printf("Wrote launchd service to %s\n", servicePlist)
	if err := runServiceManager("launchctl", "enable", "system/"+serviceLabel); err != nil {
		return err
	}
	return runServiceManager("launchctl", "bootstrap", "system", servicePlist)
}

func serviceStatus() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	return runServiceManager("launchctl", "print", "system/"+serviceLabel)
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
	if err := requireAdmin(); err != nil {
		return err
	}
	return runServiceManager("launchctl", "kickstart", "-k", "system/"+serviceLabel)
}

func serviceUninstall() error {
	if err := requireAdmin(); err != nil {
		return err
	}
	if _, err := os.Stat(servicePlist); os.IsNotExist(err) {
		fmt.Println("ahweb service is not installed")
		return nil
	} else if err != nil {
		return err
	}
	if err := runServiceManager("launchctl", "bootout", "system", servicePlist); err != nil {
		return err
	}
	if err := runServiceManager("launchctl", "disable", "system/"+serviceLabel); err != nil {
		return err
	}
	return os.Remove(servicePlist)
}

func requireAdmin() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command needs admin permission; run with sudo")
	}
	return nil
}

func installUser() (*user.User, error) {
	name := os.Getenv("SUDO_USER")
	if name == "" && os.Geteuid() != 0 {
		current, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("find current user: %w", err)
		}
		name = current.Username
	}
	if name == "" || name == "root" {
		return nil, fmt.Errorf("could not identify the installing user; use sudo ahweb install")
	}
	account, err := user.Lookup(name)
	if err != nil {
		return nil, fmt.Errorf("find installing user %q: %w", name, err)
	}
	return account, nil
}

func launchdPlist(executable, username, home, logDir, path string, port int) []byte {
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
	b.WriteString(`</string></array><key>UserName</key><string>`)
	write(username)
	b.WriteString(`</string><key>EnvironmentVariables</key><dict><key>HOME</key><string>`)
	write(home)
	b.WriteString(`</string><key>PATH</key><string>`)
	write(path)
	b.WriteString(`</string></dict><key>RunAtLoad</key><true/><key>KeepAlive</key><true/><key>StandardOutPath</key><string>`)
	write(filepath.Join(logDir, "stdout.log"))
	b.WriteString(`</string><key>StandardErrorPath</key><string>`)
	write(filepath.Join(logDir, "stderr.log"))
	b.WriteString(`</string></dict></plist>`)
	return b.Bytes()
}
