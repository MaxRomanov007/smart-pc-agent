// Package installer connects to a router over SSH, detects its architecture,
// downloads the latest release binary from GitHub and configures autostart.
package installer

import (
	"errors"
	"fmt"
)

// ErrAuth is returned when SSH authentication fails.
// The caller should prompt for new credentials and retry.
var ErrAuth = errors.New("SSH authentication failed")

// Credentials holds SSH connection parameters.
type Credentials struct {
	Host     string // e.g. "192.168.1.1"
	Port     int    // defaults to 22
	User     string
	Password string
}

// Options controls installation behavior.
type Options struct {
	// GitHub repo in "owner/repo" format, e.g. "acme/router-agent".
	Repo string
	// Remote absolute path where the binary will be placed, e.g. "/usr/bin/agent".
	RemoteBinaryPath string
	// ServiceName is used for init scripts, e.g. "agent".
	ServiceName string
}

type InstallStep int

const (
	InstallStepDialSSH InstallStep = iota
	InstallStepDetectArch
	InstallStepGetLatestTag
	InstallStepDownloadBinary
	InstallStepUpload
	InstallStepConfigureAutostart
	InstallStepRemoveTempFile
	InstallStepCloseConnection
)

// InstallWithSteps connects to the router, detects its architecture, downloads the
// latest release binary from GitHub, uploads it and configures autostart.
// Returns ErrAuth if credentials are rejected so the caller can retry.
// Uses stepFunc to provide installing step
func InstallWithSteps(creds Credentials, opts Options, stepFunc func(InstallStep)) error {
	stepFunc(InstallStepDialSSH)
	client, err := dialSSH(creds)
	if err != nil {
		return err
	}
	defer func() {
		stepFunc(InstallStepCloseConnection)
		_ = client.Close()
	}()

	stepFunc(InstallStepDetectArch)
	arch, err := detectArch(client)
	if err != nil {
		return fmt.Errorf("arch detection: %w", err)
	}

	stepFunc(InstallStepGetLatestTag)
	tag, err := latestTag(opts.Repo)
	if err != nil {
		return fmt.Errorf("latest release: %w", err)
	}

	stepFunc(InstallStepDownloadBinary)
	binary, err := downloadBinary(opts.Repo, tag, arch)
	if err != nil {
		return fmt.Errorf("download (%s, %s): %w", tag, arch, err)
	}
	defer func() {
		stepFunc(InstallStepRemoveTempFile)
		removeTempFile(binary)
	}()

	stepFunc(InstallStepUpload)
	if err := upload(client, binary, opts.RemoteBinaryPath); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	stepFunc(InstallStepConfigureAutostart)
	if err := configureAutostart(client, opts.RemoteBinaryPath, opts.ServiceName); err != nil {
		return fmt.Errorf("autostart: %w", err)
	}

	return nil
}
