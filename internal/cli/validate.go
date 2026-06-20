package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"github.com/willpinha/yhub/internal/config"
)

func newValidateCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate the yhub.toml configuration file",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := config.Load(fs, config.FileName)
			if err != nil {
				return handleLoadError(cmd, err)
			}

			verr := cfg.Validate()
			if verr != nil {
				printViolations(cmd, verr)
				return cli.Exit("", 1)
			}

			fmt.Fprintln(cmd.Root().Writer, "yhub.toml is valid")
			return nil
		},
	}
}

func handleLoadError(cmd *cli.Command, err error) error {
	w := cmd.Root().ErrWriter
	switch {
	case errors.Is(err, config.ErrNotFound):
		fmt.Fprintln(w, "yhub.toml not found in the current directory")
	case errors.Is(err, config.ErrInvalidTOML):
		fmt.Fprintf(w, "invalid yhub.toml: %v\n", err)
	default:
		fmt.Fprintln(w, err)
	}
	return cli.Exit("", 1)
}

func printViolations(cmd *cli.Command, err error) {
	w := cmd.Root().ErrWriter
	violations := unwrapJoined(err)
	fmt.Fprintf(w, "yhub.toml has %d validation error(s):\n", len(violations))
	for _, v := range violations {
		fmt.Fprintf(w, "  - %s\n", v)
	}
}

func unwrapJoined(err error) []error {
	type joined interface{ Unwrap() []error }
	if j, ok := err.(joined); ok {
		return j.Unwrap()
	}
	return []error{err}
}
