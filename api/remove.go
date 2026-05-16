package api

import (
	"fmt"
	"os/exec"
	"strings"
)

type InstalledPackage struct {
	Name    string
	Version string
	Summary string
}

func GetInstalled(query string) ([]InstalledPackage, error) {
	filter := "*"
	if query != "" {
		filter = fmt.Sprintf("*%s*", strings.ReplaceAll(query, "'", ""))
	}
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf(`dnf repoquery --installed -q --qf "%%{name}|||%%{version}|||%%{summary}\n" "%s" 2>/dev/null | sort -u`, filter),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pkgs []InstalledPackage
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|||", 3)
		if len(parts) != 3 {
			continue
		}
		pkgs = append(pkgs, InstalledPackage{
			Name:    strings.TrimSpace(parts[0]),
			Version: strings.TrimSpace(parts[1]),
			Summary: strings.TrimSpace(parts[2]),
		})
	}
	return pkgs, nil
}
