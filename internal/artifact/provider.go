package artifact

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
	"go.junhyung.kr/mcserver-image-builder/internal/fsutil"
)

type Provider struct {
	client   *http.Client
	cacheDir string
}

func NewProvider(cacheDir string) *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
		cacheDir: cacheDir,
	}
}

func (c *Provider) FetchServerJar(src *config.ServerSource, contextDir string) error {
	return c.FetchServerJarWithProgress(src, contextDir, nil)
}

func (c *Provider) FetchServerJarWithProgress(src *config.ServerSource, contextDir string, onProgress ProgressFunc) error {
	dest := filepath.Join(contextDir, "server.jar")
	s := c.resolveJarSource(src)
	return c.downloadSourceWithProgress(s, dest, onProgress)
}

func (c *Provider) FetchPlugins(plugins []config.Plugin, contextDir string) error {
	pluginsDir := filepath.Join(contextDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("creating plugins directory: %w", err)
	}
	for _, plugin := range plugins {
		if err := c.fetchPlugin(plugin, pluginsDir); err != nil {
			return fmt.Errorf("fetching plugin %s: %w", plugin.Name, err)
		}
	}
	return nil
}

func (c *Provider) FetchPlugin(plugin config.Plugin, contextDir string) error {
	return c.FetchPluginWithProgress(plugin, contextDir, nil)
}

func (c *Provider) FetchPluginWithProgress(plugin config.Plugin, contextDir string, onProgress ProgressFunc) error {
	pluginsDir := filepath.Join(contextDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("creating plugins directory: %w", err)
	}
	return c.fetchPluginWithProgress(plugin, pluginsDir, onProgress)
}

func (c *Provider) fetchPlugin(plugin config.Plugin, pluginsDir string) error {
	return c.fetchPluginWithProgress(plugin, pluginsDir, nil)
}

func (c *Provider) fetchPluginWithProgress(plugin config.Plugin, pluginsDir string, onProgress ProgressFunc) error {
	s := c.resolveDownloadSource(&plugin.Source)

	if plugin.Extract {
		return c.downloadAndExtract(s, plugin.Name, pluginsDir)
	}

	fn, ok := s.(fileNamer)
	if !ok {
		return fmt.Errorf("source does not support file name resolution")
	}
	name, err := fn.fileName()
	if err != nil {
		return err
	}
	return c.downloadSourceWithProgress(s, filepath.Join(pluginsDir, name), onProgress)
}

func (c *Provider) FetchDownload(src *config.DownloadSource, destPath string, extract bool) error {
	return c.FetchDownloadWithProgress(src, destPath, extract, nil)
}

func (c *Provider) FetchDownloadWithProgress(src *config.DownloadSource, destPath string, extract bool, onProgress ProgressFunc) error {
	s := c.resolveDownloadSource(src)
	if !extract {
		return c.downloadSourceWithProgress(s, destPath, onProgress)
	}
	if err := os.MkdirAll(destPath, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", destPath, err)
	}
	return c.downloadAndExtract(s, "resource", destPath)
}

func (c *Provider) Download(url, destPath string) error {
	return c.downloadSource(&urlSource{url: url}, destPath)
}

func (c *Provider) ResolveJarDescription(src *config.ServerSource) string {
	s := c.resolveJarSource(src)
	if d, ok := s.(describer); ok {
		return d.describe()
	}
	return "unknown"
}

func (c *Provider) resolveJarSource(src *config.ServerSource) source {
	for _, r := range jarResolvers {
		if s, ok := r(src, c.client); ok {
			return s
		}
	}
	return &urlSource{}
}

func (c *Provider) resolveDownloadSource(src *config.DownloadSource) source {
	for _, r := range downloadResolvers {
		if s, ok := r(src, c.client); ok {
			return s
		}
	}
	return &urlSource{}
}

func (c *Provider) downloadAndExtract(s source, name, destDir string) error {
	tmpFile := filepath.Join(destDir, name+".tgz")

	if err := c.downloadSource(s, tmpFile); err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := fsutil.ExtractTGZ(tmpFile, destDir); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	return nil
}
