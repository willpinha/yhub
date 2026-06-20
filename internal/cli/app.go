package cli

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func NewApp(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "yhub",
		Usage: "Manage all your Git repositories through a single, centralized repository",
		Commands: []*cli.Command{
			newListCommand(fs),
			newValidateCommand(fs),
		},
	}
}

func newListCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all repositories declared in yhub.toml",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Fprintln(cmd.Root().Writer, "list: not implemented yet")
			return nil
		},
	}
}

func newValidateCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate the yhub.toml configuration file",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Fprintln(cmd.Root().Writer, "validate: not implemented yet")
			return nil
		},
	}
}
