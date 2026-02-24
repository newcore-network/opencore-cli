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

func TestValidateSourceFiles_ControllerDecoratorWithServerImportIsAllowed(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server.ts", `
import { Controller } from '@open-core/framework/server'

@Controller()
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

func TestValidateSourceFiles_ControllerDecoratorWithClientImportIsAllowed(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/client.ts", `
import { Controller } from '@open-core/framework/client'

@Controller()
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

func TestValidateSourceFiles_ControllerDecoratorWithBothImportsIsRejected(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/ambiguous.ts", `
import { Controller as ServerController } from '@open-core/framework/server'
import { Controller as ClientController } from '@open-core/framework/client'

@Controller()
export class DemoController {}
`)

	issues, err := rb.validateSourceFiles(resourcePath)
	if err != nil {
		t.Fatalf("validateSourceFiles returned error: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if !strings.Contains(issues[0].Message, "ambiguous @Controller") {
		t.Fatalf("unexpected issue message: %s", issues[0].Message)
	}
}

func TestGenerateAutoloadControllers_ControllerDecoratorFollowsImportSide(t *testing.T) {
	resourcePath := t.TempDir()
	rb := NewResourceBuilder(".")

	writeTestFile(t, resourcePath, "src/server.ts", `
import { Controller } from '@open-core/framework/server'

@Controller()
export class ServerController {}
`)

	writeTestFile(t, resourcePath, "src/client.ts", `
import { Controller } from '@open-core/framework/client'

@Controller()
export class ClientController {}
`)

	writeTestFile(t, resourcePath, "src/ignored.ts", `
@Controller()
export class IgnoredController {}
`)

	if err := rb.generateAutoloadControllers(resourcePath); err != nil {
		t.Fatalf("generateAutoloadControllers returned error: %v", err)
	}

	serverContent, err := os.ReadFile(filepath.Join(resourcePath, ".opencore", "autoload.server.controllers.ts"))
	if err != nil {
		t.Fatalf("failed to read server autoload file: %v", err)
	}

	clientContent, err := os.ReadFile(filepath.Join(resourcePath, ".opencore", "autoload.client.controllers.ts"))
	if err != nil {
		t.Fatalf("failed to read client autoload file: %v", err)
	}

	serverText := string(serverContent)
	clientText := string(clientContent)

	if !strings.Contains(serverText, "../src/server") {
		t.Fatalf("expected server autoload to include server controller import, got: %s", serverText)
	}
	if strings.Contains(serverText, "../src/client") || strings.Contains(serverText, "../src/ignored") {
		t.Fatalf("server autoload includes unexpected imports: %s", serverText)
	}

	if !strings.Contains(clientText, "../src/client") {
		t.Fatalf("expected client autoload to include client controller import, got: %s", clientText)
	}
	if strings.Contains(clientText, "../src/server") || strings.Contains(clientText, "../src/ignored") {
		t.Fatalf("client autoload includes unexpected imports: %s", clientText)
	}
}
