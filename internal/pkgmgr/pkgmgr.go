package pkgmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Choice string

const (
	ChoiceAuto Choice = "auto"
	ChoicePnpm Choice = "pnpm"
	ChoiceYarn Choice = "yarn"
	ChoiceNpm  Choice = "npm"
)

type Resolved struct {
	Choice  Choice
	Version string
}

type packageJSON struct {
	PackageManager string `json:"packageManager"`
}

func ParseChoice(s string) (Choice, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ChoiceAuto, nil
	}
	s = strings.TrimPrefix(s, "--")
	s = strings.TrimPrefix(s, "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "")

	switch s {
	case string(ChoiceAuto):
		return ChoiceAuto, nil
	case string(ChoicePnpm):
		return ChoicePnpm, nil
	case string(ChoiceYarn):
		return ChoiceYarn, nil
	case string(ChoiceNpm):
		return ChoiceNpm, nil
	default:
		return "", fmt.Errorf("invalid package manager: %s", s)
	}
}

func PreferenceFromEnv() Choice {
	v := strings.TrimSpace(os.Getenv("OPENCORE_PACKAGE_MANAGER"))
	c, err := ParseChoice(v)
	if err != nil {
		return ChoiceAuto
	}
	return c
}

// PreferenceFromProject tries to infer the preferred package manager from the current project.
// It does NOT validate if the binary exists.
func PreferenceFromProject(projectRoot string) Choice {
	choice, _ := PreferenceFromProjectStrict(projectRoot)
	return choice
}

// PreferenceFromProjectStrict infers the project preference and reports a
// malformed package.json or packageManager declaration instead of hiding it.
func PreferenceFromProjectStrict(projectRoot string) (Choice, error) {
	// 1) package.json "packageManager" field (highest priority)
	if projectRoot == "" {
		projectRoot = "."
	}
	packagePath := filepath.Join(projectRoot, "package.json")
	if b, err := os.ReadFile(packagePath); err == nil {
		var pkg packageJSON
		if err := json.Unmarshal(b, &pkg); err != nil {
			return ChoiceAuto, fmt.Errorf("invalid %s: %w", packagePath, err)
		}
		if strings.TrimSpace(pkg.PackageManager) != "" {
			if c, err := parsePackageManagerField(pkg.PackageManager); err == nil {
				return c, nil
			} else {
				return ChoiceAuto, fmt.Errorf("invalid packageManager in %s: %w", packagePath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return ChoiceAuto, fmt.Errorf("failed to read %s: %w", packagePath, err)
	}

	// 2) Lockfiles
	if fileExists(filepath.Join(projectRoot, "pnpm-lock.yaml")) {
		return ChoicePnpm, nil
	}
	if fileExists(filepath.Join(projectRoot, "yarn.lock")) || fileExists(filepath.Join(projectRoot, ".yarnrc.yml")) {
		return ChoiceYarn, nil
	}
	if fileExists(filepath.Join(projectRoot, "package-lock.json")) {
		return ChoiceNpm, nil
	}

	return ChoiceAuto, nil
}

// EffectivePreference applies env override (if not auto) and otherwise falls back to project inference.
func EffectivePreference(projectRoot string) Choice {
	env := PreferenceFromEnv()
	if env != "" && env != ChoiceAuto {
		return env
	}
	proj := PreferenceFromProject(projectRoot)
	if proj != "" {
		return proj
	}
	return ChoiceAuto
}

func Resolve(preference Choice) (Resolved, error) {
	if preference == "" {
		preference = ChoiceAuto
	}
	if preference == ChoiceAuto {
		if v, ok := detectVersion("pnpm"); ok {
			return Resolved{Choice: ChoicePnpm, Version: v}, nil
		}
		if v, ok := detectYarnModernVersion(); ok {
			return Resolved{Choice: ChoiceYarn, Version: v}, nil
		}
		if v, ok := detectVersion("npm"); ok {
			return Resolved{Choice: ChoiceNpm, Version: v}, nil
		}
		return Resolved{}, fmt.Errorf("no supported package manager found (pnpm, yarn>=2, npm)")
	}

	switch preference {
	case ChoicePnpm:
		if v, ok := detectVersion("pnpm"); ok {
			return Resolved{Choice: ChoicePnpm, Version: v}, nil
		}
		return Resolved{}, fmt.Errorf("pnpm not found")
	case ChoiceYarn:
		if v, ok := detectYarnModernVersion(); ok {
			return Resolved{Choice: ChoiceYarn, Version: v}, nil
		}
		if v, ok := detectVersion("yarn"); ok {
			return Resolved{}, fmt.Errorf("yarn v1 is not supported (found %s); please use yarn berry (v2+)", v)
		}
		return Resolved{}, fmt.Errorf("yarn not found")
	case ChoiceNpm:
		if v, ok := detectVersion("npm"); ok {
			return Resolved{Choice: ChoiceNpm, Version: v}, nil
		}
		return Resolved{}, fmt.Errorf("npm not found")
	default:
		return Resolved{}, fmt.Errorf("invalid package manager: %s", preference)
	}
}

func (r Resolved) InstallCmd() string {
	switch r.Choice {
	case ChoicePnpm:
		return "pnpm install"
	case ChoiceYarn:
		return "yarn install"
	case ChoiceNpm:
		return "npm install"
	default:
		return ""
	}
}

func (r Resolved) AddDevCmd(pkgs ...string) string {
	args := strings.Join(pkgs, " ")
	switch r.Choice {
	case ChoicePnpm:
		return strings.TrimSpace("pnpm add -D " + args)
	case ChoiceYarn:
		return strings.TrimSpace("yarn add -D " + args)
	case ChoiceNpm:
		return strings.TrimSpace("npm install -D " + args)
	default:
		return ""
	}
}

func (r Resolved) AddCmd(pkgs ...string) string {
	args := strings.Join(pkgs, " ")
	switch r.Choice {
	case ChoicePnpm:
		return strings.TrimSpace("pnpm add " + args)
	case ChoiceYarn:
		return strings.TrimSpace("yarn add " + args)
	case ChoiceNpm:
		return strings.TrimSpace("npm install " + args)
	default:
		return ""
	}
}

func (r Resolved) ExecCmd(bin string, args ...string) string {
	rest := strings.TrimSpace(strings.Join(args, " "))
	if rest != "" {
		rest = " " + rest
	}
	switch r.Choice {
	case ChoicePnpm:
		return "pnpm dlx " + bin + rest
	case ChoiceYarn:
		return "yarn dlx " + bin + rest
	case ChoiceNpm:
		return "npm exec -- " + bin + rest
	default:
		return ""
	}
}

func detectVersion(bin string) (string, bool) {
	cmd := exec.Command(bin, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", false
	}
	return v, true
}

func detectYarnModernVersion() (string, bool) {
	v, ok := detectVersion("yarn")
	if !ok {
		return "", false
	}
	major, ok := parseMajor(v)
	if !ok {
		return "", false
	}
	if major < 2 {
		return "", false
	}
	return v, true
}

func parseMajor(version string) (int, bool) {
	version = strings.TrimSpace(version)
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return 0, false
	}
	majorStr := strings.TrimSpace(parts[0])
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return 0, false
	}
	return major, true
}

func choiceFromPackageManagerField(field string) (Choice, bool) {
	choice, err := parsePackageManagerField(field)
	return choice, err == nil
}

func parsePackageManagerField(field string) (Choice, error) {
	f := strings.TrimSpace(strings.ToLower(field))
	parts := strings.SplitN(f, "@", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("expected <npm|pnpm|yarn>@<version>, got %q", field)
	}
	choice, err := ParseChoice(parts[0])
	if err != nil || choice == ChoiceAuto {
		return "", fmt.Errorf("unsupported package manager %q", parts[0])
	}
	if _, _, _, err := parseVersion(parts[1]); err != nil {
		return "", err
	}
	if choice == ChoiceYarn {
		major, _, _, _ := parseVersion(parts[1])
		if major < 2 {
			return "", fmt.Errorf("yarn v1 is not supported; use Yarn Berry (v2+)")
		}
	}
	return choice, nil
}

func parseVersion(version string) (int, int, int, error) {
	version = strings.SplitN(strings.TrimSpace(strings.TrimPrefix(version, "v")), "+", 2)[0]
	version = strings.SplitN(version, "-", 2)[0]
	parts := strings.Split(version, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return 0, 0, 0, fmt.Errorf("invalid package manager version %q", version)
	}
	values := [3]int{}
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return 0, 0, 0, fmt.Errorf("invalid package manager version %q", version)
		}
		values[i] = value
	}
	return values[0], values[1], values[2], nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
