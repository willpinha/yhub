package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/mail"
	"path"
	"slices"
	"strings"

	"github.com/spf13/afero"
)

const configPath = "yhub.json"

var DefaultPlatforms = Platforms{
	"github":    Platform{Host: "github.com"},
	"gitlab":    Platform{Host: "gitlab.com"},
	"bitbucket": Platform{Host: "bitbucket.org"},
}

type Platform struct {
	Host string `json:"host"`
}

type Platforms map[string]Platform

type Profile struct {
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	SSHKey    string `json:"ssh_key"`
}

type Profiles map[string]Profile

type Repository struct {
	Repository string   `json:"repository"`
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases"`
	Platform   string   `json:"platform"`
	Profile    string   `json:"profile"`
}

type Repositories map[string][]Repository

func (r Repositories) All() iter.Seq2[string, Repository] {
	dirs := slices.Sorted(maps.Keys(r))

	return func(yield func(string, Repository) bool) {
		for _, dir := range dirs {
			for _, repo := range r[dir] {
				if !yield(dir, repo) {
					return
				}
			}
		}
	}
}

type Config struct {
	fs              afero.Fs
	Platforms       Platforms    `json:"platforms"`
	DefaultPlatform string       `json:"default_platform"`
	Profiles        Profiles     `json:"profiles"`
	DefaultProfile  string       `json:"default_profile"`
	Repositories    Repositories `json:"repositories"`
}

type configValidateFunc func() error

func (c *Config) validate() error {
	validators := []configValidateFunc{
		c.validatePlatforms,
		c.validateProfiles,
		c.validateRepositories,
	}

	for _, v := range validators {
		if err := v(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) validatePlatforms() error {
	for name, p := range c.Platforms {
		if name == "" {
			return errors.New("platforms cannot have empty names")
		}

		if p.Host == "" {
			return fmt.Errorf("platform '%s' has an empty host", name)
		}
	}

	if c.DefaultPlatform != "" {
		if _, ok := c.Platforms[c.DefaultPlatform]; !ok {
			return fmt.Errorf(
				"default platform '%s' does not exist. Valid platforms are: %v",
				c.DefaultPlatform, slices.Sorted(maps.Keys(c.Platforms)),
			)
		}
	}

	return nil
}

func (c *Config) validateProfiles() error {
	if len(c.Profiles) == 0 {
		return errors.New("profiles cannot be empty")
	}

	isValidEmail := func(s string) bool {
		addr, err := mail.ParseAddress(s)

		return err == nil && addr.Address == s
	}

	for name, p := range c.Profiles {
		if name == "" {
			return errors.New("profiles cannot have empty names")
		}

		switch {
		case p.SSHKey == "":
			return fmt.Errorf("profile '%s' has an empty SSH key", name)
		case p.UserName == "":
			return fmt.Errorf("profile '%s' has an empty user name", name)
		case p.UserEmail == "":
			return fmt.Errorf("profile '%s' has an empty user email", name)
		case !isValidEmail(p.UserEmail):
			return fmt.Errorf("profile '%s' has an invalid email address: %s", name, p.UserEmail)
		}
	}

	if c.DefaultProfile != "" {
		if _, ok := c.Profiles[c.DefaultProfile]; !ok {
			return fmt.Errorf(
				"default profile '%s' does not exist. Valid profiles are: %v",
				c.DefaultProfile, slices.Sorted(maps.Keys(c.Profiles)),
			)
		}
	}

	return nil
}

func (c *Config) validateRepositories() error {
	dirs := slices.Sorted(maps.Keys(c.Repositories))

	for _, dir := range dirs {
		if !isCleanRelativePath(dir) {
			return fmt.Errorf("directory '%s' must be a clean relative path inside the hub", dir)
		}

		if len(c.Repositories[dir]) == 0 {
			return fmt.Errorf("directory '%s' has no repositories", dir)
		}
	}

	identifiers := Set[string]{}
	repositories := Set[string]{}
	destinations := Set[string]{}

	for dir, repo := range c.Repositories.All() {
		if err := c.validateRepository(dir, repo); err != nil {
			return err
		}

		if repositories.Has(repo.Repository) {
			return fmt.Errorf("repository '%s' is defined more than once", repo.Repository)
		}
		repositories.Add(repo.Repository)

		for _, id := range append([]string{repo.Name}, repo.Aliases...) {
			if identifiers.Has(id) {
				return fmt.Errorf("name or alias '%s' is used by more than one repository", id)
			}
			identifiers.Add(id)
		}

		destinations.Add(path.Join(dir, repo.Name))
	}

	for _, dir := range dirs {
		for p := dir; p != "."; p = path.Dir(p) {
			if destinations.Has(p) {
				return fmt.Errorf("directory '%s' is inside the clone destination '%s'", dir, p)
			}
		}
	}

	return nil
}

func (c *Config) validateRepository(dir string, repo Repository) error {
	switch {
	case !isRepositoryPath(repo.Repository):
		return fmt.Errorf(
			"repository '%s' in directory '%s' must have the format '<owner>/<name>'",
			repo.Repository, dir,
		)
	case repo.Name == "" || repo.Name == "." || repo.Name == ".." || strings.Contains(repo.Name, "/"):
		return fmt.Errorf("repository '%s' must have a valid directory name, got '%s'", repo.Repository, repo.Name)
	case slices.Contains(repo.Aliases, ""):
		return fmt.Errorf("repository '%s' has an empty alias", repo.Repository)
	case repo.Platform == "" && c.DefaultPlatform == "":
		return fmt.Errorf("repository '%s' has no platform, and no default_platform has been specified", repo.Repository)
	case repo.Profile == "" && c.DefaultProfile == "":
		return fmt.Errorf("repository '%s' has no profile, and no default_profile has been specified", repo.Repository)
	}

	if repo.Platform != "" {
		if _, ok := c.Platforms[repo.Platform]; !ok {
			return fmt.Errorf("repository '%s' references unknown platform '%s'", repo.Repository, repo.Platform)
		}
	}

	if repo.Profile != "" {
		if _, ok := c.Profiles[repo.Profile]; !ok {
			return fmt.Errorf("repository '%s' references unknown profile '%s'", repo.Repository, repo.Profile)
		}
	}

	return nil
}

func isCleanRelativePath(dir string) bool {
	if dir == "" || path.IsAbs(dir) || path.Clean(dir) != dir {
		return false
	}

	return dir != ".." && !strings.HasPrefix(dir, "../")
}

func isRepositoryPath(repository string) bool {
	segments := strings.Split(repository, "/")

	return len(segments) >= 2 && !slices.Contains(segments, "")
}

func NewConfig(fs afero.Fs) (*Config, error) {
	f, err := fs.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := &Config{fs: fs}

	if err = json.NewDecoder(f).Decode(config); err != nil {
		return nil, err
	}

	if config.Platforms == nil {
		config.Platforms = Platforms{}
	}

	for name, dp := range DefaultPlatforms {
		if _, ok := config.Platforms[name]; !ok {
			config.Platforms[name] = dp
		}
	}

	if err = config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}
