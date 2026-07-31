package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/spf13/cobra"
)

const (
	packageManagerAuto = "auto"
	channelStable      = "stable"
	channelBeta        = "beta"
)

var findPackageManager = exec.LookPath

var runPackageManager = func(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func packageManagerFromUserAgent(userAgent string) string {
	name := strings.SplitN(strings.TrimSpace(userAgent), "/", 2)[0]
	switch name {
	case "npm", "pnpm", "yarn":
		return name
	default:
		return ""
	}
}

func resolvePackageManager(requested string) (string, error) {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "" {
		requested = packageManagerAuto
	}

	if requested != packageManagerAuto {
		if requested != "npm" && requested != "pnpm" && requested != "yarn" {
			return "", fmt.Errorf("unsupported package manager %q; use npm, pnpm, yarn, or auto", requested)
		}
		if _, err := findPackageManager(requested); err != nil {
			return "", fmt.Errorf("%s is not available on PATH", requested)
		}
		return requested, nil
	}

	preferred := packageManagerFromUserAgent(os.Getenv("npm_config_user_agent"))
	candidates := []string{"npm", "pnpm", "yarn"}
	if preferred != "" {
		candidates = append([]string{preferred}, candidates...)
	}
	for _, manager := range candidates {
		if _, err := findPackageManager(manager); err == nil {
			return manager, nil
		}
	}

	return "", fmt.Errorf("could not find npm, pnpm, or yarn on PATH")
}

func updateArgs(packageManager, channel string) ([]string, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = channelStable
	}

	var tag string
	switch channel {
	case channelStable:
		tag = "latest"
	case channelBeta:
		tag = "beta"
	default:
		return nil, fmt.Errorf("unsupported update channel %q; use stable or beta", channel)
	}

	packageSpec := "@open-core/cli@" + tag
	switch packageManager {
	case "npm":
		return []string{"install", "--global", packageSpec}, nil
	case "pnpm":
		return []string{"add", "--global", packageSpec}, nil
	case "yarn":
		return []string{"global", "add", packageSpec}, nil
	default:
		return nil, fmt.Errorf("unsupported package manager %q", packageManager)
	}
}

func NewUpdateCommand() *cobra.Command {
	var channel string
	var packageManager string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update OpenCore CLI with npm, pnpm, or Yarn",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := resolvePackageManager(packageManager)
			if err != nil {
				return err
			}
			args, err = updateArgs(manager, channel)
			if err != nil {
				return err
			}

			fmt.Println(ui.Info(fmt.Sprintf("Updating OpenCore CLI with %s...", manager)))
			if err := runPackageManager(manager, args); err != nil {
				return fmt.Errorf("failed to update OpenCore CLI with %s: %w", manager, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&channel, "channel", channelStable, "Release channel to install (stable|beta)")
	cmd.Flags().StringVarP(&packageManager, "package-manager", "p", packageManagerAuto, "Package manager to use (auto|npm|pnpm|yarn)")
	return cmd
}
