package docker

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"go.junhyung.kr/mcserver-image-builder/internal/schema"
)

//go:embed templates/Dockerfile-server.tmpl
var serverDockerfileTemplate string

//go:embed templates/Dockerfile-proxy.tmpl
var proxyDockerfileTemplate string

//go:embed templates/entrypoint-server.sh
var serverEntrypoint string

//go:embed templates/entrypoint-proxy.sh
var proxyEntrypoint string

var dockerfileTemplates = map[string]string{
	"server": serverDockerfileTemplate,
	"proxy":  proxyDockerfileTemplate,
}

var entrypoints = map[string]string{
	"server": serverEntrypoint,
	"proxy":  proxyEntrypoint,
}

type TemplateData struct {
	Kind      schema.Kind
	BaseImage string
	WarmCache bool
	Files     []FileMapping
}

type FileMapping struct {
	Src       string
	MountPath string
}

func RenderDockerfile(data TemplateData, contextDir string) error {
	profile, err := data.Kind.Profile()
	if err != nil {
		return fmt.Errorf("unknown kind %q", data.Kind)
	}

	tmplStr := dockerfileTemplates[profile.DockerTemplate]
	entrypoint := entrypoints[profile.DockerTemplate]

	if err := os.WriteFile(filepath.Join(contextDir, "entrypoint.sh"), []byte(entrypoint), 0o755); err != nil {
		return fmt.Errorf("writing entrypoint.sh: %w", err)
	}

	tmpl, err := template.New("Dockerfile").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing Dockerfile template: %w", err)
	}

	dockerfilePath := filepath.Join(contextDir, "Dockerfile")
	out, err := os.Create(dockerfilePath)
	if err != nil {
		return fmt.Errorf("creating Dockerfile: %w", err)
	}
	defer out.Close()

	if err := tmpl.Execute(out, data); err != nil {
		return fmt.Errorf("rendering Dockerfile: %w", err)
	}

	return nil
}
