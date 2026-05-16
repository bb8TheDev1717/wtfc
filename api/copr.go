package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const baseURL = "https://copr.fedorainfracloud.org/api_3"

type Project struct {
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
	u := fmt.Sprintf("%s/project/search/?query=%s&limit=%d", baseURL, url.QueryEscape(query), limit)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Items, nil
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
