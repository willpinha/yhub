package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
)

var (
	ErrEmptyField             = errors.New("empty required field")
	ErrUnknownProfile         = errors.New("unknown profile")
	ErrDuplicateAlias         = errors.New("duplicate alias")
	ErrDuplicateName          = errors.New("duplicate name")
	ErrBadRepositoryFormat    = errors.New("bad repository format")
	ErrInvalidProtocol        = errors.New("invalid protocol")
	ErrUnknownPlatform        = errors.New("unknown platform")
	ErrEmptyPlatformHost      = errors.New("empty platform host")
	ErrMissingLocalProfile    = errors.New("missing local profile")
	ErrUndeclaredLocalProfile = errors.New("undeclared local profile")
)

var repositoryRe = regexp.MustCompile(`^[^/\s]+/[^/\s]+$`)

type violation struct {
	sentinel error
	msg      string
}

func (v *violation) Error() string { return v.msg }
func (v *violation) Unwrap() error { return v.sentinel }

func newViolation(sentinel error, format string, args ...any) error {
	return &violation{
		sentinel: sentinel,
		msg:      fmt.Sprintf(format, args...),
	}
}

func sortedGroupNames(c *Config) []string {
	names := make([]string, 0, len(c.Groups))
	for g := range c.Groups {
		names = append(names, g)
	}
	sort.Strings(names)
	return names
}

type Validator func(c *Config) []error

func validateRequiredFields(c *Config) []error {
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Repository == "" {
				errs = append(errs, newViolation(
					ErrEmptyField,
					`group %q repo %q: field "repository" is empty`,
					group, repo.Alias,
				))
			}
			if repo.Name == "" {
				errs = append(errs, newViolation(
					ErrEmptyField,
					`group %q repo %q: field "name" is empty`,
					group, repo.Alias,
				))
			}
			if repo.Alias == "" {
				errs = append(errs, newViolation(
					ErrEmptyField,
					`group %q: field "alias" is empty`,
					group,
				))
			}
		}
	}
	return errs
}

func validateProfiles(c *Config) []error {
	profileSet := make(map[string]bool, len(c.Profiles))
	for _, p := range c.Profiles {
		profileSet[p] = true
	}
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Profile != "" && !profileSet[repo.Profile] {
				errs = append(errs, newViolation(
					ErrUnknownProfile,
					`group %q repo %q: profile %q is not declared in profiles`,
					group, repo.Alias, repo.Profile,
				))
			}
		}
	}
	return errs
}

func validateUniqueAliases(c *Config) []error {
	seen := make(map[string]string)
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Alias == "" {
				continue
			}
			loc := fmt.Sprintf("group %q", group)
			if prev, ok := seen[repo.Alias]; ok {
				errs = append(errs, newViolation(
					ErrDuplicateAlias,
					`alias %q is used in both %s and %s`,
					repo.Alias, prev, loc,
				))
			} else {
				seen[repo.Alias] = loc
			}
		}
	}
	return errs
}

func validateUniqueNames(c *Config) []error {
	seen := make(map[string]string)
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Name == "" {
				continue
			}
			loc := fmt.Sprintf("group %q", group)
			if prev, ok := seen[repo.Name]; ok {
				errs = append(errs, newViolation(
					ErrDuplicateName,
					`name %q is used in both %s and %s`,
					repo.Name, prev, loc,
				))
			} else {
				seen[repo.Name] = loc
			}
		}
	}
	return errs
}

func validateRepositoryFormat(c *Config) []error {
	var errs []error
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Repository != "" && !repositoryRe.MatchString(repo.Repository) {
				errs = append(errs, newViolation(
					ErrBadRepositoryFormat,
					`group %q repo %q: repository %q does not match "owner/repo" format`,
					group, repo.Alias, repo.Repository,
				))
			}
		}
	}
	return errs
}

func validateProtocols(c *Config) []error {
	var errs []error
	if c.DefaultProtocol != "" && c.DefaultProtocol != "https" && c.DefaultProtocol != "ssh" {
		errs = append(errs, newViolation(
			ErrInvalidProtocol,
			`default_protocol %q is not one of "https" or "ssh"`,
			c.DefaultProtocol,
		))
	}
	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Protocol != "" && repo.Protocol != "https" && repo.Protocol != "ssh" {
				errs = append(errs, newViolation(
					ErrInvalidProtocol,
					`group %q repo %q: protocol %q is not one of "https" or "ssh"`,
					group, repo.Alias, repo.Protocol,
				))
			}
		}
	}
	return errs
}

func validatePlatforms(c *Config) []error {
	var errs []error

	platformKeys := make([]string, 0, len(c.Platforms))
	for k := range c.Platforms {
		platformKeys = append(platformKeys, k)
	}
	sort.Strings(platformKeys)

	for _, name := range platformKeys {
		if c.Platforms[name].Host == "" {
			errs = append(errs, newViolation(
				ErrEmptyPlatformHost,
				`platform %q has an empty host`,
				name,
			))
		}
	}

	known := func(name string) bool {
		if _, ok := builtinPlatforms[name]; ok {
			return true
		}
		_, ok := c.Platforms[name]
		return ok
	}

	if c.DefaultPlatform != "" && !known(c.DefaultPlatform) {
		errs = append(errs, newViolation(
			ErrUnknownPlatform,
			`default_platform %q is not a known platform`,
			c.DefaultPlatform,
		))
	}

	for _, group := range sortedGroupNames(c) {
		for _, repo := range c.Groups[group] {
			if repo.Platform != "" && !known(repo.Platform) {
				errs = append(errs, newViolation(
					ErrUnknownPlatform,
					`group %q repo %q: platform %q is not a known platform`,
					group, repo.Alias, repo.Platform,
				))
			}
		}
	}

	return errs
}

func (c *Config) Validate() error {
	validators := []Validator{
		validateRequiredFields,
		validateProfiles,
		validateUniqueAliases,
		validateUniqueNames,
		validateRepositoryFormat,
		validateProtocols,
		validatePlatforms,
	}
	var violations []error
	for _, v := range validators {
		violations = append(violations, v(c)...)
	}
	return errors.Join(violations...)
}

func ValidateProfiles(c *Config, local *LocalConfig) error {
	declared := make(map[string]bool, len(c.Profiles))
	for _, name := range c.Profiles {
		declared[name] = true
	}

	var localProfiles map[string]Profile
	if local != nil {
		localProfiles = local.Profiles
	}

	var errs []error

	declaredNames := make([]string, 0, len(declared))
	for name := range declared {
		declaredNames = append(declaredNames, name)
	}
	sort.Strings(declaredNames)

	for _, name := range declaredNames {
		if _, ok := localProfiles[name]; !ok {
			errs = append(errs, newViolation(
				ErrMissingLocalProfile,
				`profile %q is declared in yhub.toml but missing in yhub.local.toml`,
				name,
			))
		}
	}

	localKeys := make([]string, 0, len(localProfiles))
	for k := range localProfiles {
		localKeys = append(localKeys, k)
	}
	sort.Strings(localKeys)

	for _, name := range localKeys {
		if !declared[name] {
			errs = append(errs, newViolation(
				ErrUndeclaredLocalProfile,
				`profile %q is defined in yhub.local.toml but not declared in yhub.toml`,
				name,
			))
		}
	}

	return errors.Join(errs...)
}
