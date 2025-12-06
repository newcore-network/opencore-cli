package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

var officialTemplates = map[string]string{
	"chat":   "https://github.com/newcore-network/opencore-template-chat",
	"admin":  "https://github.com/newcore-network/opencore-template-admin",
	"racing": "https://github.com/newcore-network/opencore-template-racing",
}

func NewCloneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone [template]",
		Short: "Clone an official template",
		Long:  "Download and set up an official OpenCore template from GitHub.",
		Args:  cobra.ExactArgs(1),
		RunE:  runClone,
	}

	return cmd
}

type cloneModel struct {
	spinner  spinner.Model
	template string
	url      string
	done     bool
	err      error
}

func (m cloneModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.clone(),
	)
}

func (m cloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case cloneResultMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m cloneModel) View() string {
	if m.done {
		if m.err != nil {
			return ui.Error(fmt.Sprintf("Failed to clone template: %v", m.err)) + "\n"
		}
		return ui.Success(fmt.Sprintf("Template '%s' cloned successfully!", m.template)) + "\n\n" +
			ui.BoxStyle.Render(fmt.Sprintf("Next steps:\n  cd resources/%s\n  pnpm install", m.template))
	}

	return fmt.Sprintf("%s Cloning template %s...\n", m.spinner.View(), m.template)
}

type cloneResultMsg struct {
	err error
}

func (m cloneModel) clone() tea.Cmd {
	return func() tea.Msg {
		targetPath := filepath.Join("resources", m.template)

		// Check if directory already exists
		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			return cloneResultMsg{err: fmt.Errorf("directory '%s' already exists", targetPath)}
		}

		// Clone repository
		cmd := exec.Command("git", "clone", m.url, targetPath)
		if err := cmd.Run(); err != nil {
			return cloneResultMsg{err: err}
		}

		// Remove .git directory
		gitDir := filepath.Join(targetPath, ".git")
		os.RemoveAll(gitDir)

		return cloneResultMsg{err: nil}
	}
}

func runClone(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Clone Template"))
	fmt.Println()

	templateName := args[0]

	// Check if template exists
	templateURL, exists := officialTemplates[templateName]
	if !exists {
		fmt.Println(ui.Error(fmt.Sprintf("Unknown template: %s", templateName)))
		fmt.Println()
		fmt.Println("Available templates:")
		for name := range officialTemplates {
			fmt.Printf("  • %s\n", name)
		}
		return fmt.Errorf("template not found")
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	m := cloneModel{
		spinner:  s,
		template: templateName,
		url:      templateURL,
		done:     false,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if finalModel.(cloneModel).err != nil {
		return finalModel.(cloneModel).err
	}

	return nil
}
