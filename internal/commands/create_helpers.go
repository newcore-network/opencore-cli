package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

type createNamePrompt struct {
	Title       string
	Description string
	Kind        string
}

func validateCreateName(kind string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s name cannot be empty", kind)
		}
		if strings.Contains(s, " ") {
			return fmt.Errorf("%s name cannot contain spaces", kind)
		}
		return nil
	}
}

func getNameFromArgsOrPrompt(args []string, p createNamePrompt) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	var name string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(p.Title).
				Description(p.Description).
				Value(&name).
				Validate(validateCreateName(p.Kind)),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}

	return name, nil
}

func featuresMessage(withClient, withNUI bool) string {
	msg := "Features:\n  • Server-side code"
	if withClient {
		msg += "\n  • Client-side code"
	}
	if withNUI {
		msg += "\n  • NUI (UI)"
	}
	return msg
}

func renderCreateBox(content string) {
	fmt.Println(ui.BoxStyle.Render(content))
	fmt.Println()
}
