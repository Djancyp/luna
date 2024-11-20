package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type stage int

const (
	stageProjectName stage = iota
	stageFrontendType
	stageTailwindCSS
	stageSummary
)

type model struct {
	stage        stage
	project      textinput.Model
	FrontendType []string         // List of frontend type options
	TailwindCSS  bool             // Whether to include Tailwind CSS
	cursor       int              // Cursor for navigating choices
	selected     map[int]struct{} // Selected items
}

func initialModel() model {
	projectName := textinput.New()
	projectName.Placeholder = "Enter project directory name"
	projectName.Focus()
	projectName.Width = 30

	return model{
		stage:        stageProjectName,
		project:      projectName,
		FrontendType: []string{"TypeScript", "JavaScript"},
		TailwindCSS:  true,
		selected:     map[int]struct{}{},
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "enter":
			switch m.stage {
			case stageProjectName:
				if m.project.Value() == "" {
					// Exit if no project name is provided
					return m, tea.Quit
				}
				m.stage = stageFrontendType // Move to next stage
			case stageFrontendType:
				// Move to the summary stage
				m.stage = stageSummary
			case stageSummary:
				// Exit after displaying the summary
				fmt.Printf("Project Name: %s\nFrontend Type: %s\n",
					m.project.Value(),
					m.FrontendType[m.cursor],
				)
				return m, tea.Quit
			}
		case "up", "k":
			if m.stage == stageFrontendType && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.stage == stageFrontendType && m.cursor < len(m.FrontendType)-1 {
				m.cursor++
			}
		}

		if m.stage == stageProjectName {
			m.project, cmd = m.project.Update(msg)
		}
	}

	return m, cmd
}

func (m model) View() string {
	switch m.stage {
	case stageProjectName:
		return fmt.Sprintf(
			"Project Name:\n%s\n\nPress Enter to continue, or Esc to quit.",
			m.project.View(),
		)
	case stageFrontendType:
		var b strings.Builder
		b.WriteString("Select Frontend Type:\n\n")
		for i, choice := range m.FrontendType {
			cursor := " " // No cursor by default
			if m.cursor == i {
				cursor = ">" // Show cursor for the current choice
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
		}
		b.WriteString("\nPress Enter to confirm, or Esc to quit.")
		return b.String()
	case stageSummary:
		return fmt.Sprintf(
			"Summary:\nProject Name: %s\nFrontend Type: %s\n\nPress Enter to finish.",
			m.project.Value(),
			m.FrontendType[m.cursor],
		)
	}

	return "Press q to quit.\n"
}
