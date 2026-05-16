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
	modeSearch mode = iota
	modeBrowse
)

type searchDoneMsg struct {
	results []api.Project
	err     error
}

type Model struct {
	input     textinput.Model
	results   []api.Project
	cursor    int
	loading   bool
	err       error
	width     int
	height    int
	lastQuery string
	mode      mode
}

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	styleGreen    = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	styleRed      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleBorder   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("99")).Padding(0, 1)
)

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. neovim, hyprland, quickshell..."
	ti.Focus()
	ti.CharLimit = 100
	return Model{input: ti, mode: modeSearch}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.mode == modeSearch {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				if len(m.results) > 0 {
					m.mode = modeBrowse
					m.input.Blur()
					return m, nil
				}
				return m, tea.Quit
			case "enter":
				q := strings.TrimSpace(m.input.Value())
				if q != "" {
					m.loading = true
					m.lastQuery = q
					m.cursor = 0
					m.results = nil
					m.err = nil
					m.mode = modeBrowse
					m.input.Blur()
					return m, doSearch(q)
				}
			}
		} else {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc", "/":
				m.mode = modeSearch
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
			case "y":
				if len(m.results) > 0 {
					exec.Command("wl-copy", m.results[m.cursor].EnableCmd()).Run()
				}
			case "i":
				if len(m.results) > 0 {
					p := m.results[m.cursor]
					return m, tea.ExecProcess(
						exec.Command("bash", "-c",
							fmt.Sprintf("sudo dnf copr enable %s -y && sudo dnf install %s -y; echo; read -p 'Press Enter to return...'", p.FullName, p.Name),
						), func(err error) tea.Msg {
							return searchDoneMsg{results: m.results}
						},
					)
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
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := api.Search(query, 20)
		return searchDoneMsg{results: results, err: err}
	}
}

func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("wtfc — where the fuck is copr") + "\n")

	if m.mode == modeSearch {
		sb.WriteString(styleDim.Render("Enter = search  esc = browse") + "\n\n")
	} else {
		sb.WriteString(styleDim.Render("↑↓ = navigate  i = install  y = copy  / = new search  q = quit") + "\n\n")
	}

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

		line := fmt.Sprintf("%-30s  %-60s  %s", p.FullName, desc, styleDim.Render(distroStr))

		if i == m.cursor {
			sb.WriteString(styleSelected.Render(" > "+line) + "\n")
		} else {
			sb.WriteString("   " + line + "\n")
		}
	}

	if len(m.results) > 0 {
		selected := m.results[m.cursor]
		sb.WriteString("\n")
		sb.WriteString(styleBorder.Render(
			styleGreen.Render(selected.EnableCmd()) + "\n" +
				styleDim.Render("sudo dnf install "+selected.Name),
		) + "\n")
	}

	return sb.String()
}
