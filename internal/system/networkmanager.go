package system

import (
	"os/exec"
	"strings"
)

func DetectNetworkManager() bool {
	cmd := exec.Command("systemctl", "is-active", "NetworkManager")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

func IsVPNActive() bool {
	cmd := exec.Command("nmcli", "-t", "-f", "TYPE,STATE", "connection", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			connType := strings.TrimSpace(parts[0])
			if connType == "vpn" || connType == "wireguard" {
				return true
			}
		}
	}
	return false
}

func GetCurrentDNS() ([]string, error) {
	cmd := exec.Command("resolvectl", "dns")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var servers []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// resolvectl output: "Link 2 (ens3): 8.8.8.8 8.8.4.4"
		if idx := strings.Index(line, "):"); idx >= 0 {
			addrs := strings.Fields(line[idx+2:])
			servers = append(servers, addrs...)
		}
	}
	return servers, nil
}
