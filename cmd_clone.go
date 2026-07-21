package main

import (
	"context"
	"errors"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func CloneCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:      "clone",
		Usage:     "Clone repositories using the SSH protocol",
		ArgsUsage: "<repository>...",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Clone all repositories",
			},
			&cli.StringFlag{
				Name:  "dir",
				Usage: "Clone all repositories inside a configured directory",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")
			dir := cmd.String("dir")

			switch {
			case all && dir != "":
				return errors.New("--all cannot be combined with --dir")
			case all && cmd.Args().Len() != 0:
				return errors.New("--all cannot be combined with repository arguments")
			case dir != "" && cmd.Args().Len() != 0:
				return errors.New("--dir cannot be combined with repository arguments")
			case !all && dir == "" && cmd.Args().Len() == 0:
				return errors.New("expected at least one repository name or alias")
			}

			config, err := NewConfig(fs)
			if err != nil {
				return err
			}

			switch {
			case all:
				return config.CloneAll(ctx)
			case dir != "":
				return config.CloneDir(ctx, dir)
			}

			return config.Clone(ctx, cmd.Args().Slice())
		},
	}
}
