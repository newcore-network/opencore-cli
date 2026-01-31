package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (rb *ResourceBuilder) generateAutoloadControllers(resourcePath string) error {
	resourcePath = filepath.Clean(resourcePath)

	outDir := filepath.Join(resourcePath, ".opencore")
	serverOutFile := filepath.Join(outDir, "autoload.server.controllers.ts")
	clientOutFile := filepath.Join(outDir, "autoload.client.controllers.ts")
	baseDir := outDir

	var serverImports []string
	var clientImports []string

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
		if !(strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx")) {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(content)
		hasServer := strings.Contains(text, "@Server.Controller")
		hasClient := strings.Contains(text, "@Client.Controller")
		if !hasServer && !hasClient {
			return nil
		}

		relImport, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		relImport = filepath.ToSlash(relImport)

		if !strings.HasPrefix(relImport, ".") {
			relImport = "./" + relImport
		}

		relImport = strings.TrimSuffix(relImport, ".ts")
		relImport = strings.TrimSuffix(relImport, ".tsx")

		if hasServer {
			serverImports = append(serverImports, fmt.Sprintf("import %q;\n", relImport))
		}
		if hasClient {
			clientImports = append(clientImports, fmt.Sprintf("import %q;\n", relImport))
		}
		return nil
	})
	if err != nil {
		return err
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
