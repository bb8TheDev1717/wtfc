package api

import (
	"fmt"
	"os/exec"
	"strings"
)

type DNFPackage struct {
	Name    string
	Summary string
}

func SearchDNF(query string) ([]DNFPackage, error) {
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf(`dnf repoquery -q --qf "%%{name}|||%%{summary}" --arch x86_64,noarch "*%s*" 2>/dev/null | sort -u | head -50`,
			strings.ReplaceAll(query, "'", "")),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pkgs []DNFPackage
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|||", 2)
		if len(parts) != 2 {
			continue
		}
		pkgs = append(pkgs, DNFPackage{
			Name:    strings.TrimSpace(parts[0]),
			Summary: strings.TrimSpace(parts[1]),
		})
	}
	return pkgs, nil
}
