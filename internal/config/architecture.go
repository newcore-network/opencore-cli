package config

import (
	"os"
	"path/filepath"
)

type Architecture string

const (
	ArchitectureDomainDriven Architecture = "domain-driven"
	ArchitectureLayerBased   Architecture = "layer-based"
	ArchitectureFeatureBased Architecture = "feature-based"
	ArchitectureHybrid       Architecture = "hybrid"
	ArchitectureUnknown      Architecture = "unknown"
)

// DetectArchitecture detects the project architecture based on directory structure
func DetectArchitecture(projectPath string) Architecture {
	coreSrc := filepath.Join(projectPath, "core", "src")

	// Check for domain-driven (modules/ directory)
	if _, err := os.Stat(filepath.Join(coreSrc, "modules")); err == nil {
		return ArchitectureDomainDriven
	}

	// Check for layer-based (client/ and server/ at root with controllers/ subdirs)
	clientControllers := filepath.Join(coreSrc, "client", "controllers")
	serverControllers := filepath.Join(coreSrc, "server", "controllers")
	if _, err := os.Stat(clientControllers); err == nil {
		if _, err := os.Stat(serverControllers); err == nil {
			return ArchitectureLayerBased
		}
	}

	// Check for feature-based (features/ directory)
	if _, err := os.Stat(filepath.Join(coreSrc, "features")); err == nil {
		return ArchitectureFeatureBased
	}

	// Check for hybrid (both core-modules/ and features/)
	coreModules := filepath.Join(coreSrc, "core-modules")
	features := filepath.Join(coreSrc, "features")
	if _, err := os.Stat(coreModules); err == nil {
		if _, err := os.Stat(features); err == nil {
			return ArchitectureHybrid
		}
	}

	return ArchitectureUnknown
}

// GetArchitectureDescription returns a description of the architecture
func GetArchitectureDescription(arch Architecture) string {
	switch arch {
	case ArchitectureDomainDriven:
		return "Domain-Driven: Modules organized by business domain (recommended for large projects)"
	case ArchitectureLayerBased:
		return "Layer-Based: Separation by technical layers (recommended for large teams)"
	case ArchitectureFeatureBased:
		return "Feature-Based: Simple feature organization (recommended for small/medium projects)"
	case ArchitectureHybrid:
		return "Hybrid: Mix of domain modules and simple features (recommended for evolving projects)"
	default:
		return "Unknown architecture"
	}
}

// GetFeatureBasePath returns the base path for creating new features based on architecture
func GetFeatureBasePath(projectPath string, arch Architecture) string {
	coreSrc := filepath.Join(projectPath, "core", "src")

	switch arch {
	case ArchitectureDomainDriven:
		return filepath.Join(coreSrc, "modules")
	case ArchitectureLayerBased:
		return coreSrc // Will create in server/controllers and client/controllers
	case ArchitectureFeatureBased:
		return filepath.Join(coreSrc, "features")
	case ArchitectureHybrid:
		return filepath.Join(coreSrc, "features") // Default to features for hybrid
	default:
		return filepath.Join(coreSrc, "features") // Fallback
	}
}

