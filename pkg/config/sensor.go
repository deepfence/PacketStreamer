package config

import (
	"errors"
)

func ValidateSensorConfig(config *Config) error {
	if config.Output.File == nil && config.Output.Server == nil {
		return errors.New("no output configured")
	}
	if config.Output.Server != nil && config.Output.Server.Port == nil {
		return errors.New("no port configured for server output")
	}

	return nil
}
