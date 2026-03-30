package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
	"go.junhyung.kr/mcserver-image-builder/internal/ui"
)

//go:embed templates/paper.yaml
var serverTemplate string

//go:embed templates/velocity.yaml
var proxyTemplate string

//go:embed templates/Makefile
var makefileTemplate string

type templateData struct {
	Name string
}

func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <template> <server-names...>",
		Short: "Scaffold a new project structure",
		Long:  "Templates: paper-standalone, paper-proxied",
		Args:  cobra.MinimumNArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return []string{"paper-standalone", "paper-proxied"}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args[0], args[1:])
		},
	}

	return cmd
}

func runInit(cmd *cobra.Command, templateName string, serverNames []string) error {
	workingDir, err := resolveWorkingDir(cmd)
	if err != nil {
		return err
	}

	switch templateName {
	case "paper-standalone":
		return initStandalone(workingDir, serverNames[0])
	case "paper-proxied":
		return initProxied(workingDir, serverNames)
	default:
		return fmt.Errorf("unknown template %q (available: paper-standalone, paper-proxied)", templateName)
	}
}

func initStandalone(workingDir, name string) error {
	dir := filepath.Join(workingDir, name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory %s already exists", name)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := renderTemplate(serverTemplate, filepath.Join(dir, "mcserver.yaml"), templateData{Name: name}); err != nil {
		return err
	}

	ui.Done("Initialized %s", dir)
	return nil
}

func initProxied(workingDir string, serverNames []string) error {
	if err := os.MkdirAll(filepath.Join(workingDir, "resources"), 0o755); err != nil {
		return fmt.Errorf("creating resources directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(workingDir, "proxy"), 0o755); err != nil {
		return fmt.Errorf("creating proxy directory: %w", err)
	}

	if err := renderTemplate(proxyTemplate, filepath.Join(workingDir, "proxy", "mcserver.yaml"), templateData{Name: "proxy"}); err != nil {
		return err
	}

	for _, name := range serverNames {
		dir := filepath.Join(workingDir, "servers", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating server directory %s: %w", name, err)
		}

		if err := renderTemplate(serverTemplate, filepath.Join(dir, "mcserver.yaml"), templateData{Name: name}); err != nil {
			return err
		}
	}

	if err := os.WriteFile(filepath.Join(workingDir, "Makefile"), []byte(makefileTemplate), 0o644); err != nil {
		return fmt.Errorf("creating Makefile: %w", err)
	}

	ui.Done("Initialized proxied project with servers: %v", serverNames)
	return nil
}

func renderTemplate(tmplStr, destPath string, data templateData) error {
	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", destPath, err)
	}
	defer out.Close()

	return tmpl.Execute(out, data)
}
