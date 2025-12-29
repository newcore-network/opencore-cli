package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WizardStep represents a single step in the wizard
type WizardStep struct {
	Title       string
	Description string
	Type        StepType
	Options     []WizardOption // For select/multiselect type
	Validate    func(string) error
}

// WizardOption represents an option in a select step
type WizardOption struct {
	Label string
	Value string
	Desc  string
}

// StepType defines the type of input for a step
type StepType int

const (
	StepTypeInput StepType = iota
	StepTypeSelect
	StepTypeConfirm
	StepTypeMultiSelect // New: multiple selection
)

// WizardResult holds the collected values from the wizard
type WizardResult struct {
	Values map[string]interface{}
}

// WizardModel is the BubbleTea model for the wizard
type WizardModel struct {
	steps         []WizardStep
	currentStep   int
	values        map[string]interface{}
	textInput     textinput.Model
	selectIndex   int
	selectedItems map[int]bool // For multi-select
	confirmVal    bool
	err           error
	done          bool
	cancelled     bool
	width         int
	height        int
	spinner       spinner.Model
	transitioning bool
	transitionDir int // 1 = forward, -1 = backward
	frameCount    int
}

// Styles for the wizard
var (
	// Gradient colors for the title
	gradientColors = []string{"#9333EA", "#7C3AED", "#6366F1", "#3B82F6"}

	wizardTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED")).
				Padding(0, 3).
				MarginBottom(1)

	wizardSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA")).
				Italic(true)

	wizardBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 3).
			Width(65)

	wizardBoxActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#A78BFA")).
				Padding(1, 3).
				Width(65)

	stepActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)

	stepInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4B5563"))

	stepCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981"))

	stepNumberActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED")).
				Bold(true).
				Padding(0, 1)

	stepNumberInactive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Background(lipgloss.Color("#1F2937")).
				Padding(0, 1)

	stepNumberCompleted = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#10B981")).
				Bold(true).
				Padding(0, 1)

	stepTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6")).
			Bold(true).
			MarginBottom(1)

	descriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				Italic(true)

	optionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	optionSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA")).
				Bold(true)

	optionHoverStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#374151")).
				Padding(0, 1)

	optionDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			MarginLeft(4).
			Italic(true)

	checkboxChecked = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	checkboxUnchecked = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4B5563"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			MarginTop(1)

	progressBarFilled = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA"))

	progressBarEmpty = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1F2937"))

	progressTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	confirmYesActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#10B981")).
				Bold(true).
				Padding(0, 2)

	confirmNoActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#EF4444")).
			Bold(true).
			Padding(0, 2)

	confirmInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 2)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))
)

// NewWizard creates a new wizard model
func NewWizard(steps []WizardStep) WizardModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 45
	ti.Prompt = "› "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))

	return WizardModel{
		steps:         steps,
		currentStep:   0,
		values:        make(map[string]interface{}),
		textInput:     ti,
		selectIndex:   0,
		selectedItems: make(map[int]bool),
		confirmVal:    true,
		width:         80,
		height:        24,
		spinner:       s,
	}
}

// tick message for animations
type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(50_000_000, func(_ time.Time) tea.Msg { // 50ms
		return tickMsg{}
	})
}

// Init initializes the wizard
func (m WizardModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick, tickCmd())
}

// Update handles messages
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.frameCount++
		if m.transitioning {
			m.transitioning = false
		}
		return m, tickCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "up":
			if m.steps[m.currentStep].Type == StepTypeSelect ||
				m.steps[m.currentStep].Type == StepTypeMultiSelect {
				if m.selectIndex > 0 {
					m.selectIndex--
				}
			}
			return m, nil

		case "down":
			if m.steps[m.currentStep].Type == StepTypeSelect ||
				m.steps[m.currentStep].Type == StepTypeMultiSelect {
				if m.selectIndex < len(m.steps[m.currentStep].Options)-1 {
					m.selectIndex++
				}
			}
			return m, nil

		case "left":
			if m.steps[m.currentStep].Type == StepTypeConfirm {
				m.confirmVal = true
			}
			return m, nil

		case "right":
			if m.steps[m.currentStep].Type == StepTypeConfirm {
				m.confirmVal = false
			}
			return m, nil

		case " ": // Space to toggle in multi-select
			if m.steps[m.currentStep].Type == StepTypeMultiSelect {
				m.selectedItems[m.selectIndex] = !m.selectedItems[m.selectIndex]
				return m, nil
			}
			if m.steps[m.currentStep].Type == StepTypeConfirm {
				m.confirmVal = !m.confirmVal
				return m, nil
			}

		case "tab":
			if m.steps[m.currentStep].Type == StepTypeConfirm {
				m.confirmVal = !m.confirmVal
				return m, nil
			}
			// Tab in multi-select toggles current and moves down
			if m.steps[m.currentStep].Type == StepTypeMultiSelect {
				m.selectedItems[m.selectIndex] = !m.selectedItems[m.selectIndex]
				if m.selectIndex < len(m.steps[m.currentStep].Options)-1 {
					m.selectIndex++
				}
				return m, nil
			}

		case "backspace":
			if m.steps[m.currentStep].Type == StepTypeInput {
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
			// Go back to previous step if input is empty
			if m.currentStep > 0 && m.textInput.Value() == "" {
				m.currentStep--
				m.err = nil
				m.transitioning = true
				m.transitionDir = -1
				m.loadStepValue()
			}
			return m, nil
		}

		// Handle text input for input type
		if m.steps[m.currentStep].Type == StepTypeInput {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *WizardModel) handleEnter() (tea.Model, tea.Cmd) {
	step := m.steps[m.currentStep]

	// Get value based on step type
	var value interface{}
	switch step.Type {
	case StepTypeInput:
		value = m.textInput.Value()
		// Validate
		if step.Validate != nil {
			if err := step.Validate(value.(string)); err != nil {
				m.err = err
				return m, nil
			}
		}
	case StepTypeSelect:
		value = step.Options[m.selectIndex].Value
	case StepTypeConfirm:
		value = m.confirmVal
	case StepTypeMultiSelect:
		// Collect all selected values
		var selected []string
		for i, opt := range step.Options {
			if m.selectedItems[i] {
				selected = append(selected, opt.Value)
			}
		}
		value = selected
	}

	// Save value
	m.values[step.Title] = value
	m.err = nil

	// Move to next step or finish
	if m.currentStep < len(m.steps)-1 {
		m.currentStep++
		m.selectIndex = 0
		m.confirmVal = true
		m.selectedItems = make(map[int]bool)
		m.textInput.SetValue("")
		m.transitioning = true
		m.transitionDir = 1
		m.loadStepValue()
	} else {
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *WizardModel) loadStepValue() {
	step := m.steps[m.currentStep]
	if val, ok := m.values[step.Title]; ok {
		switch step.Type {
		case StepTypeInput:
			m.textInput.SetValue(val.(string))
		case StepTypeSelect:
			for i, opt := range step.Options {
				if opt.Value == val.(string) {
					m.selectIndex = i
					break
				}
			}
		case StepTypeConfirm:
			m.confirmVal = val.(bool)
		case StepTypeMultiSelect:
			if selected, ok := val.([]string); ok {
				m.selectedItems = make(map[int]bool)
				for _, sel := range selected {
					for i, opt := range step.Options {
						if opt.Value == sel {
							m.selectedItems[i] = true
							break
						}
					}
				}
			}
		}
	}
}

// View renders the wizard
func (m WizardModel) View() string {
	if m.done || m.cancelled {
		return ""
	}

	var content strings.Builder

	// Header with animated gradient effect
	content.WriteString(m.renderHeader())
	content.WriteString("\n\n")

	// Steps indicator with animation
	content.WriteString(m.renderSteps())
	content.WriteString("\n\n")

	// Divider (fixed width of 65)
	content.WriteString(m.renderDivider())
	content.WriteString("\n\n")

	// Current step content (box with fixed width of 65)
	content.WriteString(m.renderCurrentStep())
	content.WriteString("\n\n")

	// Progress bar with glow effect
	content.WriteString(m.renderProgress())
	content.WriteString("\n\n")

	// Status line
	content.WriteString(m.renderStatus())
	content.WriteString("\n")

	// Help text
	content.WriteString(m.renderHelp())

	// Center the entire content block
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content.String(),
	)
}

func (m WizardModel) renderHeader() string {
	// Animated title with subtle color shift
	colorIndex := (m.frameCount / 10) % len(gradientColors)
	titleColor := gradientColors[colorIndex]

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(titleColor)).
		Padding(0, 3).
		Render(" OpenCore Framework ")

	return fmt.Sprint(title)
}

func (m WizardModel) renderSteps() string {
	var parts []string

	for i, step := range m.steps {
		var numStyle, textStyle lipgloss.Style
		var prefix string
		var connector string

		if i < m.currentStep {
			// Completed
			numStyle = stepNumberCompleted
			textStyle = stepCompletedStyle
			prefix = "✓"
		} else if i == m.currentStep {
			// Active - with spinner effect
			numStyle = stepNumberActive
			textStyle = stepActiveStyle
			prefix = fmt.Sprintf("%d", i+1)
		} else {
			// Inactive
			numStyle = stepNumberInactive
			textStyle = stepInactiveStyle
			prefix = fmt.Sprintf("%d", i+1)
		}

		stepNum := numStyle.Render(prefix)
		stepText := textStyle.Render(step.Title)
		parts = append(parts, fmt.Sprintf("%s %s", stepNum, stepText))

		// Connector
		if i < len(m.steps)-1 {
			if i < m.currentStep {
				connector = stepCompletedStyle.Render(" ━━ ")
			} else if i == m.currentStep {
				// Animated connector
				chars := []string{"━", "─", "╌", "─"}
				char := chars[(m.frameCount/5)%len(chars)]
				connector = stepActiveStyle.Render(" " + char + char + " ")
			} else {
				connector = stepInactiveStyle.Render(" ── ")
			}
			parts = append(parts, connector)
		}
	}

	return strings.Join(parts, "")
}

func (m WizardModel) renderDivider() string {
	// Animated divider
	width := 65
	pattern := "─"
	accent := "◆"

	// Calculate accent position based on current step
	progress := float64(m.currentStep) / float64(len(m.steps))
	accentPos := int(progress * float64(width-1))

	var divider strings.Builder
	for i := 0; i < width; i++ {
		if i == accentPos {
			divider.WriteString(stepActiveStyle.Render(accent))
		} else {
			divider.WriteString(dividerStyle.Render(pattern))
		}
	}

	return divider.String()
}

func (m WizardModel) renderCurrentStep() string {
	step := m.steps[m.currentStep]

	var content strings.Builder

	// Step title with icon
	icon := m.getStepIcon(step.Type)
	content.WriteString(fmt.Sprintf("%s %s\n", icon, stepTitleStyle.Render(step.Title)))

	// Description
	if step.Description != "" {
		content.WriteString(descriptionStyle.Render(step.Description))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Input based on type
	switch step.Type {
	case StepTypeInput:
		content.WriteString(m.textInput.View())
		content.WriteString("\n")

	case StepTypeSelect:
		content.WriteString(m.renderSelectOptions(step.Options, false))

	case StepTypeMultiSelect:
		content.WriteString(m.renderSelectOptions(step.Options, true))

	case StepTypeConfirm:
		content.WriteString(m.renderConfirm())
	}

	// Error message with animation
	if m.err != nil {
		content.WriteString("\n")
		errorIcon := "✗"
		if m.frameCount%20 < 10 {
			errorIcon = "!"
		}
		content.WriteString(errorStyle.Render(fmt.Sprintf("%s %s", errorIcon, m.err.Error())))
		content.WriteString("\n")
	}

	// Use active style when transitioning
	boxStyle := wizardBoxStyle
	if m.transitioning {
		boxStyle = wizardBoxActiveStyle
	}

	return boxStyle.Render(content.String())
}

func (m WizardModel) getStepIcon(stepType StepType) string {
	icons := map[StepType]string{
		StepTypeInput:       "✎",
		StepTypeSelect:      "◉",
		StepTypeConfirm:     "?",
		StepTypeMultiSelect: "☰",
	}
	return stepActiveStyle.Render(icons[stepType])
}

func (m WizardModel) renderSelectOptions(options []WizardOption, multiSelect bool) string {
	var content strings.Builder

	for i, opt := range options {
		var line string
		isSelected := i == m.selectIndex
		isChecked := m.selectedItems[i]

		// Cursor/checkbox
		if multiSelect {
			// Checkbox style
			var checkbox string
			if isChecked {
				checkbox = checkboxChecked.Render("[✓]")
			} else {
				checkbox = checkboxUnchecked.Render("[ ]")
			}

			if isSelected {
				line = fmt.Sprintf(" %s %s %s",
					stepActiveStyle.Render("▸"),
					checkbox,
					optionSelectedStyle.Render(opt.Label))
			} else {
				line = fmt.Sprintf("   %s %s", checkbox, optionStyle.Render(opt.Label))
			}
		} else {
			// Radio style
			if isSelected {
				bullet := "●"
				// Animated bullet
				if m.frameCount%20 < 10 {
					bullet = "◉"
				}
				line = fmt.Sprintf(" %s %s",
					stepActiveStyle.Render(bullet),
					optionSelectedStyle.Render(opt.Label))
			} else {
				line = fmt.Sprintf("   %s", optionStyle.Render(opt.Label))
			}
		}

		content.WriteString(line + "\n")

		// Description for selected item
		if opt.Desc != "" && isSelected {
			content.WriteString(optionDescStyle.Render("  ↳ "+opt.Desc) + "\n")
		}
	}

	return content.String()
}

func (m WizardModel) renderConfirm() string {
	var yes, no string

	if m.confirmVal {
		yes = confirmYesActive.Render(" ✓ Yes ")
		no = confirmInactive.Render("   No  ")
	} else {
		yes = confirmInactive.Render("  Yes  ")
		no = confirmNoActive.Render(" ✗ No  ")
	}

	// Add animated indicator
	indicator := "◀"
	if m.frameCount%20 < 10 {
		indicator = "◁"
	}

	if m.confirmVal {
		return fmt.Sprintf("%s %s    %s", stepActiveStyle.Render(indicator), yes, no)
	}
	return fmt.Sprintf("%s    %s %s", yes, no, stepActiveStyle.Render(indicator))
}

func (m WizardModel) renderProgress() string {
	progress := float64(m.currentStep) / float64(len(m.steps))
	width := 50

	filled := int(progress * float64(width))

	// Animated progress bar with glow effect
	var bar strings.Builder

	// Filled part with gradient effect
	for i := 0; i < filled; i++ {
		intensity := float64(i) / float64(width)
		if intensity > 0.7 {
			bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Render("━"))
		} else if intensity > 0.4 {
			bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Render("━"))
		} else {
			bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Render("━"))
		}
	}

	// Animated head
	if filled < width && filled > 0 {
		heads := []string{"╸", "━", "╺", "━"}
		head := heads[(m.frameCount/3)%len(heads)]
		bar.WriteString(stepActiveStyle.Render(head))
		filled++
	}

	// Empty part
	remaining := width - filled
	if remaining > 0 {
		bar.WriteString(progressBarEmpty.Render(strings.Repeat("─", remaining)))
	}

	// Percentage with animation
	percentage := int(progress * 100)
	percentStr := fmt.Sprintf(" %d%%", percentage)

	return bar.String() + progressTextStyle.Render(percentStr)
}

func (m WizardModel) renderStatus() string {
	step := m.steps[m.currentStep]

	// Status messages based on step
	var status string
	switch step.Type {
	case StepTypeInput:
		if m.textInput.Value() == "" {
			status = "Waiting for input..."
		} else {
			status = fmt.Sprintf("Ready: \"%s\"", m.textInput.Value())
		}
	case StepTypeSelect:
		opt := step.Options[m.selectIndex]
		status = fmt.Sprintf("Selected: %s", opt.Label)
	case StepTypeMultiSelect:
		count := 0
		for _, v := range m.selectedItems {
			if v {
				count++
			}
		}
		if count == 0 {
			status = "No modules selected (optional)"
		} else {
			status = fmt.Sprintf("%d module(s) selected", count)
		}
	case StepTypeConfirm:
		if m.confirmVal {
			status = "Option: Yes"
		} else {
			status = "Option: No"
		}
	}

	// Spinner for active state
	spinnerView := m.spinner.View()

	return fmt.Sprintf("%s %s", spinnerView, progressTextStyle.Render(status))
}

func (m WizardModel) renderHelp() string {
	step := m.steps[m.currentStep]

	var help string
	switch step.Type {
	case StepTypeInput:
		help = "enter: confirm • backspace: clear/back • esc: cancel"
	case StepTypeSelect:
		help = "↑/↓: navigate • enter: select • esc: cancel"
	case StepTypeMultiSelect:
		help = "↑/↓: navigate • space: toggle • enter: confirm • esc: cancel"
	case StepTypeConfirm:
		help = "←/→ or space: toggle • enter: confirm • esc: cancel"
	}

	return helpStyle.Render(help)
}

// GetValues returns the collected values
func (m WizardModel) GetValues() map[string]interface{} {
	return m.values
}

// IsCancelled returns whether the wizard was cancelled
func (m WizardModel) IsCancelled() bool {
	return m.cancelled
}

// IsDone returns whether the wizard completed successfully
func (m WizardModel) IsDone() bool {
	return m.done
}

// GetStringValue gets a string value by key
func (m WizardModel) GetStringValue(key string) string {
	if val, ok := m.values[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// GetBoolValue gets a bool value by key
func (m WizardModel) GetBoolValue(key string) bool {
	if val, ok := m.values[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// GetStringSliceValue gets a string slice value by key
func (m WizardModel) GetStringSliceValue(key string) []string {
	if val, ok := m.values[key]; ok {
		if s, ok := val.([]string); ok {
			return s
		}
	}
	return nil
}
