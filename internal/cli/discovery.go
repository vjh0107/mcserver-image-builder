package cli

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
	"go.junhyung.kr/mcserver-image-builder/internal/schema"
)

type ServerEntry struct {
	Name     string
	Kind     schema.Kind
	FilePath string
}

func Discover(rootDir string) ([]ServerEntry, error) {
	var entries []ServerEntry
	seen := map[string]string{}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if d.Name() != config.ConfigFileName {
			return nil
		}

		header, err := config.LoadHeader(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		if header.Kind == schema.KindComponent || header.Kind == "" || header.Name == "" {
			return nil
		}

		if existing, ok := seen[header.Name]; ok {
			return fmt.Errorf("duplicate name %q: %s and %s", header.Name, existing, path)
		}

		seen[header.Name] = path
		entries = append(entries, ServerEntry{Name: header.Name, Kind: header.Kind, FilePath: path})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return entries, nil
}

func FindByName(rootDir, name string) (string, error) {
	entries, err := Discover(rootDir)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if e.Name == name {
			return e.FilePath, nil
		}
	}

	return "", fmt.Errorf("%q not found", name)
}

func Names(rootDir string) ([]string, error) {
	entries, err := Discover(rootDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	sort.Strings(names)
	return names, nil
}
