package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/blang/semver/v4"
	"github.com/minio/selfupdate"
)

const (
	githubOwner = "newcore-network"
	githubRepo  = "opencore-cli"
)

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type UpdateInfo struct {
	LatestVersion string    `json:"latest_version"`
	LastCheck     time.Time `json:"last_check"`
}

// CheckForUpdate checks if a new version is available on GitHub
func CheckForUpdate(currentVersion string, force bool) (*UpdateInfo, error) {
	// Fetch from GitHub
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	info := &UpdateInfo{
		LatestVersion: release.TagName,
		LastCheck:     time.Now(),
	}

	return info, nil
}

// NeedsUpdate compares current version with latest version
func NeedsUpdate(currentVersion, latestVersion string) bool {
	cv, err := semver.ParseTolerant(currentVersion)
	if err != nil {
		return false
	}
	lv, err := semver.ParseTolerant(latestVersion)
	if err != nil {
		return false
	}
	return lv.GT(cv)
}

// Update performs the self-update
func Update(version string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", githubOwner, githubRepo, version)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}

	platform := getPlatform()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == fmt.Sprintf("opencore-%s%s", platform, getExt()) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("could not find binary for platform %s", platform)
	}

	resp, err = http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: %s", resp.Status)
	}

	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		return err
	}

	return nil
}

func getPlatform() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	}
	return fmt.Sprintf("%s-%s", runtime.GOOS, arch)
}

func getExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// IsNPMInstallation checks if the CLI was likely installed via NPM
func IsNPMInstallation() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	// Check if the executable is inside an 'npm' or 'node_modules' directory
	return filepath.Base(filepath.Dir(filepath.Dir(executable))) == "npm" ||
		filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(executable)))) == "node_modules"
}
