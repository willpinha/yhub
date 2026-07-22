package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func SearchCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "Search a free-form text for mentions of repository names and aliases",
		ArgsUsage: "<text>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument with the text to search")
			}

			config, err := NewConfig(fs)
			if err != nil {
				return err
			}

			output, err := json.MarshalIndent(config.Search(cmd.Args().First()), "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(output))

			return nil
		},
	}
}
