package ui

import (
	"os"
	"strings"

	"golang.org/x/term"
)

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func IsCI() bool {
	if envTrue("CI") || envTrue("GITHUB_ACTIONS") {
		return true
	}

	ciVars := []string{
		"BUILD_ID",
		"BUILD_NUMBER",
		"TEAMCITY_VERSION",
		"JENKINS_URL",
		"GITLAB_CI",
		"BUILDKITE",
		"TF_BUILD",
	}

	for _, key := range ciVars {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}

	return false
}

func IsNonInteractiveSession() bool {
	return !IsTTY() || IsCI() || strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb")
}

func IsUpdateCheckDisabled() bool {
	return envTrue("OPENCORE_DISABLE_UPDATE_CHECK")
}

func envTrue(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
