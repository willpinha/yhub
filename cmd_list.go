package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func ListCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all repositories that are cloned locally",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 0 {
				return errors.New("expected no arguments")
			}

			config, err := NewConfig(fs)
			if err != nil {
				return err
			}

			results, err := config.ListCloned()
			if err != nil {
				return err
			}

			output, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(output))

			return nil
		},
	}
}
