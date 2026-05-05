package installer

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// upload streams a local file to remotePath on the router using cat over SSH
// stdin. This avoids requiring the sftp subsystem, which is absent on many
// minimal router firmwares.
func upload(client *ssh.Client, localPath, remotePath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = f
	cmd := fmt.Sprintf("cat > %s && chmod +x %s", remotePath, remotePath)
	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("upload to %s failed: %w\n%s", remotePath, err, out)
	}
	return nil
}
