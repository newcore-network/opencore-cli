package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/newcore-network/opencore-cli/internal/templates"
)

const ocManifestFileName = "oc.manifest.json"

type manifestValidationError struct {
	message string
}

func (e *manifestValidationError) Error() string {
	return e.message
}

type templateCategory string

const (
	templateCategoryResource   templateCategory = "resource"
	templateCategoryStandalone templateCategory = "standalone"
)

type templateManifest struct {
	Schema        string                         `json:"$schema,omitempty"`
	Version       int                            `json:"schemaVersion"`
	Name          string                         `json:"name"`
	DisplayName   string                         `json:"displayName,omitempty"`
	Kind          string                         `json:"kind"`
	Description   string                         `json:"description,omitempty"`
	Compatibility *templateManifestCompatibility `json:"compatibility,omitempty"`
	Requires      *templateManifestRequires      `json:"requires,omitempty"`
	Links         *templateManifestLinks         `json:"links,omitempty"`
}

type templateManifestCompatibility struct {
	Runtimes     []string `json:"runtimes,omitempty"`
	GameProfiles []string `json:"gameProfiles,omitempty"`
}

type templateManifestRequires struct {
	Templates []string `json:"templates,omitempty"`
}

type templateManifestLinks struct {
	Readme     string `json:"readme,omitempty"`
	Docs       string `json:"docs,omitempty"`
	Repository string `json:"repository,omitempty"`
}

type templateDescriptor struct {
	Name          string
	SourcePath    string
	TargetPath    string
	Category      templateCategory
	Manifest      *templateManifest
	ManifestError error
}

func parseTemplateManifest(data []byte) (*templateManifest, error) {
	manifest := &templateManifest{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(manifest); err != nil {
		return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: %v", ocManifestFileName, err)}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: trailing data: %v", ocManifestFileName, err)}
	}

	if manifest.Version != 1 {
		return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: schemaVersion must be 1", ocManifestFileName)}
	}

	if err := templates.ValidateName(manifest.Name); err != nil {
		return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: invalid name: %v", ocManifestFileName, err)}
	}
	if manifest.DisplayName == "" && manifestFieldPresent(data, "displayName") {
		return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: displayName cannot be empty", ocManifestFileName)}
	}

	switch manifest.Kind {
	case string(templateCategoryResource), string(templateCategoryStandalone), "core":
	default:
		return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: kind must be 'resource', 'standalone', or 'core'", ocManifestFileName)}
	}

	if manifest.Compatibility != nil {
		seenRuntimes := make(map[string]struct{}, len(manifest.Compatibility.Runtimes))
		for _, runtime := range manifest.Compatibility.Runtimes {
			if !isSupportedManifestRuntime(runtime) {
				return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: unsupported runtime '%s'", ocManifestFileName, runtime)}
			}
			if _, exists := seenRuntimes[runtime]; exists {
				return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: duplicate runtime '%s'", ocManifestFileName, runtime)}
			}
			seenRuntimes[runtime] = struct{}{}
		}
		seenProfiles := make(map[string]struct{}, len(manifest.Compatibility.GameProfiles))
		for _, profile := range manifest.Compatibility.GameProfiles {
			if !isSupportedManifestGameProfile(profile) {
				return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: unsupported gameProfile '%s'", ocManifestFileName, profile)}
			}
			if _, exists := seenProfiles[profile]; exists {
				return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: duplicate gameProfile '%s'", ocManifestFileName, profile)}
			}
			seenProfiles[profile] = struct{}{}
		}
	}

	if manifest.Requires != nil {
		seenDependencies := make(map[string]struct{}, len(manifest.Requires.Templates))
		for _, dependency := range manifest.Requires.Templates {
			if err := templates.ValidateName(dependency); err != nil {
				return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: invalid required template '%s': %v", ocManifestFileName, dependency, err)}
			}
			if _, exists := seenDependencies[dependency]; exists {
				return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: duplicate required template '%s'", ocManifestFileName, dependency)}
			}
			seenDependencies[dependency] = struct{}{}
		}
	}
	if manifest.Links != nil {
		for field, value := range map[string]string{
			"readme":     manifest.Links.Readme,
			"docs":       manifest.Links.Docs,
			"repository": manifest.Links.Repository,
		} {
			if value == "" && manifestNestedFieldPresent(data, "links", field) {
				return nil, &manifestValidationError{message: fmt.Sprintf("invalid %s: links.%s cannot be empty", ocManifestFileName, field)}
			}
		}
	}

	return manifest, nil
}

func manifestFieldPresent(data []byte, field string) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	_, ok := raw[field]
	return ok
}

func manifestNestedFieldPresent(data []byte, parent, field string) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	var nested map[string]json.RawMessage
	if json.Unmarshal(raw[parent], &nested) != nil {
		return false
	}
	_, ok := nested[field]
	return ok
}

func validateManifestCategory(manifest *templateManifest, category templateCategory) error {
	if manifest == nil {
		return nil
	}

	if manifest.Kind != string(category) {
		return &manifestValidationError{
			message: fmt.Sprintf(
				"invalid %s: kind '%s' does not match template category '%s'",
				ocManifestFileName,
				manifest.Kind,
				category,
			),
		}
	}

	return nil
}

func isSupportedManifestRuntime(runtime string) bool {
	switch runtime {
	case "fivem", "redm", "ragemp":
		return true
	default:
		return false
	}
}

func isSupportedManifestGameProfile(profile string) bool {
	switch profile {
	case "common", "gta5", "rdr3":
		return true
	default:
		return false
	}
}

func (m *templateManifest) effectiveName(fallback string) string {
	if m == nil {
		return fallback
	}
	if strings.TrimSpace(m.DisplayName) != "" {
		return m.DisplayName
	}
	if strings.TrimSpace(m.Name) != "" {
		return m.Name
	}
	return fallback
}

func compatibilityStatusLabel(descriptor templateDescriptor) string {
	if descriptor.ManifestError != nil {
		if _, ok := descriptor.ManifestError.(*manifestValidationError); ok {
			return "compat: invalid manifest"
		}
		return "compat: unavailable"
	}
	if descriptor.Manifest == nil || descriptor.Manifest.Compatibility == nil || len(descriptor.Manifest.Compatibility.Runtimes) == 0 {
		return "compat: unknown"
	}

	runtimes := append([]string(nil), descriptor.Manifest.Compatibility.Runtimes...)
	sort.Strings(runtimes)
	return fmt.Sprintf("compat: %s", strings.Join(runtimes, ", "))
}

func validateManifestCompatibility(descriptor templateDescriptor, runtime string) error {
	if descriptor.ManifestError != nil || descriptor.Manifest == nil || descriptor.Manifest.Compatibility == nil {
		return nil
	}

	runtimes := descriptor.Manifest.Compatibility.Runtimes
	if len(runtimes) == 0 {
		return nil
	}

	for _, supportedRuntime := range runtimes {
		if supportedRuntime == runtime {
			return nil
		}
	}

	supported := append([]string(nil), runtimes...)
	sort.Strings(supported)
	return fmt.Errorf(
		"template '%s' is not compatible with runtime '%s'\n\nSupported runtimes: %s\nUse '--force' to clone it anyway",
		descriptor.Name,
		runtime,
		strings.Join(supported, ", "),
	)
}
