// SPDX-License-Identifier: Apache-2.0 OR MIT

package config

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type SelectConfig struct {
	PR         *bool `yaml:"pr"`
	Commit     *bool `yaml:"commit"`
	Issue      *bool `yaml:"issue"`
	Review     *bool `yaml:"review"`
	Discussion *bool `yaml:"discussion"`
}

type WeightConfig struct {
	PR         *uint `yaml:"pr"`
	Commit     *uint `yaml:"commit"`
	Issue      *uint `yaml:"issue"`
	Review     *uint `yaml:"review"`
	Discussion *uint `yaml:"discussion"`
}

type ShowConfig struct {
	PR         *bool `yaml:"pr"`
	Commit     *bool `yaml:"commit"`
	Issue      *bool `yaml:"issue"`
	Review     *bool `yaml:"review"`
	Discussion *bool `yaml:"discussion"`
	TopRepo    *uint `yaml:"top-repo"`
}

type Config struct {
	Username      string       `yaml:"username"`
	Token         string       `yaml:"token"`
	IncludeOwned  *bool        `yaml:"include-owned"`
	ExcludeOwners []string     `yaml:"exclude-owners"`
	ExcludeRepos  []string     `yaml:"exclude-repos"`
	Since         string       `yaml:"since"`
	Until         string       `yaml:"until"`
	Theme         string       `yaml:"theme"`
	Select        SelectConfig `yaml:"select"`
	Weight        WeightConfig `yaml:"weight"`
	Show          ShowConfig   `yaml:"show"`
}

func boolDefault(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

func uintDefault(p *uint, def uint) uint {
	if p != nil {
		return *p
	}
	return def
}

// Resolved configs with defaults applied.

type ResolvedSelect struct {
	PR, Commit, Issue, Review, Discussion bool
}

type ResolvedWeight struct {
	PR, Commit, Issue, Review, Discussion uint
}

type ResolvedShow struct {
	PR, Commit, Issue, Review, Discussion bool
	TopRepo                               uint
}

type ResolvedConfig struct {
	Username      string
	Token         string
	IncludeOwned  bool
	ExcludeOwners map[string]bool
	ExcludeRepos  map[string]bool
	Since         time.Time
	Until         time.Time
	Theme         *Theme
	Select        ResolvedSelect
	Weight        ResolvedWeight
	Show          ResolvedShow
}

func (c *Config) Resolve() (*ResolvedConfig, error) {
	token := c.Token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("token is required: set 'token' in config or GITHUB_TOKEN env var")
	}
	if c.Username == "" {
		return nil, fmt.Errorf("username is required")
	}

	// Parse date range.
	now := time.Now().UTC()
	until := now
	if c.Until != "" {
		t, err := time.Parse("2006-01-02", c.Until)
		if err != nil {
			return nil, fmt.Errorf("invalid 'until' date %q (expected YYYY-MM-DD): %w", c.Until, err)
		}
		// Set to end of day.
		until = t.Add(24*time.Hour - time.Second)
	}
	since := until.AddDate(-1, 0, 0)
	if c.Since != "" {
		t, err := time.Parse("2006-01-02", c.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid 'since' date %q (expected YYYY-MM-DD): %w", c.Since, err)
		}
		since = t
	}
	if !since.Before(until) {
		return nil, fmt.Errorf("'since' (%s) must be before 'until' (%s)", c.Since, c.Until)
	}

	themeName := c.Theme
	if themeName == "" {
		themeName = "dark"
	}
	theme, err := LookupTheme(themeName)
	if err != nil {
		return nil, err
	}

	topRepo := uintDefault(c.Show.TopRepo, 3)
	if topRepo > 100 {
		return nil, fmt.Errorf("show.top-repo must be between 0 and 100, got %d", topRepo)
	}

	excludeOwners := make(map[string]bool, len(c.ExcludeOwners))
	for _, o := range c.ExcludeOwners {
		excludeOwners[strings.ToLower(o)] = true
	}
	excludeRepos := make(map[string]bool, len(c.ExcludeRepos))
	for _, r := range c.ExcludeRepos {
		excludeRepos[strings.ToLower(r)] = true
	}

	return &ResolvedConfig{
		Username:      c.Username,
		Token:         token,
		IncludeOwned:  boolDefault(c.IncludeOwned, false),
		ExcludeOwners: excludeOwners,
		ExcludeRepos:  excludeRepos,
		Since:         since,
		Until:         until,
		Theme:         theme,
		Select: ResolvedSelect{
			PR:         boolDefault(c.Select.PR, true),
			Commit:     boolDefault(c.Select.Commit, true),
			Issue:      boolDefault(c.Select.Issue, true),
			Review:     boolDefault(c.Select.Review, false),
			Discussion: boolDefault(c.Select.Discussion, false),
		},
		Weight: ResolvedWeight{
			PR:         uintDefault(c.Weight.PR, 1),
			Commit:     uintDefault(c.Weight.Commit, 1),
			Issue:      uintDefault(c.Weight.Issue, 1),
			Review:     uintDefault(c.Weight.Review, 1),
			Discussion: uintDefault(c.Weight.Discussion, 1),
		},
		Show: ResolvedShow{
			PR:         boolDefault(c.Show.PR, true),
			Commit:     boolDefault(c.Show.Commit, true),
			Issue:      boolDefault(c.Show.Issue, true),
			Review:     boolDefault(c.Show.Review, false),
			Discussion: boolDefault(c.Show.Discussion, false),
			TopRepo:    topRepo,
		},
	}, nil
}

func Load(r io.Reader) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func LoadFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}
