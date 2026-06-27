package main

import "github.com/urfave/cli/v3"

func InfoCommand() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "",
	}
}
