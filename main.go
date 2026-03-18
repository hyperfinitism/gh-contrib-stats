package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hyperfinitism/gh-contrib-stats/internal/config"
	"github.com/hyperfinitism/gh-contrib-stats/internal/github"
	"github.com/hyperfinitism/gh-contrib-stats/internal/svg"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (reads stdin if not set)")
	output := flag.String("output", "stats.svg", "output SVG file path")
	flag.Parse()

	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath)
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			fmt.Fprintln(os.Stderr, "Usage: gh-contrib-stats --config <config.yaml> [--output <output.svg>]")
			fmt.Fprintln(os.Stderr, "       cat config.yaml | gh-contrib-stats [--output <output.svg>]")
			os.Exit(1)
		}
		cfg, err = config.Load(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	resolved, err := cfg.Resolve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Fetching contribution stats for %s...\n", resolved.Username)

	stats, err := github.FetchStats(resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching stats: %v\n", err)
		os.Exit(1)
	}

	svgContent := svg.Generate(resolved, stats)

	if err := os.WriteFile(*output, []byte(svgContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing SVG: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "SVG written to %s\n", *output)
}
