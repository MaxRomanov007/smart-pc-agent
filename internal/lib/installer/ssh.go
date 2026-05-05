package installer

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func dialSSH(creds Credentials) (*ssh.Client, error) {
	port := creds.Port
	if port == 0 {
		port = 22
	}

	cfg := &ssh.ClientConfig{
		User:            creds.User,
		Auth:            []ssh.AuthMethod{ssh.Password(creds.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", creds.Host, port)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		if isAuthError(err) {
			return nil, ErrAuth
		}
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return client, nil
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unable to authenticate") ||
		strings.Contains(msg, "no supported methods remain") ||
		strings.Contains(msg, "permission denied")
}

// runCommand opens a new session, runs cmd and returns combined output.
func runCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	return string(out), err
}

func hasCommand(client *ssh.Client, name string) bool {
	out, err := runCommand(client,
		fmt.Sprintf("which %s 2>/dev/null || command -v %s 2>/dev/null", name, name))
	return err == nil && strings.TrimSpace(out) != ""
}

func fileExists(client *ssh.Client, path string) bool {
	_, err := runCommand(client, fmt.Sprintf("test -e %s", path))
	return err == nil
}

// writeRemoteFile streams content into path over SSH stdin.
func writeRemoteFile(client *ssh.Client, path, content, mode string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = strings.NewReader(content)
	cmd := fmt.Sprintf("cat > %s && chmod %s %s", path, mode, path)
	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("write %s: %w\n%s", path, err, out)
	}
	return nil
}
