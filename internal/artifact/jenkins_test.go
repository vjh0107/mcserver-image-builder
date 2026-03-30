package artifact_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.junhyung.kr/mcserver-image-builder/internal/artifact"
	"go.junhyung.kr/mcserver-image-builder/internal/config"
)

func TestFetchPlugins_JenkinsSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/job/my-plugin/lastSuccessfulBuild/api/json":
			json.NewEncoder(w).Encode(map[string]any{
				"artifacts": []map[string]string{
					{"fileName": "plugin-1.0.jar", "relativePath": "target/plugin-1.0.jar"},
				},
			})
		case "/job/my-plugin/lastSuccessfulBuild/artifact/target/plugin-1.0.jar":
			w.Write([]byte("jar content"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := artifact.NewProvider("")
	contextDir := t.TempDir()

	plugins := []config.Plugin{
		{
			Name: "PluginA",
			Source: config.DownloadSource{
				Jenkins: &config.JenkinsSource{
					URL:      srv.URL,
					Job:      "my-plugin",
					Artifact: "plugin-*.jar",
				},
			},
		},
	}

	if err := c.FetchPlugins(plugins, contextDir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(contextDir, "plugins", "plugin-1.0.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "jar content" {
		t.Errorf("expected 'jar content', got %q", string(data))
	}
}
