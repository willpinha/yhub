package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"github.com/willpinha/yhub/internal/config"
	"github.com/willpinha/yhub/internal/git"
)

func newUncloneCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:  "unclone",
		Usage: "Remove cloned repository folders from disk",
		Commands: []*cli.Command{
			newUncloneAllCommand(fs, g),
			newUncloneGroupCommand(fs, g),
			newUncloneRepoCommand(fs, g),
		},
	}
}

func newUncloneAllCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:  "all",
		Usage: "Remove all cloned repositories",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Usage: "Remove even with uncommitted or unpushed changes"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runUncloneAction(ctx, cmd, fs, g, func(cfg *config.Config) ([]selectedRepo, []string) {
				return selectAll(cfg), nil
			})
		},
	}
}

func newUncloneGroupCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:      "group",
		Usage:     "Remove cloned repositories in the given group(s)",
		ArgsUsage: "<group>...",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Usage: "Remove even with uncommitted or unpushed changes"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				fmt.Fprintln(cmd.Root().ErrWriter, "usage: yhub unclone group <group>...")
				return cli.Exit("", 1)
			}
			return runUncloneAction(ctx, cmd, fs, g, func(cfg *config.Config) ([]selectedRepo, []string) {
				return selectGroups(cfg, args)
			})
		},
	}
}

func newUncloneRepoCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:      "repo",
		Usage:     "Remove the given cloned repositories by name or alias",
		ArgsUsage: "<name-or-alias>...",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Usage: "Remove even with uncommitted or unpushed changes"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				fmt.Fprintln(cmd.Root().ErrWriter, "usage: yhub unclone repo <name-or-alias>...")
				return cli.Exit("", 1)
			}
			return runUncloneAction(ctx, cmd, fs, g, func(cfg *config.Config) ([]selectedRepo, []string) {
				return selectRepos(cfg, args)
			})
		},
	}
}

func runUncloneAction(ctx context.Context, cmd *cli.Command, fs afero.Fs, g git.Git, selector func(*config.Config) ([]selectedRepo, []string)) error {
	cfg, err := config.Load(fs, config.FileName)
	if err != nil {
		return handleLoadError(cmd, err)
	}

	if verr := cfg.Validate(); verr != nil {
		printViolationList(cmd, unwrapJoined(verr))
		return cli.Exit("", 1)
	}

	force := cmd.Bool("force")
	selected, notFound := selector(cfg)
	out := cmd.Root().Writer

	result := runUnclone(ctx, out, fs, g, cfg, selected, force)

	printUncloneSummary(out, result, notFound)

	if result.Failed > 0 || len(notFound) > 0 {
		return cli.Exit("", 1)
	}
	return nil
}

type uncloneResult struct {
	Removed, Skipped, Failed int
	Warnings                 []string
}

func runUnclone(ctx context.Context, out io.Writer, fs afero.Fs, g git.Git, cfg *config.Config, selected []selectedRepo, force bool) uncloneResult {
	var result uncloneResult

	for _, s := range selected {
		dest := repoDest(cfg, s)

		if exists, _ := afero.DirExists(fs, dest); !exists {
			fmt.Fprintf(out, "%s: not cloned, skipping\n", s.Repo.Name)
			result.Skipped++
			continue
		}

		if !force {
			dirty, err := g.HasUncommittedChanges(ctx, dest)
			if err != nil {
				fmt.Fprintf(out, "%s: cannot check status: %v, skipping (use --force to remove)\n", s.Repo.Name, err)
				result.Skipped++
				continue
			}
			if dirty {
				fmt.Fprintf(out, "%s: has uncommitted changes, skipping (use --force to remove)\n", s.Repo.Name)
				result.Skipped++
				continue
			}

			unpushed, err := g.HasUnpushedCommits(ctx, dest)
			if err != nil {
				fmt.Fprintf(out, "%s: cannot check status: %v, skipping (use --force to remove)\n", s.Repo.Name, err)
				result.Skipped++
				continue
			}
			if unpushed {
				fmt.Fprintf(out, "%s: has unpushed commits, skipping (use --force to remove)\n", s.Repo.Name)
				result.Skipped++
				continue
			}
		}

		if err := fs.RemoveAll(dest); err != nil {
			fmt.Fprintf(out, "%s: remove failed: %v\n", s.Repo.Name, err)
			result.Failed++
			continue
		}

		fmt.Fprintf(out, "%s: uncloned\n", s.Repo.Name)
		result.Removed++
	}

	return result
}

func printUncloneSummary(out io.Writer, r uncloneResult, notFound []string) {
	for _, item := range notFound {
		fmt.Fprintf(out, "warning: %q not found\n", item)
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(out, "warning: %s\n", w)
	}
	fmt.Fprintf(out, "Summary: %d uncloned, %d skipped, %d failed\n", r.Removed, r.Skipped, r.Failed)
}
