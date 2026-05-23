package system

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopEntry = `[Desktop Entry]
Type=Application
Name=DNS Fetching
Comment=Encrypted DNS-over-HTTPS Client
Exec=%s
Icon=dns-fetching
Categories=Network;Security;
Keywords=DNS;DoH;Privacy;
X-GNOME-Autostart-enabled=true
StartupNotify=false
`

func EnableAutostart() error {
	execPath, err := resolveExecPath()
	if err != nil {
		return err
	}
	dir, err := autostartDir()
	if err != nil {
		return err
	}
	content := fmt.Sprintf(desktopEntry, execPath)
	return os.WriteFile(filepath.Join(dir, "dns-fetching.desktop"), []byte(content), 0644)
}

func DisableAutostart() error {
	dir, err := autostartDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "dns-fetching.desktop")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func IsAutostartEnabled() bool {
	dir, err := autostartDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "dns-fetching.desktop"))
	return err == nil
}

func autostartDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "autostart")
	return dir, os.MkdirAll(dir, 0755)
}

func resolveExecPath() (string, error) {
	// AppImage sets $APPIMAGE to its own path
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		return appimage, nil
	}
	return os.Executable()
}
