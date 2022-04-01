package config

import (
	"errors"
)

func ValidateReceiverConfig(config *Config) error {
	if config.Input == nil {
		return errors.New("no input configured")
	}
	if config.Input.Port == nil {
		return errors.New("no port configured for input")
	}

	return nil
}
