package commands

import (
	"strings"
	"testing"
)

func TestParseTemplateManifestValid(t *testing.T) {
	manifest, err := parseTemplateManifest([]byte(`{
		"schemaVersion": 1,
		"name": "xchat",
		"displayName": "xChat",
		"kind": "resource",
		"compatibility": {
			"runtimes": ["fivem", "redm"],
			"gameProfiles": ["gta5", "rdr3"]
		},
		"requires": {
			"templates": ["core"]
		}
	}`))
	if err != nil {
		t.Fatalf("expected manifest to parse, got error: %v", err)
	}

	if manifest.Name != "xchat" {
		t.Fatalf("expected name xchat, got %q", manifest.Name)
	}
	if manifest.Kind != "resource" {
		t.Fatalf("expected kind resource, got %q", manifest.Kind)
	}
	if len(manifest.Compatibility.Runtimes) != 2 {
		t.Fatalf("expected 2 runtimes, got %d", len(manifest.Compatibility.Runtimes))
	}
}

func TestParseTemplateManifestRejectsUnknownFields(t *testing.T) {
	_, err := parseTemplateManifest([]byte(`{
		"schemaVersion": 1,
		"name": "xchat",
		"kind": "resource",
		"unknown": true
	}`))
	if err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestParseTemplateManifestRejectsInvalidRuntime(t *testing.T) {
	_, err := parseTemplateManifest([]byte(`{
		"schemaVersion": 1,
		"name": "xchat",
		"kind": "resource",
		"compatibility": {
			"runtimes": ["altv"]
		}
	}`))
	if err == nil {
		t.Fatal("expected invalid runtime error")
	}
}

func TestParseTemplateManifestRejectsUnsafeNamesAndTrailingJSON(t *testing.T) {
	for _, name := range []string{"Chat", "chat.name", "chat:name", "../chat"} {
		data := []byte(`{"schemaVersion":1,"name":"` + name + `","kind":"resource"}`)
		if _, err := parseTemplateManifest(data); err == nil {
			t.Errorf("expected name %q to be rejected", name)
		}
	}

	if _, err := parseTemplateManifest([]byte(`{"schemaVersion":1,"name":"chat","kind":"resource"} {}`)); err == nil {
		t.Fatal("expected trailing JSON value to be rejected")
	}
}

func TestParseTemplateManifestRejectsDuplicateAndInvalidNestedValues(t *testing.T) {
	cases := []string{
		`{"schemaVersion":1,"name":"chat","kind":"resource","compatibility":{"runtimes":["fivem","fivem"]}}`,
		`{"schemaVersion":1,"name":"chat","kind":"resource","compatibility":{"gameProfiles":["gta5","gta5"]}}`,
		`{"schemaVersion":1,"name":"chat","kind":"resource","requires":{"templates":["../core"]}}`,
		`{"schemaVersion":1,"name":"chat","kind":"resource","requires":{"templates":["core","core"]}}`,
		`{"schemaVersion":1,"name":"chat","displayName":"","kind":"resource"}`,
		`{"schemaVersion":1,"name":"chat","kind":"resource","links":{"readme":""}}`,
	}
	for _, data := range cases {
		if _, err := parseTemplateManifest([]byte(data)); err == nil {
			t.Errorf("expected invalid manifest to be rejected: %s", data)
		}
	}
}

func TestValidateManifestCompatibilityAllowsUnknownManifest(t *testing.T) {
	descriptor := templateDescriptor{Name: "xchat"}
	if err := validateManifestCompatibility(descriptor, "ragemp"); err != nil {
		t.Fatalf("expected unknown manifest to be allowed, got %v", err)
	}
}

func TestValidateManifestCategoryRejectsMismatch(t *testing.T) {
	err := validateManifestCategory(&templateManifest{
		Version: 1,
		Name:    "utility",
		Kind:    "standalone",
	}, templateCategoryResource)
	if err == nil {
		t.Fatal("expected category validation error")
	}
}

func TestValidateManifestCompatibilityRejectsUnsupportedRuntime(t *testing.T) {
	descriptor := templateDescriptor{
		Name: "xchat",
		Manifest: &templateManifest{
			Version: 1,
			Name:    "xchat",
			Kind:    "resource",
			Compatibility: &templateManifestCompatibility{
				Runtimes: []string{"fivem", "redm"},
			},
		},
	}

	err := validateManifestCompatibility(descriptor, "ragemp")
	if err == nil {
		t.Fatal("expected compatibility error")
	}
	if !strings.Contains(err.Error(), "Supported runtimes: fivem, redm") {
		t.Fatalf("unexpected error: %v", err)
	}
}
