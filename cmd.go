package main

import "github.com/urfave/cli/v3"

func MainCommand() *cli.Command {
	return &cli.Command{
		Name:  "yhub",
		Usage: " Manage all your Git repositories through a single, centralized repository",
		Commands: []*cli.Command{
			CloneCommand(),
			InCommand(),
			InfoCommand(),
		},
	}
}
