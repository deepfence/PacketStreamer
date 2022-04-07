package config

import (
	"github.com/deepfence/PacketStreamer/pkg/testutils"
	"testing"
)

func TestValidateReceiverConfig(t *testing.T) {
	for _, tt := range []struct {
		TestName      string
		ShouldError   bool
		ExpectedError error
		Config        *Config
	}{
		{
			TestName:      "Errors when no input is configured",
			ShouldError:   true,
			ExpectedError: ErrNoInputConfigured,
			Config:        &Config{},
		},
		{
			TestName:      "Errors when no port is configured for the input",
			ShouldError:   true,
			ExpectedError: ErrNoPortConfiguredForInput,
			Config: &Config{
				Input: &InputConfig{},
			},
		},
	} {
		t.Run(tt.TestName, func(t *testing.T) {
			err := ValidateReceiverConfig(tt.Config)

			if tt.ShouldError && err != nil {
				testutils.ErrorsShouldMatch(t, tt.ExpectedError, err)
			}
		})
	}
}
