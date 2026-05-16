package api

import (
	"os/exec"
	"strings"
)

type UpdatePackage struct {
	Name       string
	NewVersion string
	Summary    string
}

func GetUpdates() ([]UpdatePackage, error) {
	cmd := exec.Command("bash", "-c",
		`dnf repoquery --upgrades -q --qf "%{name}|||%{version}|||%{summary}\n" 2>/dev/null | sort -u`,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pkgs []UpdatePackage
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|||", 3)
		if len(parts) != 3 {
			continue
		}
		pkgs = append(pkgs, UpdatePackage{
			Name:       strings.TrimSpace(parts[0]),
			NewVersion: strings.TrimSpace(parts[1]),
			Summary:    strings.TrimSpace(parts[2]),
		})
	}
	return pkgs, nil
}
