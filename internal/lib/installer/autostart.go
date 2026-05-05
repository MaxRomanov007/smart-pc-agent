package installer

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

// configureAutostart detects the available init system and installs a service.
// Priority: procd (OpenWrt) → sysv init.d → rc.local fallback.
func configureAutostart(client *ssh.Client, binaryPath, serviceName string) error {
	switch {
	case hasCommand(client, "procd") || fileExists(client, "/etc/rc.common"):
		return configureProcd(client, binaryPath, serviceName)
	case fileExists(client, "/etc/init.d"):
		return configureSysV(client, binaryPath, serviceName)
	default:
		return configureRcLocal(client, binaryPath)
	}
}

// configureProcd installs a procd-style init script (OpenWrt).
// If the service is already running it will be restarted to pick up the new binary.
func configureProcd(client *ssh.Client, binaryPath, serviceName string) error {
	script := fmt.Sprintf(`#!/bin/sh /etc/rc.common
START=99
STOP=10
USE_PROCD=1

start_service() {
    procd_open_instance
    procd_set_param command %s
    procd_set_param respawn 3600 5 0
    procd_set_param stdout 1
    procd_set_param stderr 1
    procd_close_instance
}
`, binaryPath)

	initPath := fmt.Sprintf("/etc/init.d/%s", serviceName)
	alreadyInstalled := fileExists(client, initPath)

	if err := writeRemoteFile(client, initPath, script, "0755"); err != nil {
		return err
	}

	if alreadyInstalled {
		// Service exists — restart to pick up the new binary.
		// Ignore error: some firmwares return non-zero even on success.
		_, _ = runCommand(client, fmt.Sprintf("%s restart", initPath))
		return nil
	}

	for _, arg := range []string{"enable", "start"} {
		cmd := fmt.Sprintf("%s %s", initPath, arg)
		if out, err := runCommand(client, cmd); err != nil {
			return fmt.Errorf("procd %s: %w\n%s", arg, err, out)
		}
	}
	return nil
}

// configureSysV installs a SysV-style init script (Padavan, older firmwares).
// If the service is already installed it stops the old process before starting the new one.
func configureSysV(client *ssh.Client, binaryPath, serviceName string) error {
	script := fmt.Sprintf(`#!/bin/sh
### BEGIN INIT INFO
# Provides:          %s
# Required-Start:    $network
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
### END INIT INFO

case "$1" in
  start) %s &;;
  stop)  killall %s 2>/dev/null;;
  *)     echo "Usage: $0 {start|stop}"; exit 1;;
esac
`, serviceName, binaryPath, serviceName)

	initPath := fmt.Sprintf("/etc/init.d/%s", serviceName)
	alreadyInstalled := fileExists(client, initPath)

	// Stop old process before overwriting the binary on disk.
	if alreadyInstalled {
		_, _ = runCommand(client, fmt.Sprintf("%s stop", initPath))
	}

	if err := writeRemoteFile(client, initPath, script, "0755"); err != nil {
		return err
	}

	if alreadyInstalled {
		_, err := runCommand(client, fmt.Sprintf("%s start", initPath))
		return err
	}

	// First install: register with rc system and start.
	cmd := fmt.Sprintf(
		"update-rc.d %s defaults 2>/dev/null || ln -sf %s /etc/rc.d/S99%s 2>/dev/null; %s start",
		serviceName, initPath, serviceName, initPath,
	)
	_, err := runCommand(client, cmd)
	return err
}

// configureRcLocal appends a start command to /etc/rc.local (last resort).
// On reinstall it kills the running process and replaces it; the rc.local line
// is already present so no duplicate is added.
func configureRcLocal(client *ssh.Client, binaryPath string) error {
	alreadyInstalled, _ := runCommand(client,
		fmt.Sprintf("grep -qF '%s' /etc/rc.local 2>/dev/null && echo yes || echo no", binaryPath))

	if alreadyInstalled == "yes\n" {
		// Kill old process so the caller's upload takes effect immediately.
		_, _ = runCommand(
			client,
			fmt.Sprintf("killall %s 2>/dev/null; %s &", binaryPath, binaryPath),
		)
		return nil
	}

	// First install: insert before "exit 0" or append.
	cmd := fmt.Sprintf(
		`sed -i 's|^exit 0|%[1]s \&\nexit 0|' /etc/rc.local 2>/dev/null || `+
			`echo '%[1]s &' >> /etc/rc.local`,
		binaryPath,
	)
	if _, err := runCommand(client, cmd); err != nil {
		return err
	}
	_, err := runCommand(client, fmt.Sprintf("%s &", binaryPath))
	return err
}
