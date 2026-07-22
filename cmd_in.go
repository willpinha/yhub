package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func InCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:      "in",
		Usage:     "Run a command inside the directory of a repository",
		ArgsUsage: "<repository> <command> [<args>...]",
		// The command and its args must reach the repository untouched, even
		// when they look like flags (ex. yhub in TP go test -v ./...)
		SkipFlagParsing: true,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return errors.New("expected a repository name or alias followed by a command")
			}

			config, err := NewConfig(fs)
			if err != nil {
				return err
			}

			repo, err := config.FindRepository(cmd.Args().First())
			if err != nil {
				return err
			}

			cloned, err := afero.DirExists(fs, repo.Path)
			if err != nil {
				return err
			}

			if !cloned {
				return fmt.Errorf("repository '%s' is not cloned at '%s'", repo.Repository, repo.Path)
			}

			return runCommand(ctx, repo.Path, cmd.Args().Slice()[1:])
		},
	}
}

func runCommand(ctx context.Context, dir string, argv []string) error {
	slog.Debug("running command", "directory", dir, "command", strings.Join(argv, " "))

	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Dir = dir
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("command '%s' failed: %w", strings.Join(argv, " "), err)
	}

	return nil
}
