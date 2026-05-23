package system

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type HelperAction string

const (
	ActionEnable  HelperAction = "enable"
	ActionDisable HelperAction = "disable"
	ActionRestore HelperAction = "restore"
	ActionSetCap  HelperAction = "setcap"
)

type HelperRequest struct {
	Action     HelperAction `json:"action"`
	BinaryPath string       `json:"binary_path,omitempty"`
}

type HelperResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func RunHelper(action HelperAction, binaryPath string) (*HelperResponse, error) {
	helperPath, err := findHelper()
	if err != nil {
		return nil, fmt.Errorf("can't find helper binary: %w", err)
	}

	reqData, err := json.Marshal(HelperRequest{Action: action, BinaryPath: binaryPath})
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("pkexec", helperPath)
	cmd.Stdin = strings.NewReader(string(reqData))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("helper failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("pkexec failed: %w", err)
	}

	var resp HelperResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("bad helper response: %w", err)
	}
	if !resp.Success {
		return &resp, fmt.Errorf("helper error: %s", resp.Error)
	}
	return &resp, nil
}

func findHelper() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}

	helperPath := filepath.Join(filepath.Dir(self), "dns-fetching-helper")
	if isInsideAppImage(helperPath) {
		extracted, err := extractHelper(helperPath)
		if err != nil {
			return "", fmt.Errorf("extract helper from AppImage: %w", err)
		}
		return extracted, nil
	}

	if _, err := os.Stat(helperPath); err == nil {
		return helperPath, nil
	}
	return exec.LookPath("dns-fetching-helper")
}

func isInsideAppImage(path string) bool {
	return strings.HasPrefix(path, "/tmp/.mount_") || os.Getenv("APPIMAGE") != ""
}
func extractHelper(srcPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	destDir := filepath.Join(home, ".config", "dns-fetching", "bin")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(destDir, "dns-fetching-helper")

	// check if source exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("helper not found in AppImage at %s: %w", srcPath, err)
	}

	// always overwrite to keep in sync with the running AppImage version
	src, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	// ensure executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return "", err
	}

	log.Printf("extracted helper (%d bytes) to %s", srcInfo.Size(), destPath)
	return destPath, nil
}

