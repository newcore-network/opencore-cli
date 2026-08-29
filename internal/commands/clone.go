package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/pkgmgr"
	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

const (
	templatesRepo = "newcore-network/opencore-templates"
	templatesURL  = "https://github.com/" + templatesRepo
	apiBaseURL    = "https://api.github.com/repos/" + templatesRepo + "/contents"
	apiBodyLimit  = 2 << 20
	manifestLimit = 1 << 20
	fileBodyLimit = 50 << 20
	cloneBodyLimit = 200 << 20
)

var cloneHTTPClient = &http.Client{Timeout: 30 * time.Second}

// GitHubContent represents a file/directory from GitHub API
type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
	URL         string `json:"url"`
}

func NewCloneCommand() *cobra.Command {
	var listTemplates bool
	var useAPI bool
	var force bool
	var branch string

	cmd := &cobra.Command{
		Use:   "clone <template>",
		Short: "Clone an official template",
		Long: fmt.Sprintf(`Download and set up an official OpenCore template.

Templates are fetched from: %s

Use --list to see all available templates.

Examples:
  opencore clone --list
  opencore clone chat
  opencore clone admin --api
  opencore clone chat --force
  opencore clone --list --branch develop
  opencore clone chat --branch develop`, templatesURL),
		Args: func(cmd *cobra.Command, args []string) error {
			listFlag, _ := cmd.Flags().GetBool("list")
			if listFlag {
				if len(args) != 0 {
					return fmt.Errorf("--list does not accept a template name")
				}
				return nil
			}
			if len(args) == 0 {
				return fmt.Errorf("missing template name\n\nUse 'opencore clone --list' to see available templates\n\nUsage: opencore clone <template>")
			}
			if len(args) > 1 {
				return fmt.Errorf("expected exactly one template name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			branch = normalizeBranch(branch)
			if err := validateBranch(branch); err != nil {
				return err
			}
			if listTemplates {
				return runListTemplates(cmd.Context(), branch)
			}
			return runClone(cmd, args, useAPI, force, branch)
		},
	}

	cmd.Flags().BoolVarP(&listTemplates, "list", "l", false, "List all available templates")
	cmd.Flags().BoolVar(&useAPI, "api", false, "Force using GitHub API instead of git sparse checkout")
	cmd.Flags().BoolVar(&force, "force", false, "Clone even if manifest compatibility does not match the current project")
	cmd.Flags().StringVarP(&branch, "branch", "b", "master", "Repository branch to use when listing/cloning templates")

	return cmd
}

func normalizeBranch(branch string) string {
	trimmed := strings.TrimSpace(branch)
	if trimmed == "" {
		return "master"
	}

	return trimmed
}

func validateBranch(branch string) error {
	if strings.HasPrefix(branch, "-") || strings.ContainsAny(branch, "\\\x00 ~^:?*[") ||
		strings.Contains(branch, "..") || strings.Contains(branch, "@{") || strings.Contains(branch, "//") ||
		strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") || strings.HasSuffix(branch, ".lock") {
		return fmt.Errorf("invalid branch name %q", branch)
	}
	return nil
}

func runListTemplates(ctx context.Context, branch string) error {
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Available Templates"))
	fmt.Println()

	resources, standalones, err := fetchGroupedTemplates(ctx, branch)
	if err != nil {
		return fmt.Errorf("failed to fetch templates: %w", err)
	}

	totalCount := len(resources) + len(standalones)
	if totalCount == 0 {
		fmt.Println(ui.Warning("No templates found in repository"))
		return nil
	}

	fmt.Println(ui.Info(fmt.Sprintf("Found %d templates in %s (branch: %s):\n", totalCount, templatesURL, branch)))

	// Show Resources
	if len(resources) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Resources") + ui.MutedStyle.Render(" (framework-connected modules)"))
		for _, t := range resources {
			fmt.Printf("  • %s %s\n", t.Manifest.effectiveName(t.Name), ui.MutedStyle.Render("["+compatibilityStatusLabel(t)+"]"))
		}
		fmt.Println()
	}

	// Show Standalones
	if len(standalones) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Standalones") + ui.MutedStyle.Render(" (independent scripts)"))
		for _, t := range standalones {
			fmt.Printf("  • %s %s\n", t.Manifest.effectiveName(t.Name), ui.MutedStyle.Render("["+compatibilityStatusLabel(t)+"]"))
		}
		fmt.Println()
	}

	fmt.Println(ui.SubtitleStyle.Render("Usage: opencore clone <template>"))

	return nil
}

func buildContentsAPIURL(path, branch string) string {
	baseURL := apiBaseURL
	if path != "" {
		baseURL = fmt.Sprintf("%s/%s", apiBaseURL, path)
	}

	return fmt.Sprintf("%s?ref=%s", baseURL, url.QueryEscape(branch))
}

// fetchGroupedTemplates fetches templates grouped by category (resources vs standalones)
func fetchGroupedTemplates(ctx context.Context, branch string) (resources []templateDescriptor, standalones []templateDescriptor, err error) {
	// Fetch root contents
	resp, err := cloneGET(ctx, buildContentsAPIURL("", branch))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil, fmt.Errorf("branch '%s' not found in templates repository", branch)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := decodeLimitedJSON(resp.Body, apiBodyLimit, &contents); err != nil {
		return nil, nil, err
	}

	// Check for container folders (resources, standalones, standalone)
	for _, item := range contents {
		if item.Type != "dir" || strings.HasPrefix(item.Name, "_") {
			continue
		}

		// Check if this is a container folder
		switch item.Name {
		case "resources":
			// Fetch contents of resources/
			resourceList, err := fetchFolderContents(ctx, "resources", templateCategoryResource, branch)
			if err == nil {
				resources = resourceList
			}
		case "standalones", "standalone":
			// Fetch contents of standalones/ or standalone/
			standaloneList, err := fetchFolderContents(ctx, item.Name, templateCategoryStandalone, branch)
			if err == nil {
				standalones = standaloneList
			}
		}
	}

	return resources, standalones, nil
}

// fetchFolderContents fetches the list of directories inside a folder
func fetchFolderContents(ctx context.Context, folderPath string, category templateCategory, branch string) ([]templateDescriptor, error) {
	requestURL := buildContentsAPIURL(folderPath, branch)
	resp, err := cloneGET(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := decodeLimitedJSON(resp.Body, apiBodyLimit, &contents); err != nil {
		return nil, err
	}

	var items []templateDescriptor
	for _, item := range contents {
		// Only include directories, skip files and _ folders
		if item.Type == "dir" && !strings.HasPrefix(item.Name, "_") {
			repoPath := folderPath + "/" + item.Name
			manifest, manifestErr := fetchTemplateManifest(ctx, repoPath, branch)
			if manifestErr == nil {
				manifestErr = validateManifestCategory(manifest, category)
			}
			items = append(items, templateDescriptor{
				Name:          item.Name,
				SourcePath:    repoPath,
				TargetPath:    filepath.Join(folderPath, item.Name),
				Category:      category,
				Manifest:      manifest,
				ManifestError: manifestErr,
			})
		}
	}

	return items, nil
}

func fetchTemplateManifest(ctx context.Context, templatePath, branch string) (*templateManifest, error) {
	requestURL := buildContentsAPIURL(templatePath, branch)
	resp, err := cloneGET(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := decodeLimitedJSON(resp.Body, apiBodyLimit, &contents); err != nil {
		return nil, err
	}

	for _, item := range contents {
		if item.Type != "file" || item.Name != ocManifestFileName || item.DownloadURL == "" {
			continue
		}

		manifestResp, err := cloneGET(ctx, item.DownloadURL)
		if err != nil {
			return nil, err
		}
		defer manifestResp.Body.Close()

		if manifestResp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to download %s: status %d", ocManifestFileName, manifestResp.StatusCode)
		}

		body, err := readLimited(manifestResp.Body, manifestLimit)
		if err != nil {
			return nil, err
		}

		return parseTemplateManifest(body)
	}

	return nil, nil
}

// resolveTemplatePaths determines the source path in the repo and target path locally
func resolveTemplate(ctx context.Context, templateName, branch string) (templateDescriptor, error) {
	// Prevent cloning container folders
	if templateName == "resources" || templateName == "standalones" || templateName == "standalone" {
		return templateDescriptor{}, fmt.Errorf("cannot clone container folders directly\n\nUse 'opencore clone --list' to see available templates")
	}

	resources, standalones, err := fetchGroupedTemplates(ctx, branch)
	if err != nil {
		return templateDescriptor{}, fmt.Errorf("failed to fetch templates: %w", err)
	}

	// Check if template exists in resources/
	for _, res := range resources {
		if res.Name == templateName {
			return res, nil
		}
	}

	// Check if template exists in standalones/
	for _, std := range standalones {
		if std.Name == templateName {
			return std, nil
		}
	}

	// Template not found
	return templateDescriptor{}, fmt.Errorf("template '%s' not found in branch '%s'\n\nUse 'opencore clone --list --branch %s' to see available templates", templateName, branch, branch)
}

type cloneModel struct {
	ctx        context.Context
	spinner    spinner.Model
	template   string // Display name (e.g., "chat")
	sourcePath string // Full path in repo (e.g., "resources/chat")
	targetPath string // Local path (e.g., "resources/chat")
	branch     string
	useAPI     bool
	status     string
	done       bool
	err        error
}

func (m cloneModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startClone(),
	)
}

func (m cloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case cloneStatusMsg:
		m.status = msg.status
		return m, nil
	case cloneResultMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m cloneModel) View() string {
	if m.done {
		if m.err != nil {
			return ui.Error(fmt.Sprintf("Failed to clone template: %v", m.err)) + "\n"
		}
		resolved, _ := pkgmgr.Resolve(pkgmgr.EffectivePreference("."))
		return ui.Success(fmt.Sprintf("Template '%s' cloned successfully!", m.template)) + "\n\n" +
			ui.BoxStyle.Render(fmt.Sprintf("Location: %s\n\nNext steps:\n  cd %s\n  %s\n\nRemember to add to opencore.config.ts:\n  resources: {\n    include: ['./resources/*'],\n  }\n  // Or if it is a standalone:\n  standalones: {\n    include: ['./standalones/*'],\n  }", m.targetPath, m.targetPath, resolved.InstallCmd()))
	}

	status := m.status
	if status == "" {
		status = "Preparing..."
	}
	return fmt.Sprintf("%s %s\n", m.spinner.View(), status)
}

type cloneStatusMsg struct {
	status string
}

type cloneResultMsg struct {
	err error
}

func (m cloneModel) startClone() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Minute)
		defer cancel()
		// Check if directory already exists
		if _, err := os.Stat(m.targetPath); !os.IsNotExist(err) {
			return cloneResultMsg{err: fmt.Errorf("directory '%s' already exists", m.targetPath)}
		}

		// Try sparse checkout first if git >= 2.25 and not forced to use API
		if !m.useAPI && canUseSparseCheckout() {
			err := cloneWithSparseCheckout(ctx, m.sourcePath, m.targetPath, m.branch)
			if err == nil {
				return cloneResultMsg{err: nil}
			}
			// If sparse checkout fails, fall back to API
		}

		// Use GitHub API
		err := cloneWithGitHubAPI(ctx, m.sourcePath, m.targetPath, m.branch)
		return cloneResultMsg{err: err}
	}
}

// canUseSparseCheckout checks if git version >= 2.25
func canUseSparseCheckout() bool {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version from "git version 2.39.0" or similar
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 3 {
		return false
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	return major > 2 || (major == 2 && minor >= 25)
}

// cloneWithSparseCheckout uses git sparse-checkout to clone only the template folder
func cloneWithSparseCheckout(ctx context.Context, templatePath, targetPath, branch string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(filepath.Dir(targetPath), ".opencore-clone-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Initialize repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", templatesURL + ".git"},
		{"git", "config", "core.sparseCheckout", "true"},
	}

	for _, args := range cmds {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git command failed: %w", err)
		}
	}

	// Configure sparse-checkout
	sparseFile := filepath.Join(tempDir, ".git", "info", "sparse-checkout")
	if err := os.WriteFile(sparseFile, []byte(templatePath+"/\n"), 0644); err != nil {
		return err
	}

	// Pull
	cmd := exec.CommandContext(ctx, "git", "pull", "--depth=1", "origin", branch)
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Move template folder to target
	srcPath := filepath.Join(tempDir, filepath.FromSlash(templatePath))
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("template '%s' not found in repository", templatePath)
	}

	if _, err := os.Lstat(targetPath); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("target path '%s' already exists", targetPath)
	}
	if err := os.Rename(srcPath, targetPath); err != nil {
		return fmt.Errorf("failed to publish cloned template: %w", err)
	}

	return nil
}

// cloneWithGitHubAPI downloads template using GitHub API
func cloneWithGitHubAPI(ctx context.Context, templatePath, targetPath, branch string) error {
	// First verify template exists
	apiURL := buildContentsAPIURL(templatePath, branch)
	resp, err := cloneGET(ctx, apiURL)
	if err != nil {
		return fmt.Errorf("failed to connect to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("template '%s' not found in branch '%s'. Use 'opencore clone --list --branch %s' to see available templates", templatePath, branch, branch)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	stagingPath, err := os.MkdirTemp(filepath.Dir(targetPath), ".opencore-download-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stagingPath)

	budget := int64(cloneBodyLimit)
	if err := downloadDirectory(ctx, templatePath, stagingPath, branch, &budget); err != nil {
		return err
	}
	if _, err := os.Lstat(targetPath); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("target path '%s' already exists", targetPath)
	}
	if err := os.Rename(stagingPath, targetPath); err != nil {
		return fmt.Errorf("failed to publish downloaded template: %w", err)
	}
	return nil
}

func downloadDirectory(ctx context.Context, remotePath, localPath, branch string, budget *int64) error {
	apiURL := buildContentsAPIURL(remotePath, branch)
	resp, err := cloneGET(ctx, apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API error: status %d while reading %s", resp.StatusCode, remotePath)
	}

	var contents []GitHubContent
	if err := decodeLimitedJSON(resp.Body, apiBodyLimit, &contents); err != nil {
		return err
	}

	for _, item := range contents {
		if err := validateGitHubItem(remotePath, item); err != nil {
			return err
		}
		localItemPath := filepath.Join(localPath, item.Name)

		if item.Type == "dir" {
			if err := os.MkdirAll(localItemPath, 0755); err != nil {
				return err
			}
			if err := downloadDirectory(ctx, item.Path, localItemPath, branch, budget); err != nil {
				return err
			}
		} else if item.Type == "file" {
			if err := downloadFile(ctx, item.DownloadURL, localItemPath, budget); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unsupported GitHub item type %q for %s", item.Type, item.Path)
		}
	}

	return nil
}

func downloadFile(ctx context.Context, downloadURL, localPath string, budget *int64) error {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() != "raw.githubusercontent.com" {
		return fmt.Errorf("refusing untrusted download URL %q", downloadURL)
	}
	resp, err := cloneGET(ctx, downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}
	limit := int64(fileBodyLimit)
	if *budget < limit {
		limit = *budget
	}
	if limit <= 0 || (resp.ContentLength >= 0 && resp.ContentLength > limit) {
		return fmt.Errorf("download exceeds size limit")
	}

	file, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(resp.Body, limit+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written > limit {
		_ = os.Remove(localPath)
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		return fmt.Errorf("download exceeds size limit")
	}
	*budget -= written
	return nil
}

func runClone(cmd *cobra.Command, args []string, forceAPI bool, force bool, branch string) error {
	fmt.Println(ui.TitleStyle.Render("Clone Template"))
	fmt.Println()

	templateName := args[0]

	if err := templates.ValidateName(templateName); err != nil {
		return fmt.Errorf("invalid template name: %w", err)
	}

	// Prevent cloning system folders (folders starting with _)
	if strings.HasPrefix(templateName, "_") {
		return fmt.Errorf("cannot clone system folders (folders starting with '_')\n\nUse 'opencore clone --list' to see available templates")
	}

	template, err := resolveTemplate(cmd.Context(), templateName, branch)
	if err != nil {
		return err
	}
	if template.ManifestError != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Skipping manifest checks for '%s': %v", template.Name, template.ManifestError)))
		fmt.Println()
	}

	if runtime, ok, err := detectCurrentProjectRuntime(); err != nil {
		return err
	} else if ok {
		if compatibilityErr := validateManifestCompatibility(template, runtime); compatibilityErr != nil {
			if !force {
				return compatibilityErr
			}
			fmt.Println(ui.Warning(compatibilityErr.Error()))
			fmt.Println()
		}
	} else if template.Manifest != nil && template.Manifest.Compatibility != nil && len(template.Manifest.Compatibility.Runtimes) > 0 {
		fmt.Println(ui.Info(fmt.Sprintf("Template '%s' declares compatibility with: %s", template.Name, strings.Join(template.Manifest.Compatibility.Runtimes, ", "))))
		fmt.Println(ui.MutedStyle.Render("No local opencore.config.ts found, skipping compatibility enforcement."))
		fmt.Println()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	m := cloneModel{
		ctx:        cmd.Context(),
		spinner:    s,
		template:   templateName,
		sourcePath: template.SourcePath,
		targetPath: template.TargetPath,
		branch:     branch,
		useAPI:     forceAPI,
		done:       false,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if fm, ok := finalModel.(cloneModel); ok && fm.err != nil {
		return fm.err
	}

	if runtime, ok, err := detectCurrentProjectRuntime(); err != nil {
		return err
	} else if ok {
		if err := applyPostCloneRuntimeAdjustments(template.TargetPath, runtime); err != nil {
			return err
		}
	}

	return nil
}

func cloneGET(ctx context.Context, requestURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "opencore-cli")
	return cloneHTTPClient.Do(req)
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d-byte size limit", limit)
	}
	return body, nil
}

func decodeLimitedJSON(reader io.Reader, limit int64, target any) error {
	body, err := readLimited(reader, limit)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}
	return nil
}

func validateGitHubItem(parent string, item GitHubContent) error {
	if item.Name == "" || item.Name == "." || item.Name == ".." || path.Base(item.Name) != item.Name || filepath.Base(item.Name) != item.Name {
		return fmt.Errorf("unsafe item name %q returned by GitHub", item.Name)
	}
	if item.Path != path.Join(parent, item.Name) {
		return fmt.Errorf("unexpected item path %q returned by GitHub", item.Path)
	}
	return nil
}

func applyPostCloneRuntimeAdjustments(targetPath, runtime string) error {
	if runtime != "ragemp" {
		return nil
	}

	manifestPath := filepath.Join(targetPath, "fxmanifest.lua")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to inspect cloned manifest: %w", err)
	}

	if err := os.Remove(manifestPath); err != nil {
		return fmt.Errorf("failed to remove fxmanifest.lua for ragemp template clone: %w", err)
	}

	return nil
}

func detectCurrentProjectRuntime() (runtime string, ok bool, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false, err
	}

	if _, err := config.FindProjectRoot(wd); err != nil {
		return "", false, nil
	}

	cfg, _, err := config.LoadWithProjectRoot()
	if err != nil {
		return "", false, fmt.Errorf("failed to load current project config: %w", err)
	}

	return cfg.RuntimeKind(), true, nil
}
