package main

import (
	"context"
	"errors"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func UncloneCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:      "unclone",
		Usage:     "Remove the local directories of cloned repositories",
		ArgsUsage: "<repository>...",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Remove all cloned repositories",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Remove repositories even if they have unsaved work",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")

			switch {
			case all && cmd.Args().Len() != 0:
				return errors.New("--all cannot be combined with repository arguments")
			case !all && cmd.Args().Len() == 0:
				return errors.New("expected at least one repository name or alias")
			}

			config, err := NewConfig(fs)
			if err != nil {
				return err
			}

			if all {
				return config.UncloneAll(ctx, cmd.Bool("force"))
			}

			return config.Unclone(ctx, cmd.Args().Slice(), cmd.Bool("force"))
		},
	}
}
