package main

import (
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func MainCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "yhub",
		Usage: "Manage all your Git repositories through a single, centralized repository",
		Commands: []*cli.Command{
			CloneCommand(fs),
			InCommand(fs),
			ListCommand(fs),
			SearchCommand(fs),
		},
	}
}
