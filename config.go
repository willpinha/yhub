package main

import (
	"encoding/json"

	"github.com/spf13/afero"
)

const configPath = "yhub.json"

var DefaultPlatforms = Platforms{
	"github":    Platform{Host: "github.com"},
	"gitlab":    Platform{Host: "gitlab.com"},
	"bitbucket": Platform{Host: "bitbucket.com"},
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

type Config struct {
	DefaultProfile string   `json:"default_profile"`
	Profiles       Profiles `json:"profiles"`
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
