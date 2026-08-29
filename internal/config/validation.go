package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

func normalizeAndValidate(config *Config) error {
	config.Name = strings.TrimSpace(config.Name)
	config.OutDir = strings.TrimSpace(config.OutDir)
	config.Destination = strings.TrimSpace(config.Destination)
	config.Core.Path = strings.TrimSpace(config.Core.Path)
	config.Core.ResourceName = strings.TrimSpace(config.Core.ResourceName)
	if strings.ContainsRune(config.OutDir, '\x00') || strings.ContainsRune(config.Destination, '\x00') {
		return fmt.Errorf("config output paths must not contain a NUL byte")
	}

	if err := validateName("config.name", config.Name, true); err != nil {
		return err
	}
	if err := validateSourcePath("config.core.path", config.Core.Path, true); err != nil {
		return err
	}
	if err := validateName("config.core.resourceName", config.Core.ResourceName, true); err != nil {
		return err
	}
	if err := validateBuild("config.build", &config.Build); err != nil {
		return err
	}
	if config.Core.Build != nil {
		if err := validateBuild("config.core.build", config.Core.Build); err != nil {
			return err
		}
	}
	if err := validateEntryPoints("config.core.entryPoints", config.Core.EntryPoints); err != nil {
		return err
	}
	if err := validateOptionalSourcePath("config.core.customCompiler", config.Core.CustomCompiler); err != nil {
		return err
	}
	if err := validateViews("config.core.views", config.Core.Views); err != nil {
		return err
	}
	if err := validateCollection("config.resources", config.Resources.Include, config.Resources.Explicit, config.Resources.Views); err != nil {
		return err
	}
	if config.Standalones != nil {
		if err := validateCollection("config.standalones", config.Standalones.Include, config.Standalones.Explicit, config.Standalones.Views); err != nil {
			return err
		}
	}
	if err := validateAdapter(config.Adapter); err != nil {
		return err
	}
	return validateDev(&config.Dev)
}

func validateCollection(field string, globs []string, resources []ExplicitResource, views *ViewsConfig) error {
	if err := validateGlobs(field+".include", globs); err != nil {
		return err
	}
	if err := validateViews(field+".views", views); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(resources))
	for i := range resources {
		resource := &resources[i]
		prefix := fmt.Sprintf("%s.explicit[%d]", field, i)
		resource.Path = strings.TrimSpace(resource.Path)
		resource.ResourceName = strings.TrimSpace(resource.ResourceName)
		if err := validateSourcePath(prefix+".path", resource.Path, true); err != nil {
			return err
		}
		key := normalizedConfigPath(resource.Path)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%s.path duplicates %q", prefix, resource.Path)
		}
		seen[key] = struct{}{}
		if err := validateName(prefix+".resourceName", resource.ResourceName, false); err != nil {
			return err
		}
		if err := validateEntryPoints(prefix+".entryPoints", resource.EntryPoints); err != nil {
			return err
		}
		if err := validateOptionalSourcePath(prefix+".customCompiler", resource.CustomCompiler); err != nil {
			return err
		}
		if err := validateViews(prefix+".views", resource.Views); err != nil {
			return err
		}
		if resource.Build != nil {
			if err := validateDependencyResolution(prefix+".build.dependencyResolution", resource.Build.DependencyResolution); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateName(field, value string, required bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
	if value == "." || value == ".." || strings.ContainsAny(value, `/\\`) {
		return fmt.Errorf("%s must be a name, not a path", field)
	}
	if strings.ContainsRune(value, '\x00') || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return fmt.Errorf("%s contains invalid control characters", field)
	}
	return nil
}

func validateSourcePath(field, value string, required bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%s contains a NUL byte", field)
	}
	path := filepath.FromSlash(value)
	if filepath.IsAbs(path) {
		return fmt.Errorf("%s must be relative to the project root", field)
	}
	clean := filepath.Clean(path)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s must not escape the project root", field)
	}
	return nil
}

func validateOptionalSourcePath(field, value string) error {
	return validateSourcePath(field, value, false)
}

func validateGlobs(field string, patterns []string) error {
	for i, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return fmt.Errorf("%s[%d] must not be empty", field, i)
		}
		if _, err := filepath.Match(filepath.FromSlash(pattern), ""); err != nil {
			return fmt.Errorf("%s[%d] is invalid: %w", field, i, err)
		}
		if err := validateSourcePath(fmt.Sprintf("%s[%d]", field, i), pattern, true); err != nil {
			return err
		}
	}
	return nil
}

func validateEntryPoints(field string, points *EntryPoints) error {
	if points == nil {
		return nil
	}
	if err := validateOptionalSourcePath(field+".server", points.Server); err != nil {
		return err
	}
	return validateOptionalSourcePath(field+".client", points.Client)
}

func validateViews(field string, views *ViewsConfig) error {
	if views == nil {
		return nil
	}
	for name, value := range map[string]string{"path": views.Path, "entryPoint": views.EntryPoint, "outputDir": views.OutputDir} {
		if err := validateOptionalSourcePath(field+"."+name, value); err != nil {
			return err
		}
	}
	if err := validateGlobs(field+".ignore", views.Ignore); err != nil {
		return err
	}
	for i, name := range views.ForceInclude {
		if err := validateOptionalSourcePath(fmt.Sprintf("%s.forceInclude[%d]", field, i), name); err != nil {
			return err
		}
	}
	return nil
}

func validateBuild(field string, build *BuildConfig) error {
	if build.MaxWorkers < 0 {
		return fmt.Errorf("%s.maxWorkers must not be negative", field)
	}
	return validateDependencyResolution(field+".dependencyResolution", build.DependencyResolution)
}

func validateDependencyResolution(field string, resolution *DependencyResolutionConfig) error {
	if resolution == nil {
		return nil
	}
	resolution.Mode = strings.ToLower(strings.TrimSpace(resolution.Mode))
	resolution.PackageManager = strings.ToLower(strings.TrimSpace(resolution.PackageManager))
	if resolution.Mode != "" && !oneOf(resolution.Mode, "auto", "isolated", "shared-resource", "bundle", "symlink") {
		return fmt.Errorf("%s.mode must be auto, isolated, shared-resource, bundle, or symlink", field)
	}
	if resolution.PackageManager != "" && !oneOf(resolution.PackageManager, "auto", "npm", "pnpm", "yarn") {
		return fmt.Errorf("%s.packageManager must be auto, npm, pnpm, or yarn", field)
	}
	if resolution.Mode == "shared-resource" {
		if err := validateName(field+".sharedResourceName", strings.TrimSpace(resolution.SharedResourceName), true); err != nil {
			return err
		}
	}
	return nil
}

func validateAdapter(adapter *AdapterConfig) error {
	if adapter == nil {
		return nil
	}
	for side, binding := range map[string]*AdapterBinding{"server": adapter.Server, "client": adapter.Client} {
		if binding == nil {
			continue
		}
		if !binding.Valid {
			return fmt.Errorf("config.adapter.%s is invalid: %s", side, strings.TrimSpace(binding.Message))
		}
		if strings.TrimSpace(binding.Name) == "" {
			return fmt.Errorf("config.adapter.%s.name is required", side)
		}
	}
	return nil
}

func validateDev(dev *DevConfig) error {
	if dev.Bridge.Port < 0 || dev.Bridge.Port > 65535 || dev.Port < 0 || dev.Port > 65535 {
		return fmt.Errorf("config.dev bridge port must be between 1 and 65535 when set")
	}
	mode := strings.ToLower(strings.TrimSpace(dev.Restart.Mode))
	if mode != "" && !oneOf(mode, "auto", "process", "txadmin", "none") {
		return fmt.Errorf("config.dev.restart.mode must be auto, process, txadmin, or none")
	}
	if dev.Process.StopTimeoutMs < 0 {
		return fmt.Errorf("config.dev.process.stopTimeoutMs must not be negative")
	}
	return validateOptionalSourcePath("config.dev.process.cwd", dev.Process.Cwd)
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
