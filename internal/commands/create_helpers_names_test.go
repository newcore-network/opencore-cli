package commands

import "testing"

func TestValidateCreateName(t *testing.T) {
	validate := validateCreateName("resource")

	valid := []string{"my-resource", "my_resource", "Resource123", "a"}
	for _, name := range valid {
		if err := validate(name); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", name, err)
		}
	}

	invalid := []string{"", "has space", "../../etc/passwd", "../escape", "nested/path", "a/b", "."}
	for _, name := range invalid {
		if err := validate(name); err == nil {
			t.Errorf("expected %q to be rejected, got no error", name)
		}
	}
}

func TestGetNameFromArgsOrPromptRejectsTraversalInArgs(t *testing.T) {
	prompt := createNamePrompt{Title: "Name", Description: "desc", Kind: "resource"}

	if _, err := getNameFromArgsOrPrompt([]string{"../../etc/passwd"}, prompt); err == nil {
		t.Fatal("expected an error for a traversal name passed as a CLI argument, got nil")
	}
}

func TestGetNameFromArgsOrPromptAcceptsValidArgs(t *testing.T) {
	prompt := createNamePrompt{Title: "Name", Description: "desc", Kind: "resource"}

	name, err := getNameFromArgsOrPrompt([]string{"my-resource"}, prompt)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if name != "my-resource" {
		t.Fatalf("expected name %q, got %q", "my-resource", name)
	}
}
