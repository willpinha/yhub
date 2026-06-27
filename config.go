package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"path/filepath"
	"sort"

	"github.com/spf13/afero"
)

const (
	repositoriesDir = "repositories"
	configPath      = "yhub.json"
)

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
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	Alias    string `json:"alias"`
	Platform string `json:"platform"`
	Profile  string `json:"profile"`
}

func (r Repository) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

type RepositoriesTree struct {
	Repositories []Repository
	SubDirs      map[string]RepositoriesTree
}

func (rt *RepositoriesTree) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	tok, err := dec.Token()
	if err != nil {
		return err
	}

	if tok == json.Delim('[') {
		return json.Unmarshal(data, &rt.Repositories)
	}

	return json.Unmarshal(data, &rt.SubDirs)
}

func (rt RepositoriesTree) Walk(fn func(dir string, repo Repository) error) error {
	return rt.walk(repositoriesDir, fn)
}

func (rt RepositoriesTree) walk(baseDir string, fn func(dir string, repo Repository) error) error {
	for _, r := range rt.Repositories {
		if err := fn(baseDir, r); err != nil {
			return err
		}
	}

	subDirs := make([]string, 0, len(rt.SubDirs))
	for name := range rt.SubDirs {
		subDirs = append(subDirs, name)
	}
	sort.Strings(subDirs)

	for _, sd := range subDirs {
		if err := rt.SubDirs[sd].walk(filepath.Join(baseDir, sd), fn); err != nil {
			return err
		}
	}

	return nil
}

type Config struct {
	fs              afero.Fs
	Platforms       Platforms        `json:"platforms"`
	DefaultPlatform string           `json:"default_platform"`
	Profiles        Profiles         `json:"profiles"`
	DefaultProfile  string           `json:"default_profile"`
	Repositories    RepositoriesTree `json:"repositories"`
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
	defaultPlatformFound := false

	for name, p := range c.Platforms {
		if name == "" {
			return errors.New("platforms cannot have empty names")
		}

		if p.Host == "" {
			return fmt.Errorf("platform '%s' has an empty host", name)
		}

		if name == c.DefaultPlatform {
			defaultPlatformFound = true
		}
	}
	if len(c.Profiles) == 0 {
		return errors.New("profiles cannot be empty")
	}
	if !defaultPlatformFound {
		return fmt.Errorf("default platform '%s' does not exist. Valid platforms are: %v", c.DefaultPlatform)
	}

	return nil
}

func (c *Config) validateProfiles() error {
	if len(c.Profiles) == 0 {
		return errors.New("profiles cannot be empty")
	}

	defaultProfileFound := false

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
			return fmt.Errorf("profile '%s' has an empty SSH key")
		case p.UserName == "":
			return fmt.Errorf("profile '%s' has an empty user name")
		case p.UserEmail == "":
			return fmt.Errorf("profile '%s' has an empty user email")
		case !isValidEmail(p.UserEmail):
			return fmt.Errorf("profile '%s' has an invalid email address: %s", name, p.UserEmail)
		}
	}

	if !defaultProfileFound {
		return fmt.Errorf("default profile '%s' does not exist. Valid profiles are: %v", c.DefaultProfile)
	}

	return nil
}

func (c *Config) validateRepositories() error {
	aliases := Set[string]{}
	fullnames := Set[string]{}

	if err := c.Repositories.Walk(func(dir string, repo Repository) error {
		fullname := repo.FullName()

		switch {
		case repo.Owner == "":
		case repo.Name == "":
		case repo.Alias == "":
		case repo.Platform == "" && c.DefaultPlatform == "":
			return fmt.Errorf("repository '%s' has no platform, and no default_platform has been specified", fullname)
		case repo.Profile == "" && c.DefaultProfile == "":
			return fmt.Errorf("repository '%s' has no profile, and no default_profile has been specified", fullname)
		case aliases.Has(repo.Alias):
			return fmt.Errorf("repository alias '%s' is defined more than once", repo.Alias)
		case fullnames.Has(fullname):
			return fmt.Errorf("repository '%s' is defined more than once", fullname)
		}

		aliases.Add(repo.Alias)
		fullnames.Add(fullname)

		return nil
	}); err != nil {
		return err
	}

	return nil
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
