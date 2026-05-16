package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bb8TheDev1717/wtfc-d/api"
)

type mode int

const (
	modeMenu mode = iota
	modeCoprSearch
	modeCoprBrowse
	modeDNFSearch
	modeDNFBrowse
	modeRemoveSearch
	modeRemoveBrowse
	modeUpdateBrowse
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

type fuzzyTriggerMsg struct {
	query string
	gen   int
}

type removeDoneMsg struct {
	results []api.InstalledPackage
	err     error
}

type updateDoneMsg struct {
	results []api.UpdatePackage
	err     error
}

type Model struct {
	input          textinput.Model
	results        []api.Project
	dnfResults     []api.DNFPackage
	removeResults  []api.InstalledPackage
	updateResults  []api.UpdatePackage
	selected       map[string]bool
	cursor         int
	menuCursor     int
	loading        bool
	err            error
	width          int
	height         int
	lastQuery      string
	mode           mode
	searchGen      int
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
	"  Remove installed packages",
	"  Check for updates",
}

var menuIcons = []string{"󰏗", "󰏖", "󰆴", "󰚰"}

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
				case 2:
					m.mode = modeRemoveSearch
					m.input.Placeholder = "search installed packages..."
					m.input.Focus()
					return m, textinput.Blink
				case 3:
					m.mode = modeUpdateBrowse
					m.loading = true
					m.updateResults = nil
					m.cursor = 0
					m.selected = make(map[string]bool)
					return m, doGetUpdates()
				}
			}

		case modeCoprSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeMenu
				m.input.Blur()
				m.input.SetValue("")
				m.results = nil
				return m, nil
			case "enter":
				if len(m.results) > 0 {
					m.mode = modeCoprBrowse
					m.input.Blur()
					return m, nil
				}
				q := strings.TrimSpace(m.input.Value())
				if q != "" {
					m.loading = true
					m.lastQuery = q
					m.cursor = 0
					m.results = nil
					m.err = nil
					return m, doCoprSearch(q)
				}
			}

		case modeDNFSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeMenu
				m.input.Blur()
				m.input.SetValue("")
				m.dnfResults = nil
				return m, nil
			case "enter":
				if len(m.dnfResults) > 0 {
					m.mode = modeDNFBrowse
					m.input.Blur()
					return m, nil
				}
				q := strings.TrimSpace(m.input.Value())
				if q != "" {
					m.loading = true
					m.lastQuery = q
					m.cursor = 0
					m.dnfResults = nil
					m.err = nil
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

		case modeUpdateBrowse:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.mode = modeMenu
				m.updateResults = nil
				m.selected = make(map[string]bool)
				return m, nil
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < len(m.updateResults)-1 {
					m.cursor++
				}
			case " ":
				if len(m.updateResults) > 0 {
					key := m.updateResults[m.cursor].Name
					m.selected[key] = !m.selected[key]
				}
			case "i":
				if len(m.updateResults) > 0 {
					var pkgs []string
					if len(m.selected) > 0 {
						for _, p := range m.updateResults {
							if m.selected[p.Name] {
								pkgs = append(pkgs, p.Name)
							}
						}
					} else {
						pkgs = []string{m.updateResults[m.cursor].Name}
					}
					return m, tea.ExecProcess(
						exec.Command("bash", "-c",
							fmt.Sprintf("sudo dnf upgrade %s -y; echo; read -p 'Press Enter to return...'", strings.Join(pkgs, " ")),
						), func(err error) tea.Msg {
							return updateDoneMsg{results: m.updateResults}
						},
					)
				}
			case "I":
				return m, tea.ExecProcess(
					exec.Command("bash", "-c", "sudo dnf upgrade -y; echo; read -p 'Press Enter to return...'"),
					func(err error) tea.Msg { return updateDoneMsg{results: m.updateResults} },
				)
			}
			return m, nil

		case modeRemoveSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeMenu
				m.input.Blur()
				m.input.SetValue("")
				m.removeResults = nil
				return m, nil
			case "enter":
				if len(m.removeResults) > 0 {
					m.mode = modeRemoveBrowse
					m.input.Blur()
					return m, nil
				}
				q := strings.TrimSpace(m.input.Value())
				m.loading = true
				m.lastQuery = q
				m.cursor = 0
				m.removeResults = nil
				m.err = nil
				return m, doRemoveSearch(q)
			}

		case modeRemoveBrowse:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.mode = modeMenu
				m.input.Blur()
				m.input.SetValue("")
				m.removeResults = nil
				m.lastQuery = ""
				m.selected = make(map[string]bool)
				return m, nil
			case "/":
				m.mode = modeRemoveSearch
				m.input.Focus()
				return m, textinput.Blink
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down":
				if m.cursor < len(m.removeResults)-1 {
					m.cursor++
				}
			case " ":
				if len(m.removeResults) > 0 {
					key := m.removeResults[m.cursor].Name
					m.selected[key] = !m.selected[key]
				}
			case "i":
				if len(m.removeResults) > 0 {
					var pkgs []string
					if len(m.selected) > 0 {
						for _, p := range m.removeResults {
							if m.selected[p.Name] {
								pkgs = append(pkgs, p.Name)
							}
						}
					} else {
						pkgs = []string{m.removeResults[m.cursor].Name}
					}
					return m, tea.ExecProcess(
						exec.Command("bash", "-c",
							fmt.Sprintf("sudo dnf remove %s -y; echo; read -p 'Press Enter to return...'", strings.Join(pkgs, " ")),
						), func(err error) tea.Msg {
							return removeDoneMsg{results: m.removeResults}
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

	case updateDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.updateResults = msg.results
		}
		return m, nil

	case removeDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.removeResults = msg.results
		}
		return m, nil

	case fuzzyTriggerMsg:
		return m.handleFuzzyTrigger(msg)

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

	prevVal := m.input.Value()
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	newVal := m.input.Value()

	if newVal != prevVal && newVal != "" && (m.mode == modeCoprSearch || m.mode == modeDNFSearch || m.mode == modeRemoveSearch) {
		m.searchGen++
		gen := m.searchGen
		debounce := tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
			return fuzzyTriggerMsg{query: newVal, gen: gen}
		})
		return m, tea.Batch(inputCmd, debounce)
	}
	return m, inputCmd
}

func (m *Model) handleFuzzyTrigger(msg fuzzyTriggerMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.searchGen {
		return m, nil
	}
	q := strings.TrimSpace(msg.query)
	if q == "" {
		return m, nil
	}
	m.loading = true
	m.lastQuery = q
	m.cursor = 0
	switch m.mode {
	case modeCoprSearch:
		m.results = nil
		return m, doCoprSearch(q)
	case modeRemoveSearch:
		m.removeResults = nil
		return m, doRemoveSearch(q)
	default:
		m.dnfResults = nil
		return m, doDNFSearch(q)
	}
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

func doGetUpdates() tea.Cmd {
	return func() tea.Msg {
		results, err := api.GetUpdates()
		return updateDoneMsg{results: results, err: err}
	}
}

func doRemoveSearch(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := api.GetInstalled(query)
		return removeDoneMsg{results: results, err: err}
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
	case modeCoprSearch, modeDNFSearch, modeRemoveSearch:
		return m.viewSearch(w, h)
	case modeCoprBrowse, modeDNFBrowse, modeRemoveBrowse, modeUpdateBrowse:
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
	isRemove := m.mode == modeRemoveSearch
	name, title := "wtfd", "DNF Search"
	if isCopr {
		name, title = "wtfc", "COPR Search"
	} else if isRemove {
		name, title = "wtfd", "Remove"
	}

	panelW := w - 8

	var sb strings.Builder
	sb.WriteString(styleTitle.Render(name) + " " + styleDim.Render("/ "+title) + "\n\n")
	sb.WriteString(m.input.View() + "\n\n")

	if m.loading {
		sb.WriteString(styleDim.Render("searching...") + "\n")
	} else if isCopr && len(m.results) > 0 {
		sb.WriteString(styleHint.Render(fmt.Sprintf("%d results — Enter to navigate", len(m.results))) + "\n")
		for i, p := range m.results {
			if i >= 5 {
				sb.WriteString(styleDim.Render(fmt.Sprintf("  ... and %d more", len(m.results)-5)) + "\n")
				break
			}
			sb.WriteString("  " + styleGreen.Render(p.FullName) + "  " + styleDim.Render(truncate(p.Description, 50)) + "\n")
		}
	} else if !isCopr && !isRemove && len(m.dnfResults) > 0 {
		sb.WriteString(styleHint.Render(fmt.Sprintf("%d results — Enter to navigate", len(m.dnfResults))) + "\n")
		for i, p := range m.dnfResults {
			if i >= 5 {
				sb.WriteString(styleDim.Render(fmt.Sprintf("  ... and %d more", len(m.dnfResults)-5)) + "\n")
				break
			}
			sb.WriteString("  " + styleGreen.Render(p.Name) + "  " + styleDim.Render(truncate(p.Summary, 50)) + "\n")
		}
	} else if isRemove && len(m.removeResults) > 0 {
		sb.WriteString(styleHint.Render(fmt.Sprintf("%d results — Enter to navigate", len(m.removeResults))) + "\n")
		for i, p := range m.removeResults {
			if i >= 5 {
				sb.WriteString(styleDim.Render(fmt.Sprintf("  ... and %d more", len(m.removeResults)-5)) + "\n")
				break
			}
			sb.WriteString("  " + styleGreen.Render(p.Name) + "  " + styleDim.Render(p.Version) + "\n")
		}
	}

	sb.WriteString("\n" + styleHint.Render("Enter = browse results  ·  esc = menu  ·  ctrl+c = quit"))

	panel := stylePanel.Width(panelW).Render(sb.String())
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, panel)
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", " "), "\n", " ")
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func (m Model) viewBrowse(w, h int) string {
	isCopr := m.mode == modeCoprBrowse
	isRemove := m.mode == modeRemoveBrowse
	isUpdate := m.mode == modeUpdateBrowse
	name, title := "wtfd", "DNF Results"
	if isCopr {
		name, title = "wtfc", "COPR Results"
	} else if isRemove {
		name, title = "wtfd", "Remove"
	} else if isUpdate {
		name, title = "wtfd", "Updates"
	}

	panelW := w - 4

	// Header
	var header strings.Builder
	header.WriteString(styleTitle.Render(name) + " " + styleDim.Render("/ "+title) + "\n")
	header.WriteString(m.input.View() + "\n")
	hints := "↑↓ navigate  ·  space select  ·  i install  ·  / search  ·  esc menu  ·  q quit"
	if isCopr {
		hints = "↑↓ navigate  ·  space select  ·  i install  ·  y copy  ·  / search  ·  esc menu"
	} else if isRemove {
		hints = "↑↓ navigate  ·  space select  ·  i remove  ·  / search  ·  esc menu  ·  q quit"
	} else if isUpdate {
		hints = "↑↓ navigate  ·  space select  ·  i update selected  ·  I update all  ·  esc menu"
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
	} else if isRemove {
		m.renderRemoveResults(&body, panelW)
	} else if isUpdate {
		m.renderUpdateResults(&body, panelW)
	} else {
		m.renderDNFResults(&body, panelW)
	}

	// Footer
	footer := m.renderFooter(isCopr, isRemove, isUpdate, panelW)

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

func (m Model) renderUpdateResults(sb *strings.Builder, panelW int) {
	if len(m.updateResults) == 0 && !m.loading {
		sb.WriteString("\n  " + styleGreen.Render("everything is up to date") + "\n")
		return
	}
	maxSummary := panelW - 50
	if maxSummary < 20 {
		maxSummary = 20
	}
	for i, p := range m.updateResults {
		summary := p.Summary
		if len(summary) > maxSummary {
			summary = summary[:maxSummary-3] + "..."
		}
		check := "  "
		if m.selected[p.Name] {
			check = styleGreen.Render("✓ ")
		}
		line := fmt.Sprintf("%-28s  %-14s  %s", p.Name, styleDim.Render("→ "+p.NewVersion), summary)
		if i == m.cursor {
			sb.WriteString(styleSelected.Width(panelW).Render(" › "+check+line) + "\n")
		} else {
			sb.WriteString("   " + check + line + "\n")
		}
	}
}

func (m Model) renderRemoveResults(sb *strings.Builder, panelW int) {
	if len(m.removeResults) == 0 && m.lastQuery != "" {
		sb.WriteString("\n  " + styleDim.Render("no results") + "\n")
		return
	}
	maxSummary := panelW - 50
	if maxSummary < 20 {
		maxSummary = 20
	}
	for i, p := range m.removeResults {
		summary := p.Summary
		if len(summary) > maxSummary {
			summary = summary[:maxSummary-3] + "..."
		}
		check := "  "
		if m.selected[p.Name] {
			check = styleRed.Render("✓ ")
		}
		line := fmt.Sprintf("%-28s  %-14s  %s", p.Name, styleDim.Render(p.Version), summary)
		if i == m.cursor {
			sb.WriteString(styleSelected.Width(panelW).Render(" › "+check+line) + "\n")
		} else {
			sb.WriteString("   " + check + line + "\n")
		}
	}
}

func (m Model) renderFooter(isCopr, isRemove, isUpdate bool, panelW int) string {
	if len(m.selected) > 0 {
		var selNames []string
		action := "install"
		col := styleGreen
		if isRemove {
			action = "remove"
			col = styleRed
		}
		if isCopr {
			for _, p := range m.results {
				if m.selected[p.FullName] {
					selNames = append(selNames, col.Render(p.FullName))
				}
			}
		} else if isRemove {
			for _, p := range m.removeResults {
				if m.selected[p.Name] {
					selNames = append(selNames, col.Render(p.Name))
				}
			}
		} else if isUpdate {
			for _, p := range m.updateResults {
				if m.selected[p.Name] {
					selNames = append(selNames, col.Render(p.Name))
				}
			}
		} else {
			for _, p := range m.dnfResults {
				if m.selected[p.Name] {
					selNames = append(selNames, col.Render(p.Name))
				}
			}
		}
		return styleFooter.Width(panelW).Render(
			styleDim.Render("selected: ")+strings.Join(selNames, styleDim.Render(", "))+"\n"+
				styleHint.Render("press i to "+action+" all"),
		)
	}

	if isCopr && len(m.results) > 0 {
		sel := m.results[m.cursor]
		return styleFooter.Width(panelW).Render(
			styleGreen.Render(sel.EnableCmd()) + "\n" +
				styleDim.Render("sudo dnf install "+sel.Name),
		)
	}
	if isRemove && len(m.removeResults) > 0 {
		return styleFooter.Width(panelW).Render(
			styleRed.Render("sudo dnf remove " + m.removeResults[m.cursor].Name),
		)
	}
	if isUpdate && len(m.updateResults) > 0 {
		return styleFooter.Width(panelW).Render(
			styleGreen.Render("sudo dnf upgrade "+m.updateResults[m.cursor].Name) + "\n" +
				styleHint.Render("I = upgrade everything"),
		)
	}
	if !isCopr && !isRemove && !isUpdate && len(m.dnfResults) > 0 {
		return styleFooter.Width(panelW).Render(
			styleGreen.Render("sudo dnf install " + m.dnfResults[m.cursor].Name),
		)
	}
	return ""
}
