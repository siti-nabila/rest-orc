package config

import (
	"time"

	"github.com/goccy/go-yaml"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

// Duration decodes human-readable YAML durations while exposing time.Duration
// to application infrastructure.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(data []byte) error {
	var value string
	if err := yaml.Unmarshal(data, &value); err != nil {
		return dictionary.DecodeDuration(err)
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return dictionary.ParseDuration(value, err)
	}

	d.Duration = parsed
	return nil
}

func newDuration(value time.Duration) Duration {
	return Duration{Duration: value}
}
