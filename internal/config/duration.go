package config

import (
	"fmt"
	"time"

	"github.com/goccy/go-yaml"
)

// Duration decodes human-readable YAML durations while exposing time.Duration
// to application infrastructure.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(data []byte) error {
	var value string
	if err := yaml.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode duration: %w", err)
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", value, err)
	}

	d.Duration = parsed
	return nil
}

func newDuration(value time.Duration) Duration {
	return Duration{Duration: value}
}
