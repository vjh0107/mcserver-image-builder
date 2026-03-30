package config_test

import (
	"strings"
	"testing"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
	"go.junhyung.kr/mcserver-image-builder/internal/schema"
)

func TestLoad_PaperWithURL(t *testing.T) {
	cfg := mustLoadServer(t, "testdata/valid-paper-with-url.yaml")

	if cfg.Name != "test-server" {
		t.Errorf("expected name 'test-server', got %q", cfg.Name)
	}
	if cfg.Source.URL != "https://example.com/paper.jar" {
		t.Errorf("unexpected server URL: %s", cfg.Source.URL)
	}
	if len(cfg.Plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(cfg.Plugins))
	}
	if cfg.Plugins[1].Extract != true {
		t.Error("expected PluginB to have extract=true")
	}
	if cfg.Plugins[2].Source.Jenkins == nil {
		t.Fatal("expected PluginC to have jenkins config")
	}
}

func TestLoad_PaperWithVersionBuild(t *testing.T) {
	cfg := mustLoadServer(t, "testdata/valid-paper-with-version.yaml")

	if cfg.Source.PaperMC == nil {
		t.Fatal("expected papermc to be set")
	}
	if cfg.Source.PaperMC.Version != "1.21.4" {
		t.Errorf("expected version '1.21.4', got %q", cfg.Source.PaperMC.Version)
	}
	if cfg.Source.PaperMC.Build != 194 {
		t.Errorf("expected build 194, got %d", cfg.Source.PaperMC.Build)
	}
}

func TestLoad_VelocityWithVersionBuild(t *testing.T) {
	cfg := mustLoadProxy(t, "testdata/valid-velocity-with-version.yaml")

	if cfg.Kind != schema.KindProxy {
		t.Errorf("expected kind 'proxy', got %q", cfg.Kind)
	}
	if cfg.Source.PaperMC.Version != "3.4.0" {
		t.Errorf("expected version '3.4.0', got %q", cfg.Source.PaperMC.Version)
	}
}

func TestLoad_MinimalConfig(t *testing.T) {
	cfg := mustLoadServer(t, "testdata/valid-minimal.yaml")

	if cfg.Name != "test-minimal" {
		t.Errorf("expected name 'test-minimal', got %q", cfg.Name)
	}
}

func TestLoad_MissingName(t *testing.T) {
	expectError(t, "testdata/invalid-missing-name.yaml", schema.KindServer, "name is required")
}

func TestLoad_NoServerSource(t *testing.T) {
	expectError(t, "testdata/invalid-no-paper-or-velocity.yaml", schema.KindServer, "one of url, papermc, jenkins, or teamcity is required")
}

func TestLoad_MultipleServerSources(t *testing.T) {
	expectError(t, "testdata/invalid-both-paper-and-velocity.yaml", schema.KindServer, "mutually exclusive")
}

func TestLoad_PaperMCVersionWithoutBuild(t *testing.T) {
	expectError(t, "testdata/invalid-version-without-build.yaml", schema.KindServer, "papermc.build is required")
}

func TestLoad_PluginMissingSource(t *testing.T) {
	expectError(t, "testdata/invalid-plugin-missing-source.yaml", schema.KindServer, "one of url, jenkins, or teamcity is required")
}

func TestLoad_PluginBothSources(t *testing.T) {
	expectError(t, "testdata/invalid-plugin-both-sources.yaml", schema.KindServer, "mutually exclusive")
}

func mustLoadServer(t *testing.T, path string) *config.ServerConfig {
	t.Helper()
	cfg, err := config.LoadWithKind(path, schema.KindServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return cfg
}

func mustLoadProxy(t *testing.T, path string) *config.ServerConfig {
	t.Helper()
	cfg, err := config.LoadWithKind(path, schema.KindProxy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return cfg
}

func expectError(t *testing.T, path string, kind schema.Kind, errSubstring string) {
	t.Helper()
	_, err := config.LoadWithKind(path, kind)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), errSubstring) {
		t.Errorf("expected error containing %q, got: %v", errSubstring, err)
	}
}
