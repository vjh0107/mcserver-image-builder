package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.junhyung.kr/mcserver-image-builder/internal/artifact"
	"go.junhyung.kr/mcserver-image-builder/internal/config"
	"go.junhyung.kr/mcserver-image-builder/internal/docker"
	"go.junhyung.kr/mcserver-image-builder/internal/fsutil"
)

func resolveConfig(file string, args []string, workingDir string) (string, error) {
	if file != "" && len(args) > 0 {
		return "", fmt.Errorf("cannot use both --file and server name argument")
	}
	if file != "" {
		return file, nil
	}
	if len(args) > 0 {
		return FindByName(workingDir, args[0])
	}
	return "", fmt.Errorf("specify a server name or use --file")
}

func resolveTag(tag, registry, serverName string) string {
	if tag != "" {
		return tag
	}
	if registry != "" {
		return fmt.Sprintf("%s/%s:latest", registry, serverName)
	}
	return fmt.Sprintf("%s:latest", serverName)
}

func resolveWorkingDir(cmd *cobra.Command) (string, error) {
	flag := cmd.Flag("working-dir").Value.String()
	if flag != "" {
		return flag, nil
	}
	return os.Getwd()
}

func newProvider(noCache bool) (*artifact.Provider, error) {
	if noCache {
		return artifact.NewProvider(""), nil
	}
	cacheDir, err := resolveCacheDir()
	if err != nil {
		return nil, err
	}
	return artifact.NewProvider(cacheDir), nil
}

func fetchServerArtifacts(provider *artifact.Provider, cfg *config.ServerConfig, destDir string) error {
	if err := provider.FetchServerJar(&cfg.Source, destDir); err != nil {
		return fmt.Errorf("downloading server jar: %w", err)
	}
	if err := provider.FetchPlugins(cfg.Plugins, destDir); err != nil {
		return fmt.Errorf("fetching plugins: %w", err)
	}
	return nil
}

func resolveCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "mcserver-image-builder"), nil
}

func resolveSrcPath(src, workingDir, configDir string) string {
	if strings.HasPrefix(src, "./") {
		return filepath.Join(configDir, src[2:])
	}
	return filepath.Join(workingDir, src)
}

func serverNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	wd, err := resolveWorkingDir(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	names, err := Names(wd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func pluginNameList(plugins []config.Plugin) []string {
	names := make([]string, len(plugins))
	for i, p := range plugins {
		names[i] = p.Name
	}
	return names
}

func toDockerFiles(resources []config.Resource) []docker.FileMapping {
	var result []docker.FileMapping
	for _, f := range resources {
		if !f.ForBuild() {
			continue
		}
		src := f.Source.Path
		if src == "" {
			src = f.MountPath
		}
		result = append(result, docker.FileMapping{
			Src:       src,
			MountPath: f.MountPath,
		})
	}
	return result
}

func copyPluginResources(workingDir, configDir string, plugins []config.Plugin, contextDir, stage string) error {
	for _, p := range plugins {
		for _, f := range p.Resources {
			if stage == "build" && !f.ForBuild() {
				continue
			}
			if stage == "warm" && !f.ForWarm() {
				continue
			}

			src := resolveSrcPath(f.Source.Path, workingDir, configDir)
			dst := filepath.Join(contextDir, f.MountPath)

			if err := fsutil.Copy(src, dst); err != nil {
				return fmt.Errorf("plugin %s resource %s: %w", p.Name, f.Source.Path, err)
			}
		}
	}
	return nil
}
