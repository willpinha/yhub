package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"github.com/willpinha/yhub/internal/config"
	"github.com/willpinha/yhub/internal/git"
)

func newCloneCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:  "clone",
		Usage: "Clone repositories declared in yhub.toml",
		Commands: []*cli.Command{
			newCloneAllCommand(fs, g),
			newCloneGroupCommand(fs, g),
			newCloneRepoCommand(fs, g),
		},
	}
}

func newCloneAllCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:  "all",
		Usage: "Clone all repositories",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runCloneAction(ctx, cmd, fs, g, func(cfg *config.Config) ([]selectedRepo, []string) {
				return selectAll(cfg), nil
			})
		},
	}
}

func newCloneGroupCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:      "group",
		Usage:     "Clone repositories in the given group(s)",
		ArgsUsage: "<group>...",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				fmt.Fprintln(cmd.Root().ErrWriter, "usage: yhub clone group <group>...")
				return cli.Exit("", 1)
			}
			return runCloneAction(ctx, cmd, fs, g, func(cfg *config.Config) ([]selectedRepo, []string) {
				return selectGroups(cfg, args)
			})
		},
	}
}

func newCloneRepoCommand(fs afero.Fs, g git.Git) *cli.Command {
	return &cli.Command{
		Name:      "repo",
		Usage:     "Clone the given repositories by name or alias",
		ArgsUsage: "<name-or-alias>...",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				fmt.Fprintln(cmd.Root().ErrWriter, "usage: yhub clone repo <name-or-alias>...")
				return cli.Exit("", 1)
			}
			return runCloneAction(ctx, cmd, fs, g, func(cfg *config.Config) ([]selectedRepo, []string) {
				return selectRepos(cfg, args)
			})
		},
	}
}

func runCloneAction(ctx context.Context, cmd *cli.Command, fs afero.Fs, g git.Git, selector func(*config.Config) ([]selectedRepo, []string)) error {
	cfg, err := config.Load(fs, config.FileName)
	if err != nil {
		return handleLoadError(cmd, err)
	}

	local, localErr := config.LoadLocal(fs, config.LocalFileName)
	if localErr != nil && errors.Is(localErr, config.ErrInvalidTOML) {
		return handleLoadError(cmd, localErr)
	}

	violations := collectViolations(cfg, local, localErr)
	if len(violations) > 0 {
		printViolationList(cmd, violations)
		return cli.Exit("", 1)
	}

	selected, notFound := selector(cfg)
	out := cmd.Root().Writer

	result := runClone(ctx, out, fs, g, cfg, local, selected)

	orphans, _ := findOrphans(fs, cfg)

	printCloneSummary(out, result, orphans, notFound)

	if result.Failed > 0 || len(notFound) > 0 {
		return cli.Exit("", 1)
	}
	return nil
}

type cloneResult struct {
	Cloned, Skipped, Failed int
	Warnings                []string
}

func runClone(ctx context.Context, out io.Writer, fs afero.Fs, g git.Git, cfg *config.Config, local *config.LocalConfig, selected []selectedRepo) cloneResult {
	var result cloneResult

	for _, s := range selected {
		dest := repoDest(cfg, s)

		if exists, _ := afero.DirExists(fs, dest); exists {
			fmt.Fprintf(out, "%s: already cloned, skipping\n", s.Repo.Name)
			result.Skipped++
			continue
		}

		url, err := cfg.CloneURL(s.Repo)
		if err != nil {
			fmt.Fprintf(out, "%s: %v\n", s.Repo.Name, err)
			result.Failed++
			continue
		}

		env := buildSSHEnv(cfg, local, s)

		if err := g.Clone(ctx, url, dest, env); err != nil {
			fmt.Fprintf(out, "%s: clone failed: %v\n", s.Repo.Name, err)
			result.Failed++
			continue
		}

		fmt.Fprintf(out, "%s: cloned\n", s.Repo.Name)
		result.Cloned++

		if s.Repo.Profile != "" && local != nil {
			if p, ok := local.Profiles[s.Repo.Profile]; ok {
				if p.Name != "" {
					if err := g.SetConfig(ctx, dest, "user.name", p.Name); err != nil {
						result.Warnings = append(result.Warnings, fmt.Sprintf("%s: set user.name: %v", s.Repo.Name, err))
					}
				}
				if p.Email != "" {
					if err := g.SetConfig(ctx, dest, "user.email", p.Email); err != nil {
						result.Warnings = append(result.Warnings, fmt.Sprintf("%s: set user.email: %v", s.Repo.Name, err))
					}
				}
				if cfg.ResolveProtocol(s.Repo) == "ssh" && p.SSHKey != "" {
					if err := g.SetConfig(ctx, dest, "core.sshCommand", sshCommand(p.SSHKey)); err != nil {
						result.Warnings = append(result.Warnings, fmt.Sprintf("%s: set core.sshCommand: %v", s.Repo.Name, err))
					}
				}
			}
		}
	}

	return result
}

func sshCommand(key string) string {
	return "ssh -i " + expandHome(key) + " -o IdentitiesOnly=yes"
}

func buildSSHEnv(cfg *config.Config, local *config.LocalConfig, s selectedRepo) []string {
	if cfg.ResolveProtocol(s.Repo) != "ssh" {
		return nil
	}
	if s.Repo.Profile == "" || local == nil {
		return nil
	}
	p, ok := local.Profiles[s.Repo.Profile]
	if !ok || p.SSHKey == "" {
		return nil
	}
	return []string{"GIT_SSH_COMMAND=" + sshCommand(p.SSHKey)}
}

func repoDest(cfg *config.Config, s selectedRepo) string {
	return filepath.Join(cfg.RepositoriesDir, s.Group, s.Repo.Name)
}

func expandHome(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

func findOrphans(fs afero.Fs, cfg *config.Config) ([]string, error) {
	exists, err := afero.DirExists(fs, cfg.RepositoriesDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	declared := make(map[string]map[string]bool)
	for group, repos := range cfg.Groups {
		declared[group] = make(map[string]bool)
		for _, repo := range repos {
			declared[group][repo.Name] = true
		}
	}

	groupEntries, err := afero.ReadDir(fs, cfg.RepositoriesDir)
	if err != nil {
		return nil, err
	}

	var orphans []string
	for _, groupEntry := range groupEntries {
		if !groupEntry.IsDir() {
			continue
		}
		groupName := groupEntry.Name()
		groupPath := filepath.Join(cfg.RepositoriesDir, groupName)

		nameEntries, err := afero.ReadDir(fs, groupPath)
		if err != nil {
			continue
		}

		for _, nameEntry := range nameEntries {
			if !nameEntry.IsDir() {
				continue
			}
			repoName := nameEntry.Name()
			if declared[groupName] == nil || !declared[groupName][repoName] {
				orphans = append(orphans, filepath.Join(groupName, repoName))
			}
		}
	}

	sort.Strings(orphans)
	return orphans, nil
}

func printCloneSummary(out io.Writer, r cloneResult, orphans []string, notFound []string) {
	for _, item := range notFound {
		fmt.Fprintf(out, "warning: %q not found\n", item)
	}
	for _, path := range orphans {
		fmt.Fprintf(out, "warning: %s is not declared in yhub.toml\n", path)
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(out, "warning: %s\n", w)
	}
	fmt.Fprintf(out, "Summary: %d cloned, %d skipped, %d failed\n", r.Cloned, r.Skipped, r.Failed)
}
