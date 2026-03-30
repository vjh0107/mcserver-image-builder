package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.junhyung.kr/mcserver-image-builder/internal/config"
	"go.junhyung.kr/mcserver-image-builder/internal/docker"
	"go.junhyung.kr/mcserver-image-builder/internal/ui"
	"go.junhyung.kr/mcserver-image-builder/internal/warm"
)

type warmOptions struct {
	file  string
	all   bool
	force bool
}

func NewWarmCommand() *cobra.Command {
	opts := &warmOptions{}

	cmd := &cobra.Command{
		Use:   "warm [server-name]",
		Short: "Pre-run server to generate plugin remapping and library caches",
		Long: `Pre-run the server in a Docker container to generate caches.

Paper servers remap plugins on first startup and download libraries.
Warm runs the server once, collects these artifacts, and includes them
in the final image so containers start faster.

Examples:
  mcserver warm lobby
  mcserver warm --all
  mcserver warm lobby --force`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.all {
				return runWarmAll(cmd, opts)
			}
			return runWarm(cmd, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "path to mcserver.yaml")
	cmd.Flags().BoolVar(&opts.all, "all", false, "warm all Paper servers in workspace")
	cmd.Flags().BoolVar(&opts.force, "force", false, "force regeneration even if cache exists")

	return cmd
}

func runWarm(cmd *cobra.Command, opts *warmOptions, args []string) error {
	workingDir, err := resolveWorkingDir(cmd)
	if err != nil {
		return err
	}

	configPath, err := resolveConfig(opts.file, args, workingDir)
	if err != nil {
		return err
	}

	cfg, err := config.LoadWithComponents(configPath, workingDir)
	if err != nil {
		return err
	}

	ui.Step("Warming %s", cfg.Name)
	if err := executeWarm(workingDir, cfg, configPath, opts.force, os.Stderr); err != nil {
		return err
	}
	ui.Done("%s", cfg.Name)
	return nil
}

func runWarmAll(cmd *cobra.Command, opts *warmOptions) error {
	workingDir, err := resolveWorkingDir(cmd)
	if err != nil {
		return err
	}

	entries, err := Discover(workingDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		cfg, err := config.LoadWithComponents(entry.FilePath, workingDir)
		if err != nil {
			return err
		}

		if !*cfg.Warm.Enabled {
			ui.Info("Warm disabled, skipping %s", entry.Name)
			continue
		}

		ui.Step("Warming %s", entry.Name)
		if err := executeWarm(workingDir, cfg, entry.FilePath, opts.force, os.Stderr); err != nil {
			return err
		}
	}

	return nil
}

func executeWarm(workingDir string, cfg *config.ServerConfig, configPath string, force bool, output io.Writer) error {
	if err := docker.EnsureAvailable(); err != nil {
		return err
	}
	if !*cfg.Warm.Enabled {
		return fmt.Errorf("warm is not enabled for %s", cfg.Name)
	}

	configDir := filepath.Dir(configPath)
	profile, _ := cfg.Kind.Profile()
	runner := warm.NewRunner(configDir, profile.CacheArtifacts, cfg.WarmArtifacts())

	if runner.IsComplete() && !force {
		ui.Info("Cache already exists, skipping (use --force to regenerate)")
		return nil
	}

	if force {
		if err := runner.Clean(); err != nil {
			return err
		}
	}

	provider, err := newProvider(false)
	if err != nil {
		return err
	}
	jarDesc := provider.ResolveJarDescription(&cfg.Source)

	tmpDir, err := os.MkdirTemp("", "mcserver-warm-dl-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := fetchServerArtifacts(provider, cfg, tmpDir); err != nil {
		return err
	}
	paperJarPath := filepath.Join(tmpDir, "server.jar")

	if err := copyPluginResources(workingDir, configDir, cfg.Plugins, tmpDir, "warm"); err != nil {
		return fmt.Errorf("copying plugin resources for warm: %w", err)
	}

	pluginsDir := filepath.Join(tmpDir, "plugins")
	pluginNames := pluginNameList(cfg.Plugins)
	warmOpts := warm.Options{
		Timeout: cfg.Warm.Timeout,
		Memory:  cfg.Warm.Memory,
		Output:  output,
	}
	if err := runner.Run(jarDesc, pluginNames, paperJarPath, pluginsDir, warmOpts); err != nil {
		return fmt.Errorf("warm failed: %w", err)
	}

	return nil
}
