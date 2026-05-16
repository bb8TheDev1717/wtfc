package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/raphi/wtfc/api"
)

type mode int

const (
	modeMenu mode = iota
	modeCoprSearch
	modeCoprBrowse
)

type searchDoneMsg struct {
	results []api.Project
	err     error
}

type installMsg struct {
	project  api.Project
	packages []string
	err      error
}

type installMultiMsg struct {
	commands []string
	err      error
}

type Model struct {
	input      textinput.Model
	results    []api.Project
	selected   map[string]bool
	cursor     int
	menuCursor int
	loading    bool
	err        error
	width      int
	height     int
	lastQuery  string
	mode       mode
}

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	styleGreen    = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	styleRed      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleBorder   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("99")).Padding(0, 1)
)

var menuItems = []string{
	"Search COPR repositories",
	"Search DNF packages",
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. neovim, hyprland, quickshell..."
	ti.CharLimit = 100
	return Model{input: ti, mode: modeMenu, selected: make(map[string]bool)}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch m.mode {
		case modeMenu:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up":
				if m.menuCursor > 0 {
					m.menuCursor--
				}
			case "down":
				if m.menuCursor < len(menuItems)-1 {
					m.menuCursor++
				}
			case "enter":
				switch m.menuCursor {
				case 0:
					m.mode = modeCoprSearch
					m.input.Placeholder = "e.g. neovim, hyprland, quickshell..."
					m.input.Focus()
					return m, textinput.Blink
				case 1:
					// DNF mode coming soon
				}
			}

		case modeCoprSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				if len(m.results) > 0 {
					m.mode = modeCoprBrowse
					m.input.Blur()
					return m, nil
				}
				m.mode = modeMenu
				m.input.Blur()
				m.input.SetValue("")
				return m, nil
			case "enter":
				q := strings.TrimSpace(m.input.Value())
				if q != "" {
					m.loading = true
					m.lastQuery = q
					m.cursor = 0
					m.results = nil
					m.err = nil
					m.mode = modeCoprBrowse
					m.input.Blur()
					return m, doCoprSearch(q)
				}
			}

		case modeCoprBrowse:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.mode = modeMenu
				m.input.Blur()
				m.input.SetValue("")
				m.results = nil
				m.lastQuery = ""
				return m, nil
			case "/":
				m.mode = modeCoprSearch
				m.input.Focus()
				return m, textinput.Blink
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < len(m.results)-1 {
					m.cursor++
				}
			case " ":
				if len(m.results) > 0 {
					key := m.results[m.cursor].FullName
					m.selected[key] = !m.selected[key]
				}
			case "y":
				if len(m.results) > 0 {
					exec.Command("wl-copy", m.results[m.cursor].EnableCmd()).Run()
				}
			case "i":
				if len(m.results) > 0 {
					if len(m.selected) > 0 {
						m.loading = true
						return m, fetchAndInstallMulti(m.results, m.selected)
					}
					p := m.results[m.cursor]
					m.loading = true
					return m, fetchAndInstall(p)
				}
			}
			return m, nil
		}

	case searchDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else if msg.results != nil {
			m.results = msg.results
		}
		return m, nil

	case installMultiMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		script := strings.Join(msg.commands, " && ") + "; echo; read -p 'Press Enter to return...'"
		return m, tea.ExecProcess(
			exec.Command("bash", "-c", script),
			func(err error) tea.Msg {
				return searchDoneMsg{results: m.results}
			},
		)

	case installMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		p := msg.project
		pkgs := strings.Join(msg.packages, " ")
		return m, tea.ExecProcess(
			exec.Command("bash", "-c",
				fmt.Sprintf("sudo dnf copr enable %s -y && sudo dnf install %s -y; echo; read -p 'Press Enter to return...'", p.FullName, pkgs),
			), func(err error) tea.Msg {
				return searchDoneMsg{results: m.results}
			},
		)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func fetchAndInstallMulti(results []api.Project, selected map[string]bool) tea.Cmd {
	return func() tea.Msg {
		var cmds []string
		for _, p := range results {
			if !selected[p.FullName] {
				continue
			}
			pkgs, err := api.GetPackages(p.OwnerName, p.Name)
			if err != nil || len(pkgs) == 0 {
				pkgs = []string{p.Name}
			}
			cmds = append(cmds, fmt.Sprintf("sudo dnf copr enable %s -y && sudo dnf install %s -y", p.FullName, strings.Join(pkgs, " ")))
		}
		return installMultiMsg{commands: cmds}
	}
}

func fetchAndInstall(p api.Project) tea.Cmd {
	return func() tea.Msg {
		pkgs, err := api.GetPackages(p.OwnerName, p.Name)
		if err != nil || len(pkgs) == 0 {
			pkgs = []string{p.Name}
		}
		return installMsg{project: p, packages: pkgs, err: err}
	}
}

func doCoprSearch(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := api.Search(query, 20)
		return searchDoneMsg{results: results, err: err}
	}
}

func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("wtfc — where the fuck is copr") + "\n")

	switch m.mode {
	case modeMenu:
		sb.WriteString(styleDim.Render("↑↓ = navigate  Enter = select  q = quit") + "\n\n")
		for i, item := range menuItems {
			if i == m.menuCursor {
				sb.WriteString(styleSelected.Render(" > "+item) + "\n")
			} else {
				sb.WriteString("   " + item + "\n")
			}
		}

	case modeCoprSearch:
		sb.WriteString(styleDim.Render("Enter = search  esc = menu") + "\n\n")
		sb.WriteString(m.input.View() + "\n")

	case modeCoprBrowse:
		sb.WriteString(styleDim.Render("↑↓ = navigate  space = select  i = install  y = copy  / = search  esc = menu  q = quit") + "\n\n")
		sb.WriteString(m.input.View() + "\n\n")

		if m.loading {
			sb.WriteString(styleDim.Render("searching...") + "\n")
			return sb.String()
		}
		if m.err != nil {
			sb.WriteString(styleRed.Render("error: "+m.err.Error()) + "\n")
			return sb.String()
		}
		if len(m.results) == 0 && m.lastQuery != "" {
			sb.WriteString(styleDim.Render("no results") + "\n")
			return sb.String()
		}

		for i, p := range m.results {
			distros := p.Distros()
			distroStr := strings.Join(distros, ", ")
			if len(distroStr) > 40 {
				distroStr = distroStr[:37] + "..."
			}
			desc := p.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			desc = strings.ReplaceAll(desc, "\r\n", " ")
			desc = strings.ReplaceAll(desc, "\n", " ")

			check := "  "
			if m.selected[p.FullName] {
				check = styleGreen.Render("✓ ")
			}
			line := fmt.Sprintf("%-30s  %-60s  %s", p.FullName, desc, styleDim.Render(distroStr))
			if i == m.cursor {
				sb.WriteString(styleSelected.Render(" > "+check+line) + "\n")
			} else {
				sb.WriteString("   " + check + line + "\n")
			}
		}

		if len(m.selected) > 0 {
			sb.WriteString("\n")
			var selNames []string
			for _, p := range m.results {
				if m.selected[p.FullName] {
					selNames = append(selNames, styleGreen.Render(p.FullName))
				}
			}
			sb.WriteString(styleBorder.Render(
				styleDim.Render("selected: ") + strings.Join(selNames, ", ") + "\n" +
					styleDim.Render("press i to install all"),
			) + "\n")
		} else if len(m.results) > 0 {
			sel := m.results[m.cursor]
			sb.WriteString("\n")
			sb.WriteString(styleBorder.Render(
				styleGreen.Render(sel.EnableCmd()) + "\n" +
					styleDim.Render("sudo dnf install "+sel.Name),
			) + "\n")
		}
	}

	return sb.String()
}
