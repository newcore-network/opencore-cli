package ui

import "testing"

func TestWizardDisabledSelectOptionCannotBeSubmitted(t *testing.T) {
	wizard := NewWizard([]WizardStep{
		{
			Title: "Adapter",
			Type:  StepTypeSelect,
			Options: []WizardOption{
				{Label: "FiveM", Value: "fivem"},
				{Label: "RedM", Value: "redm", Disabled: true},
			},
		},
	})

	wizard.selectIndex = 1
	model, _ := wizard.handleEnter()
	updated := model.(*WizardModel)

	if updated.err == nil {
		t.Fatal("expected disabled option to produce an error")
	}
	if updated.done {
		t.Fatal("expected wizard to remain active when disabled option is selected")
	}
}
