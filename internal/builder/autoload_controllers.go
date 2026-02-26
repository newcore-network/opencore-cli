package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	clientDecoratorPattern                = regexp.MustCompile(`@Client\.[A-Za-z_][A-Za-z0-9_]*`)
	serverDecoratorPattern                = regexp.MustCompile(`@Server\.[A-Za-z_][A-Za-z0-9_]*`)
	serverControllerDecoratorPattern      = regexp.MustCompile(`@Server\.Controller\s*\(`)
	clientControllerDecoratorPattern      = regexp.MustCompile(`@Client\.Controller\s*\(`)
	controllerDecoratorPattern            = regexp.MustCompile(`@Controller\s*\(`)
	frameworkServerImportPattern          = regexp.MustCompile(`(?:from\s+['"]@open-core/framework/server['"]|import\s+['"]@open-core/framework/server['"]|require\(\s*['"]@open-core/framework/server['"]\s*\))`)
	frameworkClientImportPattern          = regexp.MustCompile(`(?:from\s+['"]@open-core/framework/client['"]|import\s+['"]@open-core/framework/client['"]|require\(\s*['"]@open-core/framework/client['"]\s*\))`)
	invalidFrameworkNodeModulesImportExpr = regexp.MustCompile(`(?:from\s+['"][^'"]*node_modules[\\/]+@open-core[\\/]framework(?:[\\/][^'"]*)?['"]|import\s+['"][^'"]*node_modules[\\/]+@open-core[\\/]framework(?:[\\/][^'"]*)?['"]|require\(\s*['"][^'"]*node_modules[\\/]+@open-core[\\/]framework(?:[\\/][^'"]*)?['"]\s*\))`)
)

type SourceValidationIssue struct {
	File    string
	Line    int
	Message string
}

func (i SourceValidationIssue) String() string {
	if i.Line > 0 {
		return fmt.Sprintf("%s:%d %s", i.File, i.Line, i.Message)
	}
	return fmt.Sprintf("%s %s", i.File, i.Message)
}

func scanResourceTypeScriptFiles(resourcePath string, baseDir string, serverOutFile string, clientOutFile string) ([]string, []string, []SourceValidationIssue, error) {
	var serverImports []string
	var clientImports []string
	var issues []SourceValidationIssue

	err := filepath.WalkDir(resourcePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "dist", ".opencore":
				return filepath.SkipDir
			}
			return nil
		}

		if path == serverOutFile || path == clientOutFile {
			return nil
		}

		name := d.Name()
		if strings.HasSuffix(name, ".d.ts") {
			return nil
		}
		if !strings.HasSuffix(name, ".ts") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(content)

		relPath, relErr := filepath.Rel(resourcePath, path)
		if relErr != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		lines := strings.Split(text, "\n")
		clientDecoratorLine := 0
		serverDecoratorLine := 0
		controllerDecoratorLine := 0
		frameworkServerImportLine := 0
		frameworkClientImportLine := 0

		for idx, line := range lines {
			lineNumber := idx + 1
			if clientDecoratorLine == 0 && clientDecoratorPattern.MatchString(line) {
				clientDecoratorLine = lineNumber
			}
			if serverDecoratorLine == 0 && serverDecoratorPattern.MatchString(line) {
				serverDecoratorLine = lineNumber
			}
			if controllerDecoratorLine == 0 && controllerDecoratorPattern.MatchString(line) {
				controllerDecoratorLine = lineNumber
			}
			if frameworkServerImportLine == 0 && frameworkServerImportPattern.MatchString(line) {
				frameworkServerImportLine = lineNumber
			}
			if frameworkClientImportLine == 0 && frameworkClientImportPattern.MatchString(line) {
				frameworkClientImportLine = lineNumber
			}
			if invalidFrameworkNodeModulesImportExpr.MatchString(line) {
				issues = append(issues, SourceValidationIssue{
					File:    relPath,
					Line:    lineNumber,
					Message: "invalid framework import from node_modules path; use package import (e.g. @open-core/framework/server)",
				})
			}
		}

		if clientDecoratorLine > 0 && serverDecoratorLine > 0 {
			lineNumber := clientDecoratorLine
			if serverDecoratorLine < lineNumber {
				lineNumber = serverDecoratorLine
			}
			issues = append(issues, SourceValidationIssue{
				File:    relPath,
				Line:    lineNumber,
				Message: "mixed decorators detected: @Client.* and @Server.* cannot coexist in the same file",
			})
		}

		if controllerDecoratorLine > 0 && frameworkServerImportLine > 0 && frameworkClientImportLine > 0 {
			lineNumber := controllerDecoratorLine
			if frameworkServerImportLine < lineNumber {
				lineNumber = frameworkServerImportLine
			}
			if frameworkClientImportLine < lineNumber {
				lineNumber = frameworkClientImportLine
			}
			issues = append(issues, SourceValidationIssue{
				File:    relPath,
				Line:    lineNumber,
				Message: "ambiguous @Controller decorator detected: import either @open-core/framework/server or @open-core/framework/client, not both",
			})
		}

		hasServerController := serverControllerDecoratorPattern.MatchString(text)
		hasClientController := clientControllerDecoratorPattern.MatchString(text)

		hasGenericController := controllerDecoratorPattern.MatchString(text)
		if hasGenericController {
			if frameworkServerImportLine > 0 && frameworkClientImportLine == 0 {
				hasServerController = true
			}
			if frameworkClientImportLine > 0 && frameworkServerImportLine == 0 {
				hasClientController = true
			}
		}

		if !hasServerController && !hasClientController {
			return nil
		}

		relImport, relImportErr := filepath.Rel(baseDir, path)
		if relImportErr != nil {
			return relImportErr
		}
		relImport = filepath.ToSlash(relImport)

		if !strings.HasPrefix(relImport, ".") {
			relImport = "./" + relImport
		}

		relImport = strings.TrimSuffix(relImport, ".ts")

		if hasServerController {
			serverImports = append(serverImports, fmt.Sprintf("import %q;\n", relImport))
		}
		if hasClientController {
			clientImports = append(clientImports, fmt.Sprintf("import %q;\n", relImport))
		}
		return nil
	})

	if err != nil {
		return nil, nil, nil, err
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].File != issues[j].File {
			return issues[i].File < issues[j].File
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Message < issues[j].Message
	})

	return serverImports, clientImports, issues, nil
}

func (rb *ResourceBuilder) validateSourceFiles(resourcePath string) ([]SourceValidationIssue, error) {
	resourcePath = filepath.Clean(resourcePath)
	outDir := filepath.Join(resourcePath, ".opencore")
	serverOutFile := filepath.Join(outDir, "autoload.server.controllers.ts")
	clientOutFile := filepath.Join(outDir, "autoload.client.controllers.ts")

	_, _, issues, err := scanResourceTypeScriptFiles(resourcePath, outDir, serverOutFile, clientOutFile)
	if err != nil {
		return nil, err
	}

	return issues, nil
}

func (rb *ResourceBuilder) generateAutoloadControllers(resourcePath string) error {
	resourcePath = filepath.Clean(resourcePath)

	outDir := filepath.Join(resourcePath, ".opencore")
	serverOutFile := filepath.Join(outDir, "autoload.server.controllers.ts")
	clientOutFile := filepath.Join(outDir, "autoload.client.controllers.ts")
	baseDir := outDir

	serverImports, clientImports, issues, err := scanResourceTypeScriptFiles(resourcePath, baseDir, serverOutFile, clientOutFile)
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		return fmt.Errorf("source validation failed with %d issue(s)", len(issues))
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	sort.Strings(serverImports)
	sort.Strings(clientImports)

	serverContent := ""
	if len(serverImports) == 0 {
		serverContent = "export {};\n"
	} else {
		serverContent = strings.Join(serverImports, "")
	}

	clientContent := ""
	if len(clientImports) == 0 {
		clientContent = "export {};\n"
	} else {
		clientContent = strings.Join(clientImports, "")
	}

	if err := os.WriteFile(serverOutFile, []byte(serverContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(clientOutFile, []byte(clientContent), 0644); err != nil {
		return err
	}

	return nil
}
