package main

import (
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func CloneCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "clone",
		Usage: "",
	}
}
