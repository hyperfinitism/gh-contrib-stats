// SPDX-License-Identifier: Apache-2.0 OR MIT

package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed themes.yaml
var defaultThemesData []byte

// Theme defines the color palette for an SVG card.
type Theme struct {
	Background string            `yaml:"background"`
	Border     string            `yaml:"border"`
	Title      string            `yaml:"title"`
	Text       string            `yaml:"text"`
	Muted      string            `yaml:"muted"`
	Score      string            `yaml:"score"`
	Icons      map[string]string `yaml:"icons"`
}

// IconColor returns the icon color for a contribution type key,
// falling back to Text color if the key is not defined.
func (t *Theme) IconColor(key string) string {
	if c, ok := t.Icons[key]; ok {
		return c
	}
	return t.Text
}

type themesFile struct {
	Themes map[string]Theme `yaml:"themes"`
}

var builtinThemes map[string]Theme

func init() {
	var tf themesFile
	if err := yaml.Unmarshal(defaultThemesData, &tf); err != nil {
		panic(fmt.Sprintf("parsing embedded themes.yaml: %v", err))
	}
	builtinThemes = tf.Themes
}

// LookupTheme returns the named theme from the built-in set.
func LookupTheme(name string) (*Theme, error) {
	t, ok := builtinThemes[name]
	if !ok {
		available := make([]string, 0, len(builtinThemes))
		for k := range builtinThemes {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unknown theme %q (available: %v)", name, available)
	}
	return &t, nil
}
