package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/whooslizi/dns-fetching/internal/config"
)

const resolvedDropin = `[Resolve]
DNS=127.0.0.1
DNSStubListener=no
`

const dropinDir = "/etc/systemd/resolved.conf.d"
const dropinFile = "00-dns-fetching.conf"

func ConfigureResolved() error {
	if err := os.MkdirAll(dropinDir, 0755); err != nil {
		return fmt.Errorf("can't create drop-in dir: %w", err)
	}
	path := filepath.Join(dropinDir, dropinFile)
	if err := os.WriteFile(path, []byte(resolvedDropin), 0644); err != nil {
		return fmt.Errorf("can't write drop-in: %w", err)
	}
	return restartResolved()
}

func RestoreResolved() error {
	path := filepath.Join(dropinDir, dropinFile)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("can't remove drop-in: %w", err)
	}
	return restartResolved()
}

func restartResolved() error {
	cmd := exec.Command("systemctl", "restart", "systemd-resolved")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restart failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func BackupResolvedConfig() error {
	backupDir, err := config.BackupDir()
	if err != nil {
		return err
	}

	data, err := os.ReadFile("/etc/systemd/resolved.conf")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(filepath.Join(backupDir, "resolved.conf.bak"), data, 0600)
}

func IsResolvedRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

func SetCapability(binaryPath string) error {
	cmd := exec.Command("setcap", "cap_net_bind_service=+ep", binaryPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("setcap failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}
