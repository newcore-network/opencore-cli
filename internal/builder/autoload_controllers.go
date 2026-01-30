package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (rb *ResourceBuilder) generateAutoloadServerControllers(resourcePath string) error {
	resourcePath = filepath.Clean(resourcePath)

	primaryOutFile := filepath.Join(resourcePath, "src", ".opencore", "autoload.server.controllers.ts")
	primaryOutDir := filepath.Dir(primaryOutFile)
	baseDir := primaryOutDir

	mirrorOutFile := filepath.Join(resourcePath, ".opencore", "autoload.server.controllers.ts")
	mirrorOutDir := filepath.Dir(mirrorOutFile)

	var imports []string

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

		if path == primaryOutFile || path == mirrorOutFile {
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
		if !strings.Contains(text, "@Server.Controller") && !strings.Contains(text, "@Client.Controller") {
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

		imports = append(imports, fmt.Sprintf("import %q;\n", relImport))
		return nil
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(primaryOutDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(mirrorOutDir, 0755); err != nil {
		return err
	}

	sort.Strings(imports)

	content := ""
	if len(imports) == 0 {
		content = "export {};\n"
	} else {
		content = strings.Join(imports, "")
	}

	if err := os.WriteFile(primaryOutFile, []byte(content), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(mirrorOutFile, []byte(content), 0644); err != nil {
		return err
	}

	return nil
}
