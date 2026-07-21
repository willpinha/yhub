package main

import (
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func InCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "in",
		Usage: "",
	}
}
