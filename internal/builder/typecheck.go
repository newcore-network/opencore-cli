package builder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

const typecheckConfigFile = "tsconfig.json"

// typecheckErrorPattern matches one diagnostic line of `tsc --pretty false`.
var typecheckErrorPattern = regexp.MustCompile(`(?m)^.*error TS\d+:`)

// TypecheckResult is the outcome of one `tsc --noEmit` run over the project.
type TypecheckResult struct {
	// Output is tsc's diagnostics, trimmed; empty when the project type-checks.
	Output     string
	ErrorCount int
	Duration   time.Duration
}

func (r TypecheckResult) Passed() bool {
	return r.ErrorCount == 0
}

// tscEntryPoint locates the project's own TypeScript, so diagnostics match the version the
// project's editor uses.
func tscEntryPoint(projectPath string) (string, error) {
	tsc := filepath.Join(projectPath, "node_modules", "typescript", "bin", "tsc")
	if _, err := os.Stat(tsc); err != nil {
		return "", fmt.Errorf("build.typecheck requires typescript in the project (%s not found)", tsc)
	}
	return tsc, nil
}

// RunTypecheck type-checks the project through its root tsconfig.json. The error is reserved for
// failing to run tsc at all; type errors are reported through the result.
func RunTypecheck(ctx context.Context, projectPath string) (TypecheckResult, error) {
	start := time.Now()

	tsconfig := filepath.Join(projectPath, typecheckConfigFile)
	if _, err := os.Stat(tsconfig); err != nil {
		return TypecheckResult{}, fmt.Errorf("build.typecheck requires %s: %w", tsconfig, err)
	}
	tsc, err := tscEntryPoint(projectPath)
	if err != nil {
		return TypecheckResult{}, err
	}

	cmd := exec.CommandContext(ctx, "node", tsc, "-p", tsconfig, "--noEmit", "--pretty", "false")
	cmd.Dir = projectPath
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	runErr := cmd.Run()
	result := TypecheckResult{
		Output:   strings.TrimSpace(output.String()),
		Duration: time.Since(start),
	}
	result.ErrorCount = len(typecheckErrorPattern.FindAllString(result.Output, -1))

	var exitErr *exec.ExitError
	switch {
	case runErr == nil:
		return result, nil
	case ctx.Err() != nil:
		return result, ctx.Err()
	case errors.As(runErr, &exitErr) && result.ErrorCount > 0:
		// tsc exits non-zero whenever it reports diagnostics.
		return result, nil
	default:
		return result, fmt.Errorf("tsc failed: %w\n%s", runErr, result.Output)
	}
}

// SetTypecheckWarnOnly makes type errors print as warnings instead of failing the build. Used by
// `opencore dev`, where a type error should not stop the watch loop.
func (b *Builder) SetTypecheckWarnOnly(warnOnly bool) {
	b.typecheckWarnOnly = warnOnly
}

// typecheck runs when build.typecheck is enabled, before anything is bundled: the output
// directory is usually the deploy destination itself, so checking afterwards would already have
// shipped the failing code. It regenerates types first so the check sees current `.gen.ts` files.
func (b *Builder) typecheck(ctx context.Context, tasks []BuildTask, plain bool) error {
	if !b.config.Build.TypecheckEnabled() {
		return nil
	}

	if plain {
		fmt.Println("\nType checking...")
	} else {
		fmt.Printf("\n%s Type checking...\n", ui.Info("→"))
	}

	result, err := RunTypecheck(ctx, b.resourceBuilder.projectPath)
	if err != nil {
		if b.typecheckWarnOnly && ctx.Err() == nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Type check skipped: %v", err)))
			return nil
		}
		return err
	}

	if result.Passed() {
		message := fmt.Sprintf("Type check passed (%s)", result.Duration.Round(time.Millisecond))
		if plain {
			fmt.Println(message)
		} else {
			fmt.Println(ui.Success(message))
		}
		return nil
	}

	fmt.Println(result.Output)
	summary := fmt.Sprintf("type check failed: %d error(s)", result.ErrorCount)
	if b.typecheckWarnOnly {
		fmt.Println(ui.Warning(summary))
		return nil
	}
	return errors.New(summary)
}

// prepareTypes regenerates the types of every resource the tasks touch, then type-checks them.
func (b *Builder) prepareTypes(ctx context.Context, tasks []BuildTask, plain bool) error {
	b.regenerateTypes(tasks)
	return b.typecheck(ctx, tasks, plain)
}

// regenerateTypes runs typegen for every resource the tasks touch, one after another, whether or
// not the build type-checks. Bundling runs it again per task, in parallel, which reuses the
// analysis; by then every resource has its generated file, so each one sees the libraries all
// the others create, whatever order they were generated in.
func (b *Builder) regenerateTypes(tasks []BuildTask) {
	seen := make(map[string]bool, len(tasks))
	for _, task := range tasks {
		if task.Type == TypeViews || task.Type == TypeCopy || seen[task.Path] {
			continue
		}
		seen[task.Path] = true
		b.resourceBuilder.RunTypegen(task.Path)
	}
}
