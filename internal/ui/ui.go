package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"mkepub/internal/catalog"
)

type screen int

const (
	screenAuthor screen = iota
	screenWork
	screenProgress
	screenMail
)

// ConvertFunc is called for each selected work. Returns output path or error.
type ConvertFunc func(work *catalog.Work) (string, error)

// MailFunc is called for each selected epub path.
type MailFunc func(path string) error

type progressItem struct {
	title string
	done  bool
	err   error
}

type convertDoneMsg struct {
	index int
	path  string
	err   error
}

type mailDoneMsg struct {
	index int
	err   error
}

type Model struct {
	screen  screen
	authors []*catalog.Author
	filtered []*catalog.Author

	// author screen
	input        textinput.Model
	authorCursor int
	authorOffset int

	// work screen
	selectedAuthor *catalog.Author
	workCursor     int
	workOffset     int
	workSelected   map[int]bool

	// progress screen
	items     []progressItem
	doneCount int

	convertFn ConvertFunc
	mailFn    MailFunc
	outputDir string

	// mail screen
	mailItems    []string // epub paths
	mailCursor   int
	mailOffset   int
	mailSelected map[int]bool
	mailResults  []progressItem

	width  int
	height int
}

// Styles
var (
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	styleSelect = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleSpin   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)
	styleHelp = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func New(authors []*catalog.Author, fn ConvertFunc, mailFn MailFunc, outputDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "作家名を入力..."
	ti.Prompt = "❯ "
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40

	return Model{
		screen:       screenAuthor,
		authors:      authors,
		filtered:     authors,
		input:        ti,
		workSelected: map[int]bool{},
		convertFn:    fn,
		mailFn:       mailFn,
		outputDir:    outputDir,
		mailSelected: map[int]bool{},
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case mailDoneMsg:
		if msg.index < len(m.mailResults) {
			m.mailResults[msg.index].done = true
			m.mailResults[msg.index].err = msg.err
			m.doneCount++
		}
		if m.doneCount < len(m.mailResults) {
			return m, m.mailNext(m.doneCount)
		}
		return m, nil

	case convertDoneMsg:
		if msg.index < len(m.items) {
			m.items[msg.index].done = true
			m.items[msg.index].err = msg.err
			if msg.err == nil {
				m.items[msg.index].title = msg.path
			}
			m.doneCount++
		}
		if m.doneCount < len(m.items) {
			return m, m.convertNext(m.doneCount)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.screen == screenAuthor {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.filtered = catalog.FilterAuthors(m.authors, m.input.Value())
		m.authorCursor = clamp(m.authorCursor, 0, len(m.filtered)-1)
		m.authorOffset = clamp(m.authorOffset, 0, len(m.filtered)-1)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {

	case screenAuthor:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyRunes:
			if m.input.Value() == "" {
				switch msg.String() {
				case "q":
					return m, tea.Quit
				case "m":
					if m.mailFn != nil {
						m.mailItems = listEpubs(m.outputDir)
						m.mailCursor = 0
						m.mailOffset = 0
						m.mailSelected = map[int]bool{}
						m.mailResults = nil
						m.doneCount = 0
						m.screen = screenMail
						return m, nil
					}
				}
			}
		case tea.KeyUp:
			m.authorCursor = clamp(m.authorCursor-1, 0, len(m.filtered)-1)
			if m.authorCursor < m.authorOffset {
				m.authorOffset = m.authorCursor
			}
			return m, nil
		case tea.KeyDown:
			m.authorCursor = clamp(m.authorCursor+1, 0, len(m.filtered)-1)
			if m.authorCursor >= m.authorOffset+m.pageSize() {
				m.authorOffset = m.authorCursor - m.pageSize() + 1
			}
			return m, nil
		case tea.KeyLeft:
			ps := m.pageSize()
			m.authorOffset = clamp(m.authorOffset-ps, 0, len(m.filtered)-1)
			m.authorCursor = m.authorOffset
			return m, nil
		case tea.KeyRight:
			ps := m.pageSize()
			m.authorOffset = clamp(m.authorOffset+ps, 0, len(m.filtered)-1)
			m.authorCursor = m.authorOffset
			return m, nil
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				m.selectedAuthor = m.filtered[m.authorCursor]
				m.workCursor = 0
				m.workSelected = map[int]bool{}
				m.screen = screenWork
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.filtered = catalog.FilterAuthors(m.authors, m.input.Value())
		m.authorCursor = 0
		m.authorOffset = 0
		return m, cmd

	case screenWork:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			m.screen = screenAuthor
			return m, nil
		case tea.KeyUp:
			m.workCursor = clamp(m.workCursor-1, 0, len(m.selectedAuthor.Works)-1)
			if m.workCursor < m.workOffset {
				m.workOffset = m.workCursor
			}
		case tea.KeyDown:
			m.workCursor = clamp(m.workCursor+1, 0, len(m.selectedAuthor.Works)-1)
			if m.workCursor >= m.workOffset+m.pageSize() {
				m.workOffset = m.workCursor - m.pageSize() + 1
			}
		case tea.KeyLeft:
			ps := m.pageSize()
			m.workOffset = clamp(m.workOffset-ps, 0, len(m.selectedAuthor.Works)-1)
			m.workCursor = m.workOffset
		case tea.KeyRight:
			ps := m.pageSize()
			m.workOffset = clamp(m.workOffset+ps, 0, len(m.selectedAuthor.Works)-1)
			m.workCursor = m.workOffset
		case tea.KeySpace:
			m.workSelected[m.workCursor] = !m.workSelected[m.workCursor]
		case tea.KeyEnter:
			selected := m.selectedWorks()
			if len(selected) == 0 {
				return m, nil
			}
			m.items = make([]progressItem, len(selected))
			for i, w := range selected {
				m.items[i] = progressItem{title: w.Title}
			}
			m.doneCount = 0
			m.screen = screenProgress
			return m, m.convertNext(0)
		}
		return m, nil

	case screenProgress:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.doneCount >= len(m.items) {
				m.screen = screenAuthor
				m.items = nil
				m.doneCount = 0
				m.workSelected = map[int]bool{}
			}
		}
		return m, nil

	case screenMail:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			if m.doneCount >= len(m.mailResults) {
				m.screen = screenAuthor
			}
			return m, nil
		case tea.KeyUp:
			m.mailCursor = clamp(m.mailCursor-1, 0, len(m.mailItems)-1)
			if m.mailCursor < m.mailOffset {
				m.mailOffset = m.mailCursor
			}
		case tea.KeyDown:
			m.mailCursor = clamp(m.mailCursor+1, 0, len(m.mailItems)-1)
			if m.mailCursor >= m.mailOffset+m.pageSize() {
				m.mailOffset = m.mailCursor - m.pageSize() + 1
			}
		case tea.KeyLeft:
			ps := m.pageSize()
			m.mailOffset = clamp(m.mailOffset-ps, 0, len(m.mailItems)-1)
			m.mailCursor = m.mailOffset
		case tea.KeyRight:
			ps := m.pageSize()
			m.mailOffset = clamp(m.mailOffset+ps, 0, len(m.mailItems)-1)
			m.mailCursor = m.mailOffset
		case tea.KeySpace:
			m.mailSelected[m.mailCursor] = !m.mailSelected[m.mailCursor]
		case tea.KeyEnter:
			if len(m.mailResults) > 0 {
				// already sending or done — do nothing
				return m, nil
			}
			selected := m.selectedMails()
			if len(selected) == 0 {
				return m, nil
			}
			m.mailResults = make([]progressItem, len(selected))
			for i, p := range selected {
				m.mailResults[i] = progressItem{title: filepath.Base(p)}
			}
			m.doneCount = 0
			return m, m.mailNext(0)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) mailNext(index int) tea.Cmd {
	selected := m.selectedMails()
	if index >= len(selected) {
		return nil
	}
	path := selected[index]
	return func() tea.Msg {
		err := m.mailFn(path)
		return mailDoneMsg{index: index, err: err}
	}
}

func (m Model) selectedMails() []string {
	var result []string
	for i, p := range m.mailItems {
		if m.mailSelected[i] {
			result = append(result, p)
		}
	}
	return result
}

func listEpubs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var result []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".epub" {
			result = append(result, filepath.Join(dir, e.Name()))
		}
	}
	return result
}

func (m Model) convertNext(index int) tea.Cmd {
	selected := m.selectedWorks()
	if index >= len(selected) {
		return nil
	}
	work := selected[index]
	return func() tea.Msg {
		path, err := m.convertFn(work)
		return convertDoneMsg{index: index, path: path, err: err}
	}
}

func (m Model) selectedWorks() []*catalog.Work {
	if m.selectedAuthor == nil {
		return nil
	}
	var result []*catalog.Work
	for i, w := range m.selectedAuthor.Works {
		if m.workSelected[i] {
			result = append(result, w)
		}
	}
	return result
}

func (m Model) View() string {
	switch m.screen {
	case screenAuthor:
		return m.viewAuthor()
	case screenWork:
		return m.viewWork()
	case screenProgress:
		return m.viewProgress()
	case screenMail:
		return m.viewMail()
	}
	return ""
}

func (m Model) viewAuthor() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("mkepub — 作家選択") + "\n\n")
	sb.WriteString(m.input.View() + "\n\n")

	maxRows := m.pageSize()
	start := m.authorOffset
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	maxNameWidth := 0
	for i := start; i < end; i++ {
		if w := runewidth.StringWidth(m.filtered[i].Name()); w > maxNameWidth {
			maxNameWidth = w
		}
	}

	for i := start; i < end; i++ {
		a := m.filtered[i]
		line := fmt.Sprintf("%s (%3d作品)", padRight(a.Name(), maxNameWidth), len(a.Works))
		if i == m.authorCursor {
			sb.WriteString(styleSelect.Render("❯ "+line) + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styleDim.Render(fmt.Sprintf("全 %d 作家", len(m.filtered))))
	sb.WriteString("\n")
	hint := "↑↓ 移動   ←→ ページ移動   Enter 選択   q/Esc 終了"
	if m.mailFn != nil {
		hint += "   m 送信一覧"
	}
	sb.WriteString(styleHelp.Render(hint))

	return sb.String()
}

func (m Model) viewWork() string {
	if m.selectedAuthor == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(styleTitle.Render(m.selectedAuthor.Name()) + "\n\n")

	works := m.selectedAuthor.Works
	start := m.workOffset
	end := start + m.pageSize()
	if end > len(works) {
		end = len(works)
	}

	for i := start; i < end; i++ {
		w := works[i]
		check := "[ ]"
		if m.workSelected[i] {
			check = styleOK.Render("[x]")
		}
		converted := ""
		if m.epubExists(w) {
			converted = styleDim.Render(" ✓")
		}
		line := fmt.Sprintf("%s %s  %4s  %s%s",
			check,
			truncate(w.DisplayTitle(), 36),
			w.PublishedYear,
			styleDim.Render(w.NDC),
			converted,
		)
		if i == m.workCursor {
			sb.WriteString(styleSelect.Render("❯ ") + line + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	sb.WriteString("\n")
	selectedCount := len(m.selectedWorks())
	sb.WriteString(styleDim.Render(fmt.Sprintf("%d件選択", selectedCount)))
	sb.WriteString("\n")
	sb.WriteString(styleHelp.Render("↑↓ 移動   ←→ ページ移動   Space 選択   Enter 変換   Esc 戻る"))

	return sb.String()
}

func (m Model) viewProgress() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("変換中") + "\n\n")

	for _, item := range m.items {
		if !item.done {
			sb.WriteString(styleSpin.Render("  …  ") + item.title + "\n")
		} else if item.err != nil {
			sb.WriteString(styleErr.Render("  ✗  ") + item.title + "\n")
			sb.WriteString(styleErr.Render("       "+item.err.Error()) + "\n")
		} else {
			sb.WriteString(styleOK.Render("  ✓  ") + item.title + "\n")
		}
	}

	sb.WriteString("\n")
	if m.doneCount >= len(m.items) {
		sb.WriteString(styleHelp.Render("Enter で作家選択に戻る   Ctrl+C 終了"))
	}
	return sb.String()
}

func (m Model) epubExists(w *catalog.Work) bool {
	if m.outputDir == "" {
		return false
	}
	name := catalog.SanitizeFilename(w.AuthorName()) + "_" + catalog.SanitizeFilename(w.DisplayTitle()) + ".epub"
	_, err := os.Stat(filepath.Join(m.outputDir, name))
	return err == nil
}

func (m Model) pageSize() int {
	ps := m.height - 8
	if ps < 5 {
		ps = 5
	}
	return ps
}

func (m Model) viewMail() string {
	var sb strings.Builder

	sending := len(m.mailResults) > 0

	if sending {
		sb.WriteString(styleTitle.Render("送信中") + "\n\n")
		for _, item := range m.mailResults {
			if !item.done {
				sb.WriteString(styleSpin.Render("  …  ") + item.title + "\n")
			} else if item.err != nil {
				sb.WriteString(styleErr.Render("  ✗  ") + item.title + "\n")
				sb.WriteString(styleErr.Render("       "+item.err.Error()) + "\n")
			} else {
				sb.WriteString(styleOK.Render("  ✓  ") + item.title + "\n")
			}
		}
		sb.WriteString("\n")
		if m.doneCount >= len(m.mailResults) {
			sb.WriteString(styleHelp.Render("Esc で戻る"))
		}
		return sb.String()
	}

	sb.WriteString(styleTitle.Render("送信一覧") + "\n\n")

	if len(m.mailItems) == 0 {
		sb.WriteString(styleDim.Render("EPUBファイルが見つかりません") + "\n")
		sb.WriteString("\n" + styleHelp.Render("Esc で戻る"))
		return sb.String()
	}

	maxRows := m.pageSize()
	start := m.mailOffset
	end := start + maxRows
	if end > len(m.mailItems) {
		end = len(m.mailItems)
	}

	for i := start; i < end; i++ {
		name := filepath.Base(m.mailItems[i])
		check := "[ ]"
		if m.mailSelected[i] {
			check = styleOK.Render("[x]")
		}
		line := fmt.Sprintf("%s %s", check, name)
		if i == m.mailCursor {
			sb.WriteString(styleSelect.Render("❯ ") + line + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styleDim.Render(fmt.Sprintf("%d件選択", len(m.selectedMails()))))
	sb.WriteString("\n")
	sb.WriteString(styleHelp.Render("↑↓ 移動   ←→ ページ移動   Space 選択   Enter 送信   Esc 戻る"))
	return sb.String()
}

func clamp(v, min, max int) int {
	if max < min {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// truncate pads or truncates s to exactly max display columns.
// padRight pads s with spaces to exactly width display columns (no truncation).
func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncate(s string, max int) string {
	w := runewidth.StringWidth(s)
	if w <= max {
		return s + strings.Repeat(" ", max-w)
	}
	t := runewidth.Truncate(s, max, "…")
	tw := runewidth.StringWidth(t)
	return t + strings.Repeat(" ", max-tw)
}
