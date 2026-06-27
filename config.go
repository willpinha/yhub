package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"path/filepath"

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

type Repository struct{}

type RepositoriesTree struct {
	Repositories []Repository
	SubDirs      map[string]RepositoriesTree
}

func (rt RepositoriesTree) UnmarshalJSON(data []byte) error {
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

func (rt RepositoriesTree) Walk(fn func(dir string, repo Repository)) {
	rt.walk(repositoriesDir, fn)
}

func (rt RepositoriesTree) walk(baseDir string, fn func(dir string, repo Repository)) {
	for _, r := range rt.Repositories {
		fn(baseDir, r)
	}
	for name, sub := range rt.SubDirs {
		sub.walk(filepath.Join(baseDir, name), fn)
	}
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
	/*
		if repo.Platform == "" && c.DefaultPlatform == "" {
			return fmt.Errorf("repository '%s' has no platform, but no default_platform has been specified", repo.Name)
		}

		if repo.Profile == "" && c.DefaultProfile == "" {
			return fmt.Errorf("repository '%s' has no profile, but no default_profile has been specified", repo.Name)
		}
	*/
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

	for name, p := range DefaultPlatforms {
		if _, ok := config.Platforms[name]; !ok {
			config.Platforms[name] = p
		}
	}

	if err = config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}
