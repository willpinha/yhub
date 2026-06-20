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

			local, localErr := config.LoadLocal(fs, config.LocalFileName)
			if localErr != nil && errors.Is(localErr, config.ErrInvalidTOML) {
				return handleLoadError(cmd, localErr)
			}

			violations := collectViolations(cfg, local, localErr)
			if len(violations) > 0 {
				printViolationList(cmd, violations)
				return cli.Exit("", 1)
			}

			fmt.Fprintln(cmd.Root().Writer, "yhub.toml is valid")
			return nil
		},
	}
}

func collectViolations(cfg *config.Config, local *config.LocalConfig, localErr error) []error {
	var violations []error

	violations = append(violations, unwrapJoined(cfg.Validate())...)

	if localErr != nil && errors.Is(localErr, config.ErrNotFound) {
		if len(cfg.Profiles) > 0 {
			violations = append(violations, fmt.Errorf(
				"yhub.local.toml not found, but %d profile(s) are declared in yhub.toml",
				len(cfg.Profiles),
			))
		}
	} else {
		violations = append(violations, unwrapJoined(config.ValidateProfiles(cfg, local))...)
	}

	return violations
}

func handleLoadError(cmd *cli.Command, err error) error {
	w := cmd.Root().ErrWriter
	switch {
	case errors.Is(err, config.ErrNotFound):
		fmt.Fprintln(w, "yhub.toml not found in the current directory")
	case errors.Is(err, config.ErrInvalidTOML):
		fmt.Fprintf(w, "invalid config: %v\n", err)
	default:
		fmt.Fprintln(w, err)
	}
	return cli.Exit("", 1)
}

func printViolationList(cmd *cli.Command, violations []error) {
	w := cmd.Root().ErrWriter
	fmt.Fprintf(w, "yhub.toml has %d validation error(s):\n", len(violations))
	for _, v := range violations {
		fmt.Fprintf(w, "  - %s\n", v)
	}
}

func unwrapJoined(err error) []error {
	if err == nil {
		return nil
	}
	type joined interface{ Unwrap() []error }
	if j, ok := err.(joined); ok {
		return j.Unwrap()
	}
	return []error{err}
}
