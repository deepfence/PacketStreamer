package config

import (
	"github.com/deepfence/PacketStreamer/pkg/testutils"
	"testing"
)

func TestValidateSensorConfig(t *testing.T) {
	for _, tt := range []struct {
		TestName      string
		ShouldError   bool
		ExpectedError error
		Config        *Config
	}{
		{
			TestName:      "Errors when no output is defined",
			ShouldError:   true,
			ExpectedError: ErrNoOutputConfigured,
			Config: &Config{
				Output: OutputConfig{
					File:   nil,
					Server: nil,
				},
			},
		},
		{
			TestName:      "Errors when mo port is configured for server output",
			ShouldError:   true,
			ExpectedError: ErrNoPortConfiguredForServerOutput,
			Config: &Config{
				Output: OutputConfig{
					Server: &ServerOutputConfig{
						Port: nil,
					},
				},
			},
		},
	} {
		t.Run(tt.TestName, func(t *testing.T) {
			err := ValidateSensorConfig(tt.Config)

			if tt.ShouldError && err != nil {
				testutils.ErrorsShouldMatch(t, tt.ExpectedError, err)
			}
		})
	}
}
