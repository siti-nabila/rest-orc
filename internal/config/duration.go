package config

import (
	"time"
)

// Duration exposes time.Duration without leaking parsing details to consumers.
type Duration struct {
	time.Duration
}

func newDuration(value time.Duration) Duration {
	return Duration{Duration: value}
}
