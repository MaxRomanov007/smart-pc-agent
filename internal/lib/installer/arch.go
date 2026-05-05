package installer

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// routerArch holds the name of the binary to download, e.g. "agent-linux-armv7".
type routerArch = string

func detectArch(client *ssh.Client) (routerArch, error) {
	uname, err := runCommand(client, "uname -m")
	if err != nil {
		return "", fmt.Errorf("uname -m: %w", err)
	}
	uname = strings.TrimSpace(uname)

	switch uname {
	case "x86_64":
		return "agent-linux-amd64", nil
	case "i686", "i386":
		return "agent-linux-386", nil
	case "aarch64":
		return "agent-linux-arm64", nil
	case "mips":
		return "agent-linux-mips", nil
	case "mipsel":
		return "agent-linux-mipsle", nil
	case "mips64el":
		return "agent-linux-mips64le", nil
	}

	if strings.HasPrefix(uname, "armv") || uname == "arm" {
		return detectARMVersion(client, uname)
	}

	return "", fmt.Errorf("unrecognised architecture: %q", uname)
}

// detectARMVersion refines arm into v5/v6/v7 using /proc/cpuinfo,
// falling back to the version embedded in uname output (e.g. "armv7l"),
// and ultimately defaulting to armv7.
func detectARMVersion(client *ssh.Client, uname string) (routerArch, error) {
	// Fast path: uname already contains the version, e.g. "armv7l", "armv6l".
	switch {
	case strings.Contains(uname, "v5"):
		return "agent-linux-armv5", nil
	case strings.Contains(uname, "v6"):
		return "agent-linux-armv6", nil
	case strings.Contains(uname, "v7"):
		return "agent-linux-armv7", nil
	}

	// Slow path: parse /proc/cpuinfo.
	cpuinfo, err := runCommand(client, "cat /proc/cpuinfo")
	if err != nil {
		return "agent-linux-armv7", nil // safe default
	}

	for _, line := range strings.Split(cpuinfo, "\n") {
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "cpu architecture") {
			continue
		}
		switch {
		case strings.Contains(lower, "5"):
			return "agent-linux-armv5", nil
		case strings.Contains(lower, "6"):
			return "agent-linux-armv6", nil
		case strings.Contains(lower, "7"):
			return "agent-linux-armv7", nil
		}
	}

	return "agent-linux-armv7", nil
}
