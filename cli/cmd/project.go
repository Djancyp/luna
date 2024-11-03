package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type stage int

const (
	stageProjectName stage = iota
	stageFrontendType
	stageSummary
)

type model struct {
	stage    stage
	project  textinput.Model
	choices  []string         // List of options, if needed
	cursor   int              // List cursor position
	selected map[int]struct{} // Selected items
}

func initialModel() model {
	projectName := textinput.New()
	projectName.Placeholder = "Enter project directory name"
	projectName.Focus()
	projectName.Width = 20

	return model{
		stage:   stageProjectName,
		project: projectName,
		choices: []string{
			"TypeScript",
			"JavaScript",
		},
		selected: map[int]struct{}{},
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
		case "q":
			return m, tea.Quit
		case "enter":
			switch m.stage {
			case stageProjectName:
				m.project.Blur()
				if m.project.Value() == "" {
					return m, tea.Quit
				}
			case stageSummary:
				fmt.Println("Summary")
				return m, tea.Quit
			}
		case "esc":
			return m, tea.Quit
		}

		switch m.stage {
		case stageProjectName:
			m.project, cmd = m.project.Update(msg)
		}
	}

	return m, cmd
}

func (m model) View() string {
	switch m.stage {
	case stageProjectName:
		return fmt.Sprintf(
			"Project Name:\n%s\n\nPress Enter to continue",
			m.project.View(),
		)
	case stageSummary:
		return fmt.Sprintf(
			"Summary:\nProject Name: %s\n \n\nPress Enter to finish",
			m.project.Value(),
		)
	}

	return "Press q to quit.\n"
}
