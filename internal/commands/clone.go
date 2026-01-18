package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

const (
	templatesRepo = "newcore-network/opencore-templates"
	templatesURL  = "https://github.com/" + templatesRepo
	apiBaseURL    = "https://api.github.com/repos/" + templatesRepo + "/contents"
)

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

	cmd := &cobra.Command{
		Use:   "clone <template>",
		Short: "Clone an official template",
		Long: fmt.Sprintf(`Download and set up an official OpenCore template.

Templates are fetched from: %s

Use --list to see all available templates.

Examples:
  opencore clone --list
  opencore clone chat
  opencore clone admin --api`, templatesURL),
		Args: func(cmd *cobra.Command, args []string) error {
			listFlag, _ := cmd.Flags().GetBool("list")
			if listFlag {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("missing template name\n\nUse 'opencore clone --list' to see available templates\n\nUsage: opencore clone <template>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if listTemplates {
				return runListTemplates()
			}
			return runClone(cmd, args, useAPI)
		},
	}

	cmd.Flags().BoolVarP(&listTemplates, "list", "l", false, "List all available templates")
	cmd.Flags().BoolVar(&useAPI, "api", false, "Force using GitHub API instead of git sparse checkout")

	return cmd
}

func runListTemplates() error {
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Available Templates"))
	fmt.Println()

	resources, standalones, err := fetchGroupedTemplates()
	if err != nil {
		return fmt.Errorf("failed to fetch templates: %w", err)
	}

	totalCount := len(resources) + len(standalones)
	if totalCount == 0 {
		fmt.Println(ui.Warning("No templates found in repository"))
		return nil
	}

	fmt.Println(ui.Info(fmt.Sprintf("Found %d templates in %s:\n", totalCount, templatesURL)))

	// Show Resources
	if len(resources) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Resources") + ui.MutedStyle.Render(" (framework-connected modules)"))
		for _, t := range resources {
			fmt.Printf("  • %s\n", t)
		}
		fmt.Println()
	}

	// Show Standalones
	if len(standalones) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Standalones") + ui.MutedStyle.Render(" (independent scripts)"))
		for _, t := range standalones {
			fmt.Printf("  • %s\n", t)
		}
		fmt.Println()
	}

	fmt.Println(ui.SubtitleStyle.Render("Usage: opencore clone <template>"))

	return nil
}

// fetchGroupedTemplates fetches templates grouped by category (resources vs standalones)
func fetchGroupedTemplates() (resources []string, standalones []string, err error) {
	// Fetch root contents
	resp, err := http.Get(apiBaseURL)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
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
			resourceList, err := fetchFolderContents("resources")
			if err == nil {
				resources = resourceList
			}
		case "standalones", "standalone":
			// Fetch contents of standalones/ or standalone/
			standaloneList, err := fetchFolderContents(item.Name)
			if err == nil {
				standalones = standaloneList
			}
		}
	}

	return resources, standalones, nil
}

// fetchFolderContents fetches the list of directories inside a folder
func fetchFolderContents(folderPath string) ([]string, error) {
	url := fmt.Sprintf("%s/%s", apiBaseURL, folderPath)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, err
	}

	var items []string
	for _, item := range contents {
		// Only include directories, skip files and _ folders
		if item.Type == "dir" && !strings.HasPrefix(item.Name, "_") {
			items = append(items, item.Name)
		}
	}

	return items, nil
}

// resolveTemplatePaths determines the source path in the repo and target path locally
func resolveTemplatePaths(templateName string) (sourcePath, targetPath string, err error) {
	// Prevent cloning container folders
	if templateName == "resources" || templateName == "standalones" || templateName == "standalone" {
		return "", "", fmt.Errorf("cannot clone container folders directly\n\nUse 'opencore clone --list' to see available templates")
	}

	resources, standalones, err := fetchGroupedTemplates()
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch templates: %w", err)
	}

	// Check if template exists in resources/
	for _, res := range resources {
		if res == templateName {
			return "resources/" + templateName, filepath.Join("resources", templateName), nil
		}
	}

	// Check if template exists in standalones/
	for _, std := range standalones {
		if std == templateName {
			return "standalones/" + templateName, filepath.Join("standalones", templateName), nil
		}
	}

	// Template not found
	return "", "", fmt.Errorf("template '%s' not found\n\nUse 'opencore clone --list' to see available templates", templateName)
}

func fetchTemplateList() ([]string, error) {
	resp, err := http.Get(apiBaseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, err
	}

	var templates []string
	for _, item := range contents {
		// Only include directories, skip files like README.md
		// Also skip directories starting with _ (system folders)
		if item.Type == "dir" && !strings.HasPrefix(item.Name, "_") {
			templates = append(templates, item.Name)
		}
	}

	return templates, nil
}

type cloneModel struct {
	spinner    spinner.Model
	template   string // Display name (e.g., "chat")
	sourcePath string // Full path in repo (e.g., "resources/chat")
	targetPath string // Local path (e.g., "resources/chat")
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
		return ui.Success(fmt.Sprintf("Template '%s' cloned successfully!", m.template)) + "\n\n" +
			ui.BoxStyle.Render(fmt.Sprintf("Location: %s\n\nNext steps:\n  cd %s\n  pnpm install\n\nRemember to add to opencore.config.ts:\n  resources: {\n    include: ['./resources/*'],\n  }\n  // Or if it is a standalone:\n  standalones: {\n    include: ['./standalones/*'],\n  }", m.targetPath, m.targetPath))
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
		// Check if directory already exists
		if _, err := os.Stat(m.targetPath); !os.IsNotExist(err) {
			return cloneResultMsg{err: fmt.Errorf("directory '%s' already exists", m.targetPath)}
		}

		// Try sparse checkout first if git >= 2.25 and not forced to use API
		if !m.useAPI && canUseSparseCheckout() {
			err := cloneWithSparseCheckout(m.sourcePath, m.targetPath)
			if err == nil {
				return cloneResultMsg{err: nil}
			}
			// If sparse checkout fails, fall back to API
		}

		// Use GitHub API
		err := cloneWithGitHubAPI(m.sourcePath, m.targetPath)
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
func cloneWithSparseCheckout(template, targetPath string) error {
	tempDir, err := os.MkdirTemp("", "opencore-clone-*")
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
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git command failed: %w", err)
		}
	}

	// Configure sparse-checkout
	sparseFile := filepath.Join(tempDir, ".git", "info", "sparse-checkout")
	if err := os.WriteFile(sparseFile, []byte(template+"/\n"), 0644); err != nil {
		return err
	}

	// Pull
	cmd := exec.Command("git", "pull", "origin", "main", "--depth=1")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Move template folder to target
	srcPath := filepath.Join(tempDir, template)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("template '%s' not found in repository", template)
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// Move the folder
	if err := os.Rename(srcPath, targetPath); err != nil {
		// If rename fails (cross-device), copy instead
		return copyDir(srcPath, targetPath)
	}

	return nil
}

// cloneWithGitHubAPI downloads template using GitHub API
func cloneWithGitHubAPI(template, targetPath string) error {
	// First verify template exists
	apiURL := fmt.Sprintf("%s/%s", apiBaseURL, template)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to connect to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("template '%s' not found. Use 'opencore clone --list' to see available templates", template)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}

	// Download recursively
	return downloadDirectory(template, targetPath)
}

func downloadDirectory(remotePath, localPath string) error {
	apiURL := fmt.Sprintf("%s/%s", apiBaseURL, remotePath)
	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return err
	}

	for _, item := range contents {
		localItemPath := filepath.Join(localPath, item.Name)

		if item.Type == "dir" {
			if err := os.MkdirAll(localItemPath, 0755); err != nil {
				return err
			}
			if err := downloadDirectory(item.Path, localItemPath); err != nil {
				return err
			}
		} else {
			if err := downloadFile(item.DownloadURL, localItemPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func downloadFile(url, localPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func runClone(cmd *cobra.Command, args []string, forceAPI bool) error {
	fmt.Println(ui.TitleStyle.Render("Clone Template"))
	fmt.Println()

	templateName := args[0]

	// Validate template name (basic sanitization)
	if strings.Contains(templateName, "/") || strings.Contains(templateName, "..") {
		return fmt.Errorf("invalid template name: %s", templateName)
	}

	// Prevent cloning system folders (folders starting with _)
	if strings.HasPrefix(templateName, "_") {
		return fmt.Errorf("cannot clone system folders (folders starting with '_')\n\nUse 'opencore clone --list' to see available templates")
	}

	// Determine the source path and target path
	sourcePath, targetPath, err := resolveTemplatePaths(templateName)
	if err != nil {
		return err
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	m := cloneModel{
		spinner:    s,
		template:   templateName,
		sourcePath: sourcePath,
		targetPath: targetPath,
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

	return nil
}
