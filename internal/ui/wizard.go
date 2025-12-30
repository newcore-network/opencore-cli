package ui

import (
	"fmt"
	"strings"

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
}

// Styles for the wizard
var (
	wizardTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED")).
				Padding(0, 3).
				MarginBottom(1)

	wizardBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
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

	optionDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)

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
)

// NewWizard creates a new wizard model
func NewWizard(steps []WizardStep) WizardModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 45
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))

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
	}
}

// Init initializes the wizard
func (m WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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

	var b strings.Builder

	// Header
	b.WriteString(wizardTitleStyle.Render(" OpenCore Framework "))
	b.WriteString("\n\n")

	// Steps indicator
	b.WriteString(m.renderSteps())
	b.WriteString("\n")

	// Divider
	b.WriteString(stepInactiveStyle.Render(strings.Repeat("-", 60)))
	b.WriteString("\n\n")

	// Current step content
	b.WriteString(m.renderCurrentStep())
	b.WriteString("\n\n")

	// Progress bar
	b.WriteString(m.renderProgress())
	b.WriteString("\n\n")

	// Help text
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m WizardModel) renderSteps() string {
	var parts []string

	for i, step := range m.steps {
		var numStyle, textStyle lipgloss.Style
		var prefix string

		if i < m.currentStep {
			numStyle = stepNumberCompleted
			textStyle = stepCompletedStyle
			prefix = "*"
		} else if i == m.currentStep {
			numStyle = stepNumberActive
			textStyle = stepActiveStyle
			prefix = fmt.Sprintf("%d", i+1)
		} else {
			numStyle = stepNumberInactive
			textStyle = stepInactiveStyle
			prefix = fmt.Sprintf("%d", i+1)
		}

		stepNum := numStyle.Render(prefix)
		stepText := textStyle.Render(step.Title)
		parts = append(parts, fmt.Sprintf("%s %s", stepNum, stepText))

		// Connector between steps
		if i < len(m.steps)-1 {
			if i < m.currentStep {
				parts = append(parts, stepCompletedStyle.Render(" === "))
			} else if i == m.currentStep {
				parts = append(parts, stepActiveStyle.Render(" --> "))
			} else {
				parts = append(parts, stepInactiveStyle.Render(" --- "))
			}
		}
	}

	return strings.Join(parts, "")
}

func (m WizardModel) renderCurrentStep() string {
	step := m.steps[m.currentStep]

	var content strings.Builder

	// Step title
	content.WriteString(stepTitleStyle.Render(step.Title))
	content.WriteString("\n")

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

	// Error message
	if m.err != nil {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render("! " + m.err.Error()))
		content.WriteString("\n")
	}

	return wizardBoxStyle.Render(content.String())
}

func (m WizardModel) renderSelectOptions(options []WizardOption, multiSelect bool) string {
	var content strings.Builder

	for i, opt := range options {
		isSelected := i == m.selectIndex
		isChecked := m.selectedItems[i]

		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		if multiSelect {
			checkbox := "[ ]"
			if isChecked {
				checkbox = "[x]"
			}
			if isSelected {
				content.WriteString(optionSelectedStyle.Render(cursor + checkbox + " " + opt.Label))
			} else {
				content.WriteString(optionStyle.Render(cursor + checkbox + " " + opt.Label))
			}
		} else {
			radio := "( )"
			if isSelected {
				radio = "(*)"
			}
			if isSelected {
				content.WriteString(optionSelectedStyle.Render(cursor + radio + " " + opt.Label))
			} else {
				content.WriteString(optionStyle.Render(cursor + radio + " " + opt.Label))
			}
		}
		content.WriteString("\n")

		// Description for selected item
		if opt.Desc != "" && isSelected {
			content.WriteString(optionDescStyle.Render("      "+opt.Desc) + "\n")
		}
	}

	return content.String()
}

func (m WizardModel) renderConfirm() string {
	if m.confirmVal {
		return confirmYesActive.Render(" > [Yes]") + "    " + confirmInactive.Render("[No]")
	}
	return confirmInactive.Render("[Yes]") + "    " + confirmNoActive.Render("> [No]")
}

func (m WizardModel) renderProgress() string {
	progress := float64(m.currentStep) / float64(len(m.steps))
	width := 40
	filled := int(progress * float64(width))

	// Progress bar with colors
	filledBar := progressBarFilled.Render(strings.Repeat("#", filled))
	emptyBar := progressBarEmpty.Render(strings.Repeat("-", width-filled))
	percentage := int(progress * 100)

	return fmt.Sprintf("[%s%s] %s", filledBar, emptyBar, progressTextStyle.Render(fmt.Sprintf("%d%%", percentage)))
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
