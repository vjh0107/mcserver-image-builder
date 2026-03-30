package docker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.junhyung.kr/mcserver-image-builder/internal/docker"
	"go.junhyung.kr/mcserver-image-builder/internal/schema"
)

func TestRenderDockerfile_Server(t *testing.T) {
	data := docker.TemplateData{
		Kind:      schema.KindServer,
		BaseImage: "mc-base:latest",
		WarmCache: false,
		Files: []docker.FileMapping{
			{Src: "server.properties", MountPath: "server.properties"},
			{Src: ".embedded-world", MountPath: ".embedded-world"},
		},
	}

	result := renderToString(t, data)

	assertContains(t, result, "FROM mc-base:latest")
	assertContains(t, result, "COPY server.jar ./")
	assertContains(t, result, "COPY .embedded-world .embedded-world")
	assertContains(t, result, "COPY server.properties server.properties")
	assertContains(t, result, "COPY entrypoint.sh /entrypoint.sh")
	assertContains(t, result, "ENTRYPOINT")
	assertNotContains(t, result, "COPY libraries/")
}

func TestRenderDockerfile_ServerWithWarmCache(t *testing.T) {
	data := docker.TemplateData{
		Kind:      schema.KindServer,
		BaseImage: "mc-base:latest",
		WarmCache: true,
	}

	result := renderToString(t, data)

	assertContains(t, result, "COPY libraries/ libraries/")
	assertContains(t, result, "COPY cache/ cache/")
	assertContains(t, result, "COPY versions/ versions/")
}

func TestRenderDockerfile_Proxy(t *testing.T) {
	data := docker.TemplateData{
		Kind:      schema.KindProxy,
		BaseImage: "mc-base:latest",
		Files: []docker.FileMapping{
			{Src: "forwarding-secret.txt", MountPath: "forwarding-secret.txt"},
		},
	}

	result := renderToString(t, data)

	assertContains(t, result, "FROM mc-base:latest")
	assertContains(t, result, "COPY server.jar ./")
	assertContains(t, result, "COPY forwarding-secret.txt forwarding-secret.txt")
	assertContains(t, result, "COPY entrypoint.sh /entrypoint.sh")
	assertNotContains(t, result, "eula")
	assertNotContains(t, result, ".embedded-world")
	assertNotContains(t, result, "COPY libraries/")
}

func TestRenderDockerfile_EntrypointWritten(t *testing.T) {
	dir := t.TempDir()
	data := docker.TemplateData{Kind: schema.KindServer, BaseImage: "mc-base:latest"}

	if err := docker.RenderDockerfile(data, dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "entrypoint.sh"))
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(content), "#!/bin/sh")
}

func renderToString(t *testing.T, data docker.TemplateData) string {
	t.Helper()
	dir := t.TempDir()
	if err := docker.RenderDockerfile(data, dir); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", substr, s)
	}
}
