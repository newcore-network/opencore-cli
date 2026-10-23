package builder

import (
	"bufio"
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// createdLibrary is a library a resource creates through `createServerLibrary` or
// `createClientLibrary`, keyed as its events are: by side and literal name.
type createdLibrary struct {
	Kind typegenKind `json:"kind"`
	Name string      `json:"name"`
}

// libraryCreation locates the first call creating a library, for diagnostics.
type libraryCreation struct {
	createdLibrary
	SourceFile string
	SourceLine int
}

// createdLibrariesMarker prefixes the line of a generated file that lists the libraries its
// resource creates, so typegen can tell when another resource creates one with the same name.
const createdLibrariesMarker = "// opencore:libraries "

// libraryCreations lists the libraries a resource creates, once each, in a stable order.
func libraryCreations(analysis *resourceAnalysis, resourcePath string) []libraryCreation {
	var creations []libraryCreation
	for _, source := range analysis.sources {
		relPath := relativeSourcePath(resourcePath, source.path)
		for _, library := range source.result.Libraries {
			creations = append(creations, libraryCreation{
				createdLibrary: createdLibrary{Kind: library.Kind, Name: library.Library},
				SourceFile:     relPath,
				SourceLine:     library.Line,
			})
		}
	}

	slices.SortStableFunc(creations, func(a, b libraryCreation) int {
		return cmp.Or(
			strings.Compare(string(a.Kind), string(b.Kind)),
			strings.Compare(a.Name, b.Name),
			strings.Compare(a.SourceFile, b.SourceFile),
			cmp.Compare(a.SourceLine, b.SourceLine),
		)
	})
	return slices.CompactFunc(creations, func(a, b libraryCreation) bool {
		return a.createdLibrary == b.createdLibrary
	})
}

func writeCreatedLibrariesMarker(b *strings.Builder, creations []libraryCreation) {
	if len(creations) == 0 {
		return
	}
	libraries := make([]createdLibrary, len(creations))
	for i, creation := range creations {
		libraries[i] = creation.createdLibrary
	}
	encoded, _ := json.Marshal(libraries)
	fmt.Fprintf(b, "%s%s\n\n", createdLibrariesMarker, encoded)
}

// libraryCollisionWarnings reports each library this resource creates under a name another
// resource of the project also creates one with.
//
// Every resource of a project is type-checked in the same program, and the framework keys
// library events by name alone, so two unrelated libraries sharing a name have their events
// merged. At runtime they never meet: each resource has its own library event bus. Other
// resources are read from their generated files, so a collision is reported by whichever of the
// two resources is generated after the other.
func libraryCollisionWarnings(projectPath string, resourcePath string, creations []libraryCreation) []SourceValidationIssue {
	if len(creations) == 0 {
		return nil
	}

	owners := map[createdLibrary][]string{}
	for _, genFile := range otherGeneratedTypeFiles(projectPath, resourcePath) {
		resourceKey := resourceRegisterKey(projectPath, filepath.Dir(filepath.Dir(genFile)))
		for _, library := range readCreatedLibraries(genFile) {
			owners[library] = append(owners[library], resourceKey)
		}
	}

	var warnings []SourceValidationIssue
	for _, creation := range creations {
		others := owners[creation.createdLibrary]
		if len(others) == 0 {
			continue
		}
		slices.Sort(others)
		warnings = append(warnings, SourceValidationIssue{
			File: creation.SourceFile,
			Line: creation.SourceLine,
			Message: fmt.Sprintf(
				"%s library %q is also created by %s; typegen merges the events of libraries sharing a name, so give one of them a different name",
				librarySide(creation.Kind), creation.Name, strings.Join(others, ", "),
			),
		})
	}
	return warnings
}

func librarySide(kind typegenKind) string {
	if kind == kindClientLibrary {
		return "client"
	}
	return "server"
}

// otherGeneratedTypeFiles finds the generated file of every resource of the project except
// resourcePath, skipping dependencies and hidden directories.
func otherGeneratedTypeFiles(projectPath string, resourcePath string) []string {
	ownDir := filepath.Join(filepath.Clean(resourcePath), ".opencore")
	var files []string
	_ = filepath.WalkDir(projectPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path == projectPath {
			return nil
		}
		if name == ".opencore" {
			if path != ownDir {
				if genFile := filepath.Join(path, typegenFileName); fileExists(genFile) {
					files = append(files, genFile)
				}
			}
			return filepath.SkipDir
		}
		if name == "node_modules" || strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		return nil
	})
	return files
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// readCreatedLibraries reads the libraries a generated file declares its resource creates. The
// marker sits in the header, so only the first lines are scanned.
func readCreatedLibraries(genFile string) []createdLibrary {
	f, err := os.Open(genFile)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for line := 0; line < 10 && scanner.Scan(); line++ {
		encoded, ok := strings.CutPrefix(scanner.Text(), createdLibrariesMarker)
		if !ok {
			continue
		}
		var libraries []createdLibrary
		if json.Unmarshal([]byte(encoded), &libraries) != nil {
			return nil
		}
		return libraries
	}
	return nil
}
