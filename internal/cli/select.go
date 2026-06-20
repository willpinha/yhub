package cli

import (
	"sort"

	"github.com/willpinha/yhub/internal/config"
)

type selectedRepo struct {
	Group string
	Repo  config.Repository
}

func selectAll(cfg *config.Config) []selectedRepo {
	groups := make([]string, 0, len(cfg.Groups))
	for name := range cfg.Groups {
		groups = append(groups, name)
	}
	sort.Strings(groups)

	var result []selectedRepo
	for _, group := range groups {
		for _, repo := range cfg.Groups[group] {
			result = append(result, selectedRepo{Group: group, Repo: repo})
		}
	}
	return result
}

func selectGroups(cfg *config.Config, names []string) (selected []selectedRepo, notFound []string) {
	for _, name := range names {
		repos, ok := cfg.Groups[name]
		if !ok {
			notFound = append(notFound, name)
			continue
		}
		for _, repo := range repos {
			selected = append(selected, selectedRepo{Group: name, Repo: repo})
		}
	}
	return
}

func findRepo(cfg *config.Config, groups []string, pred func(config.Repository) bool) *selectedRepo {
	for _, group := range groups {
		for _, repo := range cfg.Groups[group] {
			if pred(repo) {
				r := selectedRepo{Group: group, Repo: repo}
				return &r
			}
		}
	}
	return nil
}

func selectRepos(cfg *config.Config, idents []string) (selected []selectedRepo, notFound []string) {
	groups := make([]string, 0, len(cfg.Groups))
	for name := range cfg.Groups {
		groups = append(groups, name)
	}
	sort.Strings(groups)

	seen := make(map[string]bool)

	for _, ident := range idents {
		match := findRepo(cfg, groups, func(r config.Repository) bool { return r.Alias == ident })
		if match == nil {
			match = findRepo(cfg, groups, func(r config.Repository) bool { return r.Name == ident })
		}
		if match == nil {
			notFound = append(notFound, ident)
			continue
		}
		if !seen[match.Repo.Name] {
			seen[match.Repo.Name] = true
			selected = append(selected, *match)
		}
	}
	return
}
