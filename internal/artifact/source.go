package artifact

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
)

type ProgressFunc func(received, total int64)

type source interface {
	download(client *http.Client, destPath string, onProgress ProgressFunc) error
	cacheKey() string
}

type describer interface {
	describe() string
}

type fileNamer interface {
	fileName() (string, error)
}

type jarResolverFunc func(src *config.ServerSource, client *http.Client) (source, bool)
type downloadResolverFunc func(src *config.DownloadSource, client *http.Client) (source, bool)

var (
	jarResolvers      []jarResolverFunc
	downloadResolvers []downloadResolverFunc
)

func registerJarResolver(r jarResolverFunc) {
	jarResolvers = append(jarResolvers, r)
}

func registerDownloadResolver(r downloadResolverFunc) {
	downloadResolvers = append(downloadResolvers, r)
}

func (c *Provider) downloadSource(src source, destPath string) error {
	return c.downloadSourceWithProgress(src, destPath, nil)
}

func (c *Provider) downloadSourceWithProgress(src source, destPath string, onProgress ProgressFunc) error {
	key := src.cacheKey()
	return c.cachedDownload(key, destPath, func(dest string) error {
		return src.download(c.client, dest, onProgress)
	})
}

type progressReader struct {
	r          io.Reader
	received   int64
	total      int64
	onProgress ProgressFunc
	lastReport time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.received += int64(n)
	if pr.onProgress != nil && time.Since(pr.lastReport) > 100*time.Millisecond {
		pr.onProgress(pr.received, pr.total)
		pr.lastReport = time.Now()
	}
	return n, err
}

func writeHTTPResponse(req *http.Request, client *http.Client, destPath string, onProgress ProgressFunc) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", req.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, req.URL)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	var reader io.Reader = resp.Body
	if onProgress != nil && resp.ContentLength > 0 {
		reader = &progressReader{r: resp.Body, total: resp.ContentLength, onProgress: onProgress}
	}

	_, err = io.Copy(out, reader)
	return err
}
