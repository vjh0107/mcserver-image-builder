package cli

import "github.com/spf13/cobra"

var Version = "dev"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "mcserver",
		Short:        "Declarative Docker image builder for Minecraft servers",
		Version:      Version,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().String("working-dir", "", "working directory for resolving resources (default: current directory)")

	cmd.AddCommand(NewBuildCommand())
	cmd.AddCommand(NewWarmCommand())
	cmd.AddCommand(NewValidateCommand())
	cmd.AddCommand(NewInitCommand())

	return cmd
}
