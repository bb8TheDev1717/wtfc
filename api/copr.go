package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const baseURL = "https://copr.fedorainfracloud.org/api_3"

type Project struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	OwnerName   string            `json:"ownername"`
	FullName    string            `json:"full_name"`
	Description string            `json:"description"`
	Homepage    string            `json:"homepage"`
	ChootRepos  map[string]string `json:"chroot_repos"`
}

type SearchResult struct {
	Items []Project `json:"items"`
	Meta  struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

func Search(query string, limit int) ([]Project, error) {
	u := fmt.Sprintf("%s/project/search/?query=%s&limit=50", baseURL, url.QueryEscape(query))
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var active []Project
	for _, p := range result.Items {
		if p.supportsCurrentSystem() {
			active = append(active, p)
		}
	}

	q := strings.ToLower(query)
	sort.SliceStable(active, func(i, j int) bool {
		ni := strings.Contains(strings.ToLower(active[i].Name), q)
		nj := strings.Contains(strings.ToLower(active[j].Name), q)
		if ni != nj {
			return ni
		}
		return active[i].ID > active[j].ID
	})

	if len(active) > limit {
		active = active[:limit]
	}
	return active, nil
}

var supportedDistros = []string{"fedora-42", "fedora-43", "fedora-44", "fedora-rawhide"}

func (p Project) supportsCurrentSystem() bool {
	for chroot := range p.ChootRepos {
		for _, d := range supportedDistros {
			if strings.HasPrefix(chroot, d) {
				return true
			}
		}
	}
	return false
}

func (p Project) Distros() []string {
	seen := map[string]bool{}
	var distros []string
	for chroot := range p.ChootRepos {
		// chroot format: "fedora-44-x86_64" → extract "fedora-44"
		for i := len(chroot) - 1; i >= 0; i-- {
			if chroot[i] == '-' {
				d := chroot[:i]
				if !seen[d] {
					seen[d] = true
					distros = append(distros, d)
				}
				break
			}
		}
	}
	return distros
}

func (p Project) EnableCmd() string {
	return fmt.Sprintf("sudo dnf copr enable %s", p.FullName)
}

type Package struct {
	Name string `json:"name"`
}

func GetPackages(owner, project string) ([]string, error) {
	u := fmt.Sprintf("%s/package/list?ownername=%s&projectname=%s&limit=50", baseURL, url.QueryEscape(owner), url.QueryEscape(project))
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []Package `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	names := make([]string, len(result.Items))
	for i, pkg := range result.Items {
		names[i] = pkg.Name
	}
	return names, nil
}
