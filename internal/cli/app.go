package cli

import (
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"github.com/willpinha/yhub/internal/git"
)

func NewApp(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:  "yhub",
		Usage: "Manage all your Git repositories through a single, centralized repository",
		Commands: []*cli.Command{
			newListCommand(fs),
			newValidateCommand(fs),
			newCloneCommand(fs, g),
			newUncloneCommand(fs, g),
		},
	}
}
