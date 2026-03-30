package artifact_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.junhyung.kr/mcserver-image-builder/internal/artifact"
	"go.junhyung.kr/mcserver-image-builder/internal/config"
)

func TestFetchPlugins_ExtractPlugin(t *testing.T) {
	tgz := createTestTGZ(t, map[string]string{
		"PluginA.jar":        "plugin-a content",
		"PluginA/config.yml": "config: true",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tgz)
	}))
	defer srv.Close()

	c := artifact.NewProvider("")
	contextDir := t.TempDir()

	plugins := []config.Plugin{
		{Name: "PluginA", Source: config.DownloadSource{URL: srv.URL + "/plugin-a.tgz"}, Extract: true},
	}

	if err := c.FetchPlugins(plugins, contextDir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(contextDir, "plugins", "PluginA.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "plugin-a content" {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func createTestTGZ(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	tw.Close()
	gz.Close()
	return buf.Bytes()
}
