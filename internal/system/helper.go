package system

import (
	"encoding/json"
	"fmt"
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
	// check next to our own binary first
	helperPath := filepath.Join(filepath.Dir(self), "dns-fetching-helper")
	if _, err := os.Stat(helperPath); err == nil {
		return helperPath, nil
	}
	return exec.LookPath("dns-fetching-helper")
}
