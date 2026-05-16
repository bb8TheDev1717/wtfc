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

type searchDoneMsg struct {
	results []api.Project
	err     error
}

type Model struct {
	input    textinput.Model
	results  []api.Project
	cursor   int
	loading  bool
	err      error
	width    int
	height   int
	lastQuery string
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
	ti.Placeholder = "z.B. neovim, hyprland, quickshell..."
	ti.Focus()
	ti.CharLimit = 100
	return Model{input: ti}
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
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			q := strings.TrimSpace(m.input.Value())
			if q != "" && q != m.lastQuery {
				m.loading = true
				m.lastQuery = q
				m.cursor = 0
				m.results = nil
				m.err = nil
				return m, doSearch(q)
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		case "y":
			if len(m.results) > 0 {
				cmd := m.results[m.cursor].EnableCmd()
				exec.Command("wl-copy", cmd).Run()
			}
		}

	case searchDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.results = msg.results
		}
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
	sb.WriteString(styleDim.Render("Enter = suchen  ↑↓/jk = navigieren  y = kopieren  esc = raus") + "\n\n")
	sb.WriteString(m.input.View() + "\n\n")

	if m.loading {
		sb.WriteString(styleDim.Render("suche...") + "\n")
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(styleRed.Render("Fehler: "+m.err.Error()) + "\n")
		return sb.String()
	}

	if len(m.results) == 0 && m.lastQuery != "" {
		sb.WriteString(styleDim.Render("keine Ergebnisse") + "\n")
		return sb.String()
	}

	maxWidth := m.width - 4
	if maxWidth < 40 {
		maxWidth = 80
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
