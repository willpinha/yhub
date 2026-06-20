package cli

import (
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

func NewApp(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "yhub",
		Usage: "Manage all your Git repositories through a single, centralized repository",
		Commands: []*cli.Command{
			newListCommand(fs),
			newValidateCommand(fs),
		},
	}
}
