package artifact

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"go.junhyung.kr/mcserver-image-builder/internal/fsutil"
)

func cacheKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func (c *Provider) cachedDownload(cacheKeyStr, destPath string, downloadFn func(dest string) error) error {
	if c.cacheDir == "" {
		return downloadFn(destPath)
	}

	cachedPath := filepath.Join(c.cacheDir, cacheKeyStr)

	if _, err := os.Stat(cachedPath); err == nil {
		return fsutil.CopyFile(cachedPath, destPath)
	}

	if err := os.MkdirAll(c.cacheDir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	if err := downloadFn(cachedPath); err != nil {
		os.Remove(cachedPath)
		return err
	}

	return fsutil.CopyFile(cachedPath, destPath)
}

