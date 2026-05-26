package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Defaults struct {
	Timezone string `yaml:"timezone" json:"timezone"`
}

type GeneralSchedule struct {
	Cron     string `yaml:"cron" json:"cron"`
	Category string `yaml:"category,omitempty" json:"category,omitempty"`
}

type Quote struct {
	ID         string   `yaml:"id" json:"id"`
	Text       string   `yaml:"text" json:"text"`
	Author     string   `yaml:"author" json:"author"`
	Categories []string `yaml:"categories,omitempty" json:"categories,omitempty"`
	Media      []string `yaml:"media,omitempty" json:"media,omitempty"`
	Schedules  []string `yaml:"schedules,omitempty" json:"schedules,omitempty"`
}

type Config struct {
	Defaults         Defaults          `yaml:"defaults" json:"defaults"`
	GeneralSchedules []GeneralSchedule `yaml:"general_schedules,omitempty" json:"general_schedules,omitempty"`
	Quotes           []Quote           `yaml:"quotes" json:"quotes"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	seen := map[string]bool{}
	for i, q := range cfg.Quotes {
		if q.ID == "" {
			return nil, fmt.Errorf("quote at index %d has empty id", i)
		}
		if seen[q.ID] {
			return nil, fmt.Errorf("duplicate quote id %q", q.ID)
		}
		seen[q.ID] = true
		if q.Text == "" {
			return nil, fmt.Errorf("quote %s has empty text", q.ID)
		}
	}
	return &cfg, nil
}

func (q Quote) HasCategory(cat string) bool {
	if cat == "" {
		return true
	}
	for _, c := range q.Categories {
		if c == cat {
			return true
		}
	}
	return false
}
