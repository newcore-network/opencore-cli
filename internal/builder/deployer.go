package builder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/newcore-network/opencore-cli/internal/config"
)

// Deployer handles copying built resources to the destination
type Deployer struct {
	config *config.Config
}

// NewDeployer creates a new deployer
func NewDeployer(cfg *config.Config) *Deployer {
	return &Deployer{config: cfg}
}

// Deploy copies all built resources to the destination
func (d *Deployer) Deploy() error {
	if d.config.Destination == "" {
		return nil // No destination configured, skip deploy
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(d.config.Destination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy from outDir to destination
	srcDir := d.config.OutDir
	dstDir := d.config.Destination

	return d.copyDir(srcDir, dstDir)
}

// DeployResource copies a single resource to the destination
func (d *Deployer) DeployResource(resourceName string) error {
	if d.config.Destination == "" {
		return nil
	}

	srcPath := filepath.Join(d.config.OutDir, resourceName)
	dstPath := filepath.Join(d.config.Destination, resourceName)

	// Remove existing destination if present
	if err := os.RemoveAll(dstPath); err != nil {
		return fmt.Errorf("failed to clean destination: %w", err)
	}

	return d.copyDir(srcPath, dstPath)
}

// copyDir recursively copies a directory
func (d *Deployer) copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source directory not found: %w", err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := d.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := d.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func (d *Deployer) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// GetDeployedPath returns the full path where a resource will be deployed
func (d *Deployer) GetDeployedPath(resourceName string) string {
	if d.config.Destination == "" {
		return filepath.Join(d.config.OutDir, resourceName)
	}
	return filepath.Join(d.config.Destination, resourceName)
}

// HasDestination returns whether a destination is configured
func (d *Deployer) HasDestination() bool {
	return d.config.Destination != ""
}
