package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", rel, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", rel, err)
	}
}

func TestValidateSourceFiles_MixedNamespacesInSameTsFile(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/mixed.ts", `
@Client.Controller()
export class MixedController {
  @Server.Command('x')
  handle() {}
}
`)

	issues, err := rb.validateSourceFiles(resourcePath)
	if err != nil {
		t.Fatalf("validateSourceFiles returned error: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if !strings.Contains(issues[0].Message, "@Client.*") || !strings.Contains(issues[0].Message, "@Server.*") {
		t.Fatalf("unexpected issue message: %s", issues[0].Message)
	}
}

func TestValidateSourceFiles_InvalidNodeModulesFrameworkImport(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server.ts", `
import { Server } from '../../node_modules/@open-core/framework/server'

@Server.Controller()
export class DemoController {}
`)

	issues, err := rb.validateSourceFiles(resourcePath)
	if err != nil {
		t.Fatalf("validateSourceFiles returned error: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if !strings.Contains(issues[0].Message, "node_modules") {
		t.Fatalf("unexpected issue message: %s", issues[0].Message)
	}
}

func TestValidateSourceFiles_ValidFrameworkPackageImportsAreAllowed(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server.ts", `
import { Server } from '@open-core/framework/server'
import { z } from '@open-core/framework'

@Server.Controller()
export class DemoController {}
`)

	issues, err := rb.validateSourceFiles(resourcePath)
	if err != nil {
		t.Fatalf("validateSourceFiles returned error: %v", err)
	}

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(issues))
	}
}

func TestValidateSourceFiles_IgnoresNonTsAndDtsFiles(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/ignored.js", `
@Client.Controller()
@Server.Command('x')
`)

	writeTestFile(t, resourcePath, "src/types.d.ts", `
@Client.Controller()
@Server.Command('x')
`)

	writeTestFile(t, resourcePath, "src/client.ts", `
@Client.Controller()
export class ClientOnly {}
`)

	issues, err := rb.validateSourceFiles(resourcePath)
	if err != nil {
		t.Fatalf("validateSourceFiles returned error: %v", err)
	}

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(issues))
	}
}
