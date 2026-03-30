package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.junhyung.kr/mcserver-image-builder/internal/config"
	"go.junhyung.kr/mcserver-image-builder/internal/ui"
)

type validateOptions struct {
	file string
	all  bool
}

func NewValidateCommand() *cobra.Command {
	opts := &validateOptions{}

	cmd := &cobra.Command{
		Use:               "validate [server-name]",
		Short:             "Validate mcserver.yaml configuration",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "path to mcserver.yaml")
	cmd.Flags().BoolVar(&opts.all, "all", false, "validate all servers in workspace")

	return cmd
}

func runValidate(cmd *cobra.Command, opts *validateOptions, args []string) error {
	if opts.all {
		return validateAll(cmd)
	}

	workingDir, err := resolveWorkingDir(cmd)
	if err != nil {
		return err
	}

	configPath, err := resolveConfig(opts.file, args, workingDir)
	if err != nil {
		return err
	}

	_, err = config.LoadWithComponents(configPath, workingDir)
	if err != nil {
		return err
	}

	ui.Done("Valid")
	fmt.Println()
	return nil
}

func validateAll(cmd *cobra.Command) error {
	workingDir, err := resolveWorkingDir(cmd)
	if err != nil {
		return err
	}

	entries, err := Discover(workingDir)
	if err != nil {
		return err
	}

	var errors []error
	for _, entry := range entries {
		if _, err := config.LoadWithComponents(entry.FilePath, workingDir); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", entry.Name, err))
		} else {
			ui.Done("%s", entry.Name)
		}
	}

	fmt.Println()

	if len(errors) > 0 {
		for _, e := range errors {
			ui.Warn("%s", e)
		}
		return fmt.Errorf("%d server(s) have invalid configuration", len(errors))
	}

	return nil
}
