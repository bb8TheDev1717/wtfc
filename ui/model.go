package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/raphi/wtfc-d/api"
)

type mode int

const (
	modeMenu mode = iota
	modeCoprSearch
	modeCoprBrowse
	modeDNFSearch
	modeDNFBrowse
)

type searchDoneMsg struct {
	results []api.Project
	err     error
}

type dnfSearchDoneMsg struct {
	results []api.DNFPackage
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
	dnfResults []api.DNFPackage
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
	colorPurple = lipgloss.Color("99")
	colorGreen  = lipgloss.Color("76")
	colorDim    = lipgloss.Color("243")
	colorRed    = lipgloss.Color("196")
	colorBg     = lipgloss.Color("235")
	colorHover  = lipgloss.Color("236")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)

	styleSelected = lipgloss.NewStyle().
			Background(colorHover).
			Bold(true).
			Foreground(lipgloss.Color("255"))

	styleDim   = lipgloss.NewStyle().Foreground(colorDim)
	styleGreen = lipgloss.NewStyle().Foreground(colorGreen)
	styleRed   = lipgloss.NewStyle().Foreground(colorRed)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple).
			Padding(1, 2)

	styleFooter = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1)

	styleHint = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)
)

var menuItems = []string{
	"  Search COPR repositories",
	"  Search DNF packages",
}

var menuIcons = []string{"󰏗", "󰏖"}

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
					m.mode = modeDNFSearch
					m.input.Placeholder = "e.g. firefox, git, htop..."
					m.input.Focus()
					return m, textinput.Blink
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

		case modeDNFSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				if len(m.dnfResults) > 0 {
					m.mode = modeDNFBrowse
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
					m.dnfResults = nil
					m.err = nil
					m.mode = modeDNFBrowse
					m.input.Blur()
					return m, doDNFSearch(q)
				}
			}

		case modeDNFBrowse:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.mode = modeMenu
				m.input.Blur()
				m.input.SetValue("")
				m.dnfResults = nil
				m.lastQuery = ""
				m.selected = make(map[string]bool)
				return m, nil
			case "/":
				m.mode = modeDNFSearch
				m.input.Focus()
				return m, textinput.Blink
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < len(m.dnfResults)-1 {
					m.cursor++
				}
			case " ":
				if len(m.dnfResults) > 0 {
					key := m.dnfResults[m.cursor].Name
					m.selected[key] = !m.selected[key]
				}
			case "i":
				if len(m.dnfResults) > 0 {
					var pkgs []string
					if len(m.selected) > 0 {
						for _, p := range m.dnfResults {
							if m.selected[p.Name] {
								pkgs = append(pkgs, p.Name)
							}
						}
					} else {
						pkgs = []string{m.dnfResults[m.cursor].Name}
					}
					return m, tea.ExecProcess(
						exec.Command("bash", "-c",
							fmt.Sprintf("sudo dnf install %s -y; echo; read -p 'Press Enter to return...'", strings.Join(pkgs, " ")),
						), func(err error) tea.Msg {
							return dnfSearchDoneMsg{results: m.dnfResults}
						},
					)
				}
			}
			return m, nil

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
				m.selected = make(map[string]bool)
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

	case dnfSearchDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.dnfResults = msg.results
		}
		return m, nil

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

func doDNFSearch(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := api.SearchDNF(query)
		return dnfSearchDoneMsg{results: results, err: err}
	}
}

func doCoprSearch(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := api.Search(query, 20)
		return searchDoneMsg{results: results, err: err}
	}
}

func (m Model) View() string {
	w := m.width
	h := m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	switch m.mode {
	case modeMenu:
		return m.viewMenu(w, h)
	case modeCoprSearch, modeDNFSearch:
		return m.viewSearch(w, h)
	case modeCoprBrowse, modeDNFBrowse:
		return m.viewBrowse(w, h)
	}
	return ""
}

func (m Model) viewMenu(w, h int) string {
	panelW := 44

	header := lipgloss.JoinVertical(lipgloss.Center,
		styleTitle.Render("wtfc-d"),
		styleSubtitle.Render("where the fuck is copr & dnf"),
	)

	var items strings.Builder
	for i, item := range menuItems {
		icon := menuIcons[i]
		label := icon + item
		if i == m.menuCursor {
			items.WriteString(styleSelected.Width(panelW-6).Render(" › "+label) + "\n")
		} else {
			items.WriteString(styleDim.Render("   "+label) + "\n")
		}
	}

	hints := styleHint.Render("↑↓ navigate  ·  Enter select  ·  q quit")

	content := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		strings.TrimRight(items.String(), "\n"),
		"",
		hints,
	)

	panel := stylePanel.Width(panelW).Render(content)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, panel)
}

func (m Model) viewSearch(w, h int) string {
	isCopr := m.mode == modeCoprSearch
	name, title := "wtfd", "DNF Search"
	if isCopr {
		name, title = "wtfc", "COPR Search"
	}

	header := lipgloss.JoinVertical(lipgloss.Left,
		styleTitle.Render(name)+" "+styleDim.Render("/ "+title),
		"",
		m.input.View(),
		"",
		styleHint.Render("Enter = search  ·  esc = back  ·  ctrl+c = quit"),
	)

	panel := stylePanel.Width(w - 8).Render(header)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, panel)
}

func (m Model) viewBrowse(w, h int) string {
	isCopr := m.mode == modeCoprBrowse
	name, title := "wtfd", "DNF Results"
	if isCopr {
		name, title = "wtfc", "COPR Results"
	}

	panelW := w - 4

	// Header
	var header strings.Builder
	header.WriteString(styleTitle.Render(name) + " " + styleDim.Render("/ "+title) + "\n")
	header.WriteString(m.input.View() + "\n")
	hints := "↑↓ navigate  ·  space select  ·  i install  ·  / search  ·  esc menu  ·  q quit"
	if isCopr {
		hints = "↑↓ navigate  ·  space select  ·  i install  ·  y copy  ·  / search  ·  esc menu"
	}
	header.WriteString(styleHint.Render(hints))

	headerPanel := stylePanel.Width(panelW).Render(header.String())

	// Body
	var body strings.Builder
	if m.loading {
		body.WriteString("\n  " + styleDim.Render("searching...") + "\n")
	} else if m.err != nil {
		body.WriteString("\n  " + styleRed.Render("error: "+m.err.Error()) + "\n")
	} else if isCopr {
		m.renderCoprResults(&body, panelW)
	} else {
		m.renderDNFResults(&body, panelW)
	}

	// Footer
	footer := m.renderFooter(isCopr, panelW)

	full := lipgloss.JoinVertical(lipgloss.Left,
		headerPanel,
		body.String(),
		footer,
	)
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, full)
}

func (m Model) renderCoprResults(sb *strings.Builder, panelW int) {
	if len(m.results) == 0 && m.lastQuery != "" {
		sb.WriteString("\n  " + styleDim.Render("no results") + "\n")
		return
	}
	maxDesc := panelW - 40
	if maxDesc < 20 {
		maxDesc = 20
	}
	for i, p := range m.results {
		distros := p.Distros()
		distroStr := strings.Join(distros, ", ")
		if len(distroStr) > 35 {
			distroStr = distroStr[:32] + "..."
		}
		desc := p.Description
		if len(desc) > maxDesc {
			desc = desc[:maxDesc-3] + "..."
		}
		desc = strings.ReplaceAll(strings.ReplaceAll(desc, "\r\n", " "), "\n", " ")

		check := "  "
		if m.selected[p.FullName] {
			check = styleGreen.Render("✓ ")
		}
		line := fmt.Sprintf("%-28s  %-*s  %s", p.FullName, maxDesc, desc, styleDim.Render(distroStr))
		if i == m.cursor {
			sb.WriteString(styleSelected.Width(panelW).Render(" › "+check+line) + "\n")
		} else {
			sb.WriteString("   " + check + line + "\n")
		}
	}
}

func (m Model) renderDNFResults(sb *strings.Builder, panelW int) {
	if len(m.dnfResults) == 0 && m.lastQuery != "" {
		sb.WriteString("\n  " + styleDim.Render("no results") + "\n")
		return
	}
	maxSummary := panelW - 36
	if maxSummary < 20 {
		maxSummary = 20
	}
	for i, p := range m.dnfResults {
		summary := p.Summary
		if len(summary) > maxSummary {
			summary = summary[:maxSummary-3] + "..."
		}
		check := "  "
		if m.selected[p.Name] {
			check = styleGreen.Render("✓ ")
		}
		line := fmt.Sprintf("%-28s  %s", p.Name, summary)
		if i == m.cursor {
			sb.WriteString(styleSelected.Width(panelW).Render(" › "+check+line) + "\n")
		} else {
			sb.WriteString("   " + check + line + "\n")
		}
	}
}

func (m Model) renderFooter(isCopr bool, panelW int) string {
	if len(m.selected) > 0 {
		var selNames []string
		if isCopr {
			for _, p := range m.results {
				if m.selected[p.FullName] {
					selNames = append(selNames, styleGreen.Render(p.FullName))
				}
			}
		} else {
			for _, p := range m.dnfResults {
				if m.selected[p.Name] {
					selNames = append(selNames, styleGreen.Render(p.Name))
				}
			}
		}
		return styleFooter.Width(panelW).Render(
			styleDim.Render("selected: ") + strings.Join(selNames, styleDim.Render(", ")) + "\n" +
				styleHint.Render("press i to install all"),
		)
	}

	if isCopr && len(m.results) > 0 {
		sel := m.results[m.cursor]
		return styleFooter.Width(panelW).Render(
			styleGreen.Render(sel.EnableCmd()) + "\n" +
				styleDim.Render("sudo dnf install "+sel.Name),
		)
	}
	if !isCopr && len(m.dnfResults) > 0 {
		return styleFooter.Width(panelW).Render(
			styleGreen.Render("sudo dnf install " + m.dnfResults[m.cursor].Name),
		)
	}
	return ""
}
