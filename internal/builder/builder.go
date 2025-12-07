package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

type Builder struct {
	config *config.Config
}

func New(cfg *config.Config) *Builder {
	return &Builder{config: cfg}
}

type buildMsg struct {
	resource string
	success  bool
	duration time.Duration
	err      error
}

type buildModel struct {
	spinner   spinner.Model
	results   []buildMsg
	done      bool
	resources []string
	current   int
	outDir    string
}

func (m buildModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.buildNext(),
	)
}

func (m buildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case buildMsg:
		m.results = append(m.results, msg)
		m.current++

		if m.current >= len(m.resources) {
			m.done = true
			return m, tea.Quit
		}

		return m, m.buildNext()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m buildModel) View() string {
	if m.done {
		return m.renderResults()
	}

	s := ui.TitleStyle.Render("Building Resources") + "\n\n"

	// Show completed
	for _, result := range m.results {
		if result.success {
			s += ui.Success(fmt.Sprintf("[%s] compiled (%s)", result.resource, result.duration.Round(time.Millisecond)))
		} else {
			s += ui.Error(fmt.Sprintf("[%s] failed: %v", result.resource, result.err))
		}
		s += "\n"
	}

	// Show current
	if m.current < len(m.resources) {
		s += fmt.Sprintf("%s Building %s...\n", m.spinner.View(), m.resources[m.current])
	}

	return s
}

func (m buildModel) renderResults() string {
	successCount := 0
	failCount := 0
	totalDuration := time.Duration(0)

	for _, result := range m.results {
		if result.success {
			successCount++
			totalDuration += result.duration
		} else {
			failCount++
		}
	}

	s := "\n"
	for _, result := range m.results {
		if result.success {
			s += ui.Success(fmt.Sprintf("[%s] compiled (%s)", result.resource, result.duration.Round(time.Millisecond)))
		} else {
			s += ui.Error(fmt.Sprintf("[%s] failed: %v", result.resource, result.err))
		}
		s += "\n"
	}

	s += "\n"

	if failCount == 0 {
		boxContent := fmt.Sprintf(
			"✓ Build completed successfully!\n\n"+
				"Resources: %d\n"+
				"Time: %s\n"+
				"Output: %s",
			successCount,
			totalDuration.Round(time.Millisecond),
			m.outDir,
		)
		s += ui.SuccessBoxStyle.Render(boxContent)
	} else {
		boxContent := fmt.Sprintf(
			"✗ Build completed with errors\n\n"+
				"Success: %d\n"+
				"Failed: %d",
			successCount,
			failCount,
		)
		s += ui.ErrorBoxStyle.Render(boxContent)
	}

	return s
}

func (m buildModel) buildNext() tea.Cmd {
	return func() tea.Msg {
		resourcePath := m.resources[m.current]
		start := time.Now()

		err := buildResource(resourcePath)
		duration := time.Since(start)

		return buildMsg{
			resource: filepath.Base(resourcePath),
			success:  err == nil,
			duration: duration,
			err:      err,
		}
	}
}

func (b *Builder) Build() error {
	fmt.Println(ui.Logo())

	// Check if scripts/build.js exists
	buildScript := filepath.Join(".", "scripts", "build.js")
	if _, err := os.Stat(buildScript); os.IsNotExist(err) {
		return fmt.Errorf("build script not found: %s", buildScript)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	// Build core using scripts/build.js
	fmt.Printf("%s Building core...\n", s.View())

	start := time.Now()
	cmd := exec.Command("node", "scripts/build.js")
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("core build failed: %w", err)
	}

	duration := time.Since(start)

	// Build resources if any
	resources := b.config.GetResourcePaths()
	// Filter out core from resources (it's already built)
	filteredResources := []string{}
	for _, r := range resources {
		if r != b.config.Core.Path {
			filteredResources = append(filteredResources, r)
		}
	}

	if len(filteredResources) > 0 {
		m := buildModel{
			spinner:   s,
			resources: filteredResources,
			results:   []buildMsg{},
			current:   0,
			done:      false,
			outDir:    b.config.OutDir,
		}

		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			return err
		}
	}

	// Show success
	boxContent := fmt.Sprintf(
		"✓ Build completed successfully!\n\n"+
			"Core: %s\n"+
			"Resources: %d\n"+
			"Output: %s",
		duration.Round(time.Millisecond),
		len(filteredResources),
		b.config.OutDir,
	)
	fmt.Println(ui.SuccessBoxStyle.Render(boxContent))

	return nil
}

func buildResource(resourcePath string) error {
	// Check if package.json exists
	packageJSON := filepath.Join(resourcePath, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("package.json not found in %s", resourcePath)
	}

	// Run pnpm build
	cmd := exec.Command("pnpm", "build")
	cmd.Dir = resourcePath
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}
