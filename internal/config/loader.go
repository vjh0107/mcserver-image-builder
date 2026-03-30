package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.junhyung.kr/mcserver-image-builder/internal/schema"
	"gopkg.in/yaml.v3"
)

const ConfigFileName = "mcserver.yaml"

type ConfigHeader struct {
	Kind schema.Kind `yaml:"kind"`
	Name string       `yaml:"name"`
}

func LoadHeader(path string) (*ConfigHeader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var header ConfigHeader
	if err := yaml.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return &header, nil
}

func LoadName(path string) (string, error) {
	header, err := LoadHeader(path)
	if err != nil {
		return "", err
	}
	return header.Name, nil
}

func LoadWithKind(path string, kind schema.Kind) (*ServerConfig, error) {
	cfg, err := decode(path)
	if err != nil {
		return nil, err
	}
	cfg.Kind = kind
	SetDefaults(cfg)
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}

func Load(path string) (*ServerConfig, error) {
	cfg, err := decode(path)
	if err != nil {
		return nil, err
	}

	if cfg.Kind == "" {
		return nil, fmt.Errorf("kind is required (Server, Proxy, or Component)")
	}
	if !cfg.Kind.IsValid() {
		return nil, fmt.Errorf("invalid kind %q (expected Server, Proxy, or Component)", cfg.Kind)
	}

	if cfg.Kind != schema.KindComponent && len(cfg.Components) == 0 {
		SetDefaults(cfg)
		if err := Validate(cfg); err != nil {
			return nil, fmt.Errorf("validating config: %w", err)
		}
	}

	return cfg, nil
}

func LoadWithComponents(path, workingDir string) (*ServerConfig, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	if len(cfg.Components) == 0 || workingDir == "" {
		return cfg, nil
	}

	incFiles, err := resolveComponents(workingDir, cfg.Components)
	if err != nil {
		return nil, err
	}

	for _, incPath := range incFiles {
		incData, err := os.ReadFile(incPath)
		if err != nil {
			return nil, fmt.Errorf("loading component %q: %w", incPath, err)
		}

		incCfg := &ServerConfig{}
		if err := yaml.Unmarshal(incData, incCfg); err != nil {
			return nil, fmt.Errorf("parsing component %q: %w", incPath, err)
		}

		if err := rewriteRelativePaths(incCfg, filepath.Dir(incPath), workingDir); err != nil {
			return nil, fmt.Errorf("rewriting paths in component %q: %w", incPath, err)
		}
		mergeComponent(cfg, incCfg)
	}

	if cfg.Kind != schema.KindComponent {
		SetDefaults(cfg)
		if err := Validate(cfg); err != nil {
			return nil, fmt.Errorf("validating config: %w", err)
		}
	}

	return cfg, nil
}

func decode(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	cfg := &ServerConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}

func resolveComponents(workingDir string, includes []string) ([]string, error) {
	var files []string
	for _, inc := range includes {
		fullPath := filepath.Join(workingDir, inc)
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("component %q: %w", inc, err)
		}

		if !info.IsDir() {
			files = append(files, fullPath)
			continue
		}

		err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if d.Name() == ConfigFileName {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scanning component directory %q: %w", inc, err)
		}
	}
	return files, nil
}

func rewriteRelativePaths(cfg *ServerConfig, componentDir, workingDir string) error {
	rel, err := filepath.Rel(workingDir, componentDir)
	if err != nil {
		return fmt.Errorf("calculating relative path from %s to %s: %w", workingDir, componentDir, err)
	}

	rewrite := func(src string) string {
		if strings.HasPrefix(src, "./") {
			return filepath.Join(rel, src[2:])
		}
		return src
	}

	for i := range cfg.Resources {
		cfg.Resources[i].Source.Path = rewrite(cfg.Resources[i].Source.Path)
	}
	for i := range cfg.Plugins {
		for j := range cfg.Plugins[i].Resources {
			cfg.Plugins[i].Resources[j].Source.Path = rewrite(cfg.Plugins[i].Resources[j].Source.Path)
		}
	}
	return nil
}

func mergeComponent(cfg, inc *ServerConfig) {
	if cfg.Warm == nil && inc.Warm != nil {
		cfg.Warm = inc.Warm
	}
	src := &cfg.Source
	incSrc := &inc.Source
	if src.IsEmpty() && !incSrc.IsEmpty() {
		*src = *incSrc
	}
	cfg.Plugins = append(inc.Plugins, cfg.Plugins...)
	cfg.Resources = append(inc.Resources, cfg.Resources...)
}

