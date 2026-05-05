package installer

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

// Uninstall connects to the router, stops and removes the service and deletes
// the binary. Returns ErrAuth if credentials are rejected.
func Uninstall(creds Credentials, opts Options) error {
	client, err := dialSSH(creds)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := removeAutostart(client, opts.RemoteBinaryPath, opts.ServiceName); err != nil {
		return fmt.Errorf("remove autostart: %w", err)
	}

	if _, err := runCommand(client, fmt.Sprintf("rm -f %s", opts.RemoteBinaryPath)); err != nil {
		return fmt.Errorf("remove binary: %w", err)
	}

	return nil
}

// removeAutostart detects which init system was used and reverses the setup.
func removeAutostart(client *ssh.Client, binaryPath, serviceName string) error {
	switch {
	case hasCommand(client, "procd") || fileExists(client, "/etc/rc.common"):
		return removeProcd(client, serviceName)
	case fileExists(client, "/etc/init.d"):
		return removeSysV(client, serviceName)
	default:
		return removeRcLocal(client, binaryPath)
	}
}

func removeProcd(client *ssh.Client, serviceName string) error {
	initPath := fmt.Sprintf("/etc/init.d/%s", serviceName)
	if !fileExists(client, initPath) {
		return nil
	}

	for _, arg := range []string{"stop", "disable"} {
		// Ignore errors — service may already be stopped/disabled.
		_, _ = runCommand(client, fmt.Sprintf("%s %s", initPath, arg))
	}

	_, err := runCommand(client, fmt.Sprintf("rm -f %s", initPath))
	return err
}

func removeSysV(client *ssh.Client, serviceName string) error {
	initPath := fmt.Sprintf("/etc/init.d/%s", serviceName)
	if !fileExists(client, initPath) {
		return nil
	}

	_, _ = runCommand(client, fmt.Sprintf("%s stop", initPath))
	_, _ = runCommand(client, fmt.Sprintf(
		"update-rc.d -f %s remove 2>/dev/null || rm -f /etc/rc.d/S99%s 2>/dev/null",
		serviceName, serviceName,
	))

	_, err := runCommand(client, fmt.Sprintf("rm -f %s", initPath))
	return err
}

func removeRcLocal(client *ssh.Client, binaryPath string) error {
	// Kill running process, then strip the line from rc.local.
	_, _ = runCommand(client, fmt.Sprintf("killall %s 2>/dev/null", binaryPath))

	_, err := runCommand(client, fmt.Sprintf(
		`sed -i '\|%s|d' /etc/rc.local 2>/dev/null || true`,
		binaryPath,
	))
	return err
}
