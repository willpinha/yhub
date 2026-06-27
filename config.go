package main

import (
	"bytes"
	"encoding/json"
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
	Platforms       Platforms        `json:"platforms"`
	DefaultPlatform string           `json:"default_platform"`
	Profiles        Profiles         `json:"profiles"`
	DefaultProfile  string           `json:"default_profile"`
	Repositories    RepositoriesTree `json:"repositories"`
}

func (c *Config) validate() error {
	return nil
}

func NewConfig(fs afero.Fs) (*Config, error) {
	f, err := fs.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := &Config{}

	if err = json.NewDecoder(f).Decode(config); err != nil {
		return nil, err
	}

	if err = config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}
