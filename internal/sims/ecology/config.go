package ecology

import "strconv"

// Config controls the Ecology simulation dimensions.
type Config struct {
	Width  int
	Height int
}

// DefaultConfig returns the standard configuration.
func DefaultConfig() Config {
	return Config{Width: 256, Height: 256}
}

// FromMap populates the config from a string map (flag-style key/value pairs).
func FromMap(cfg map[string]string) Config {
	c := DefaultConfig()
	if cfg == nil {
		return c
	}
	if v, ok := cfg["w"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			c.Width = parsed
		}
	}
	if v, ok := cfg["h"]; ok {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			c.Height = parsed
		}
	}
	return c
}
