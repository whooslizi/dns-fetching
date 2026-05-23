package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/whooslizi/dns-fetching/internal/system"
)

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		respond(false, "", fmt.Sprintf("can't read input: %v", err))
		os.Exit(1)
	}

	var req system.HelperRequest
	if err := json.Unmarshal(data, &req); err != nil {
		respond(false, "", fmt.Sprintf("bad request: %v", err))
		os.Exit(1)
	}

	switch req.Action {
	case system.ActionEnable:
		if err := system.BackupResolvedConfig(); err != nil {
			respond(false, "", fmt.Sprintf("backup failed: %v", err))
			os.Exit(1)
		}
		if err := system.ConfigureResolved(); err != nil {
			respond(false, "", fmt.Sprintf("configure failed: %v", err))
			os.Exit(1)
		}
		if req.BinaryPath != "" {
			if err := system.SetCapability(req.BinaryPath); err != nil {
				respond(false, "", fmt.Sprintf("setcap failed: %v", err))
				os.Exit(1)
			}
		}
		respond(true, "DNS configured and capability set", "")

	case system.ActionDisable:
		if err := system.RestoreResolved(); err != nil {
			respond(false, "", fmt.Sprintf("restore failed: %v", err))
			os.Exit(1)
		}
		respond(true, "DNS settings restored", "")

	case system.ActionRestore:
		if err := system.RestoreResolved(); err != nil {
			respond(false, "", fmt.Sprintf("restore failed: %v", err))
			os.Exit(1)
		}
		respond(true, "All settings restored to defaults", "")

	case system.ActionSetCap:
		binaryPath := req.BinaryPath
		if binaryPath == "" {
			respond(false, "", "no binary path provided")
			os.Exit(1)
		}
		clean := filepath.Clean(binaryPath)
		if clean != binaryPath || !filepath.IsAbs(clean) {
			respond(false, "", "invalid binary path")
			os.Exit(1)
		}
		if err := system.SetCapability(clean); err != nil {
			respond(false, "", fmt.Sprintf("setcap failed: %v", err))
			os.Exit(1)
		}
		respond(true, "Capability set", "")

	default:
		respond(false, "", fmt.Sprintf("unknown action: %s", req.Action))
		os.Exit(1)
	}
}

func respond(success bool, message, errMsg string) {
	resp := system.HelperResponse{Success: success, Message: message, Error: errMsg}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}
