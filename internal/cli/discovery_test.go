package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.junhyung.kr/mcserver-image-builder/internal/cli"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	writeYAML(t, filepath.Join(root, "servers", "lobby", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")
	writeYAML(t, filepath.Join(root, "servers", "wild", "mcserver.yaml"), "kind: Server\nname: wild\nsource:\n  url: https://example.com/paper.jar\n")
	writeYAML(t, filepath.Join(root, "proxy", "mcserver.yaml"), "kind: Proxy\nname: proxy\nsource:\n  url: https://example.com/velocity.jar\n")

	entries, err := cli.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestDiscover_SkipsConfig(t *testing.T) {
	root := t.TempDir()
	writeYAML(t, filepath.Join(root, "servers", "lobby", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")
	writeYAML(t, filepath.Join(root, "mixins", "shared", "mcserver.yaml"), "kind: Component\nname: shared\n")

	entries, err := cli.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "lobby" {
		t.Errorf("expected 'lobby', got %q", entries[0].Name)
	}
}

func TestDiscover_SkipsDotDirs(t *testing.T) {
	root := t.TempDir()
	writeYAML(t, filepath.Join(root, "lobby", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")
	writeYAML(t, filepath.Join(root, ".warm", "mcserver.yaml"), "kind: Server\nname: hidden\nsource:\n  url: https://example.com/paper.jar\n")

	entries, err := cli.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestDiscover_DuplicateNameError(t *testing.T) {
	root := t.TempDir()
	writeYAML(t, filepath.Join(root, "a", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")
	writeYAML(t, filepath.Join(root, "b", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")

	_, err := cli.Discover(root)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestFindByName(t *testing.T) {
	root := t.TempDir()
	writeYAML(t, filepath.Join(root, "lobby", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")
	writeYAML(t, filepath.Join(root, "wild", "mcserver.yaml"), "kind: Server\nname: wild\nsource:\n  url: https://example.com/paper.jar\n")

	path, err := cli.FindByName(root, "wild")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(path)) != "wild" {
		t.Errorf("expected wild directory, got %s", path)
	}
}

func TestFindByName_NotFound(t *testing.T) {
	root := t.TempDir()
	writeYAML(t, filepath.Join(root, "lobby", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")

	_, err := cli.FindByName(root, "missing")
	if err == nil {
		t.Fatal("expected error for missing server")
	}
}

func TestNames(t *testing.T) {
	root := t.TempDir()
	writeYAML(t, filepath.Join(root, "wild", "mcserver.yaml"), "kind: Server\nname: wild\nsource:\n  url: https://example.com/paper.jar\n")
	writeYAML(t, filepath.Join(root, "lobby", "mcserver.yaml"), "kind: Server\nname: lobby\nsource:\n  url: https://example.com/paper.jar\n")

	names, err := cli.Names(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "lobby" || names[1] != "wild" {
		t.Errorf("expected [lobby wild], got %v", names)
	}
}

func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
