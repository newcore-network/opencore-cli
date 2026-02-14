package builder

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

func (b *Builder) validateTaskSources(tasks []BuildTask) error {
	resourcePaths := make(map[string]struct{})
	for _, task := range tasks {
		if task.Type == TypeViews || !task.Options.Compile {
			continue
		}
		resourcePaths[filepath.Clean(task.Path)] = struct{}{}
	}

	if len(resourcePaths) == 0 {
		return nil
	}

	var allIssues []SourceValidationIssue
	for resourcePath := range resourcePaths {
		issues, err := b.resourceBuilder.validateSourceFiles(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to validate resource %s: %w", resourcePath, err)
		}
		for _, issue := range issues {
			if !filepath.IsAbs(issue.File) {
				issue.File = filepath.ToSlash(filepath.Join(resourcePath, filepath.FromSlash(issue.File)))
			} else {
				issue.File = filepath.ToSlash(issue.File)
			}
			allIssues = append(allIssues, issue)
		}
	}

	if len(allIssues) == 0 {
		return nil
	}

	sort.Slice(allIssues, func(i, j int) bool {
		if allIssues[i].File != allIssues[j].File {
			return allIssues[i].File < allIssues[j].File
		}
		if allIssues[i].Line != allIssues[j].Line {
			return allIssues[i].Line < allIssues[j].Line
		}
		return allIssues[i].Message < allIssues[j].Message
	})

	fmt.Println(ui.Warning("Source validation failed. Build cancelled."))
	for _, issue := range allIssues {
		fmt.Println(ui.Warning(issue.String()))
	}

	return fmt.Errorf("source validation failed: %d issue(s) detected", len(allIssues))
}
