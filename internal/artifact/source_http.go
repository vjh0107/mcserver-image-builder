package artifact

import (
	"net/http"
	"path/filepath"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
)

func init() {
	registerJarResolver(resolveURLJar)
	registerDownloadResolver(resolveURLDownload)
}

func resolveURLDownload(src *config.DownloadSource, _ *http.Client) (source, bool) {
	if src.URL == "" {
		return nil, false
	}
	return &urlSource{url: src.URL}, true
}

func resolveURLJar(src *config.ServerSource, client *http.Client) (source, bool) {
	return resolveURLDownload(&src.DownloadSource, client)
}

type urlSource struct {
	url string
}

func (s *urlSource) download(client *http.Client, destPath string, onProgress ProgressFunc) error {
	req, err := http.NewRequest(http.MethodGet, s.url, nil)
	if err != nil {
		return err
	}
	return writeHTTPResponse(req, client, destPath, onProgress)
}

func (s *urlSource) cacheKey() string {
	return cacheKey("url", s.url)
}

func (s *urlSource) describe() string {
	return s.url
}

func (s *urlSource) fileName() (string, error) {
	return filepath.Base(s.url), nil
}
