package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"github.com/willpinha/yhub/internal/config"
)

func newListCommand(fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all repositories declared in yhub.toml",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := config.Load(fs, config.FileName)
			if err != nil {
				return handleLoadError(cmd, err)
			}
			renderList(cmd.Root().Writer, cfg)
			return nil
		},
	}
}

func renderList(w io.Writer, cfg *config.Config) {
	if len(cfg.Groups) == 0 {
		fmt.Fprintln(w, "no repositories declared in yhub.toml")
		return
	}

	groups := make([]string, 0, len(cfg.Groups))
	for name := range cfg.Groups {
		groups = append(groups, name)
	}
	sort.Strings(groups)

	for i, group := range groups {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, group)

		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ALIAS\tNAME\tREPOSITORY\tPROFILE")
		for _, repo := range cfg.Groups[group] {
			profile := repo.Profile
			if profile == "" {
				profile = "-"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", repo.Alias, repo.Name, repo.Repository, profile)
		}
		tw.Flush()
	}
}
