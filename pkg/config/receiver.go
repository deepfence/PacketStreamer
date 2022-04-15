package config

import (
	"errors"
)

var (
	ErrNoInputConfigured        = errors.New("no input configured")
	ErrNoPortConfiguredForInput = errors.New("no port configured for input")
)

func ValidateReceiverConfig(config *Config) error {
	if config.Input == nil {
		return ErrNoInputConfigured
	}
	if config.Input.Port == nil {
		return ErrNoPortConfiguredForInput
	}

	return nil
}
