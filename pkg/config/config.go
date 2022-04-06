package config

import (
	"fmt"
	"io/ioutil"
	"time"

	bytesize "github.com/inhies/go-bytesize"
	"gopkg.in/yaml.v3"
)

type PcapMode int

const (
	Allow PcapMode = iota
	Deny
	All
)

type InputConfig struct {
	Address string
	Port    *int
}

type FileOutputConfig struct {
	Path string
}

type ServerOutputConfig struct {
	Address string
	Port    *int
}

type S3OutputRawConfig struct {
	Bucket        string
	Region        string
	KeyFormat     *string `yaml:"keyFormat,omitempty"`
	TotalFileSize *string `yaml:"totalFileSize,omitempty"`
	UploadTimeout *string `yaml:"uploadTimeout,omitempty"`
	UsePutObject  *bool   `yaml:"usePutObject,omitempty"`
}

type OutputRawConfig struct {
	File   *FileOutputConfig
	Server *ServerOutputConfig
	S3     *S3OutputRawConfig
}

type TLSConfig struct {
	Enable   bool
	CertFile string
	KeyFile  string
}

type AuthConfig struct {
	Enable bool
	Key    string
}

type SamplingRateConfig struct {
	MaxPktsToWrite int
	MaxTotalPkts   int
}

type RawConfig struct {
	Input                  *InputConfig
	Output                 OutputRawConfig
	TLS                    TLSConfig
	Auth                   AuthConfig
	CompressBlockSize      *int             `yaml:"compressBlockSize,omitempty"`
	InputPacketLen         *int             `yaml:"inputPacketLen,omitempty"`
	LogFilename            string           `yaml:"logFilename,omitempty"`
	PcapMode               string           `yaml:"pcapMode,omitempty"`
	CapturePorts           []int            `yaml:"capturePorts,omitempty"`
	CaptureInterfacesPorts map[string][]int `yaml:"captureInterfacesPorts,omitempty"`
	IgnorePorts            []int            `yaml:"ignorePorts,omitempty"`
}

type S3OutputConfig struct {
	Bucket        string
	Region        string
	KeyFormat     string
	TotalFileSize *bytesize.ByteSize
	UploadTimeout time.Duration
}

type OutputConfig struct {
	File   *FileOutputConfig
	Server *ServerOutputConfig
	S3     *S3OutputConfig
}

type Config struct {
	Input                  *InputConfig
	Output                 OutputConfig
	TLS                    TLSConfig
	Auth                   AuthConfig
	CompressBlockSize      int
	InputPacketLen         int
	LogFilename            string
	PcapMode               PcapMode
	CapturePorts           []int
	CaptureInterfacesPorts map[string][]int
	IgnorePorts            []int
	SamplingRate           SamplingRateConfig
}

func NewConfig(configFileName string) (*Config, error) {
	configFile, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, fmt.Errorf("could not read the config file %s: %w", configFileName, err)
	}

	var rawConfig RawConfig
	if err := yaml.Unmarshal(configFile, &rawConfig); err != nil {
		return nil, fmt.Errorf("could not parse the config file %s: %w", configFileName, err)
	}

	var s3 *S3OutputConfig
	if rawConfig.Output.S3 != nil {
		var (
			totalFileSize *bytesize.ByteSize
			uploadTimeout time.Duration
		)

		key := "/packetstreamer/%Y/%m/%d/%H/%M/%S"
		if rawConfig.Output.S3.KeyFormat != nil {
			key = *rawConfig.Output.S3.KeyFormat
		}

		if rawConfig.Output.S3.TotalFileSize != nil {
			t, err := bytesize.Parse(*rawConfig.Output.S3.TotalFileSize)
			if err != nil {
				return nil, fmt.Errorf("could not parse the totalFileSize field %s: %w", *rawConfig.Output.S3.TotalFileSize, err)
			}
			totalFileSize = &t
		} else {
			t := 1 * bytesize.MB
			totalFileSize = &t
		}

		if rawConfig.Output.S3.UploadTimeout != nil {
			uploadTimeout, err = time.ParseDuration(*rawConfig.Output.S3.UploadTimeout)
			if err != nil {
				return nil, fmt.Errorf("could not parse the uploadTimeout field %s: %w", *rawConfig.Output.S3.UploadTimeout, err)
			}
		} else {
			uploadTimeout = time.Minute
		}

		s3 = &S3OutputConfig{
			Bucket:        rawConfig.Output.S3.Bucket,
			Region:        rawConfig.Output.S3.Region,
			KeyFormat:     key,
			TotalFileSize: totalFileSize,
			UploadTimeout: uploadTimeout,
		}
	}

	compressBlockSize := 65
	if rawConfig.CompressBlockSize != nil {
		compressBlockSize = *rawConfig.CompressBlockSize
	}

	inputPacketLen := 65535
	if rawConfig.InputPacketLen != nil {
		inputPacketLen = *rawConfig.InputPacketLen
	}

	var pcapMode PcapMode
	switch rawConfig.PcapMode {
	case "allow":
		pcapMode = Allow
	case "deny":
		pcapMode = Deny
	case "all":
		fallthrough
	case "":
		pcapMode = All
	default:
		return nil, fmt.Errorf("invalid pcapMode \"%s\"", rawConfig.PcapMode)
	}

	config := &Config{
		Input: rawConfig.Input,
		Output: OutputConfig{
			File:   rawConfig.Output.File,
			Server: rawConfig.Output.Server,
			S3:     s3,
		},
		TLS:                    rawConfig.TLS,
		Auth:                   rawConfig.Auth,
		CompressBlockSize:      compressBlockSize,
		InputPacketLen:         inputPacketLen,
		LogFilename:            rawConfig.LogFilename,
		PcapMode:               pcapMode,
		CapturePorts:           rawConfig.CapturePorts,
		CaptureInterfacesPorts: rawConfig.CaptureInterfacesPorts,
		IgnorePorts:            rawConfig.IgnorePorts,
		// TODO(vadorovsky): Make it configurable.
		SamplingRate: SamplingRateConfig{
			MaxPktsToWrite: 1,
			MaxTotalPkts:   1,
		},
	}

	return config, nil
}
