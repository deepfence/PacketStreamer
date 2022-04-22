package config

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/inhies/go-bytesize"
	"io/ioutil"
	"time"

	"github.com/klauspost/compress/s2"
	"gopkg.in/yaml.v3"
)

type PcapMode int

const (
	Allow PcapMode = iota
	Deny
	All
)

const (
	kilobyte = 1024
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

type S3PluginConfig struct {
	Region          string
	Bucket          string
	TotalFileSize   *bytesize.ByteSize `yaml:"totalFileSize,omitempty"`
	UploadChunkSize *bytesize.ByteSize `yaml:"uploadChunkSize,omitempty"`
	UploadTimeout   time.Duration      `yaml:"uploadTimeout,omitempty"`
	CannedACL       string             `yaml:"cannedACL,omitempty"`
}

type PluginsConfig struct {
	S3 *S3PluginConfig
}

type OutputConfig struct {
	File    *FileOutputConfig
	Server  *ServerOutputConfig
	Plugins *PluginsConfig
}

type S3OutputRawConfig struct {
	Bucket          string
	Region          string
	TotalFileSize   *string `yaml:"totalFileSize,omitempty"`
	UploadChunkSize *string `yaml:"uploadChunkSize,omitempty"`
	UploadTimeout   *string `yaml:"uploadTimeout,omitempty"`
	CannedACL       *string `yaml:"cannedACL,omitempty"`
}

type PluginsRawConfig struct {
	S3 *S3OutputRawConfig
}

type OutputRawConfig struct {
	File    *FileOutputConfig
	Server  *ServerOutputConfig
	Plugins *PluginsRawConfig
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

type Config struct {
	Input                  *InputConfig
	Output                 OutputConfig
	TLS                    TLSConfig
	Auth                   AuthConfig
	InputPacketLen         int
	LogFilename            string
	PcapMode               PcapMode
	CapturePorts           []int
	CaptureInterfacesPorts map[string][]int
	IgnorePorts            []int
	SamplingRate           SamplingRateConfig
	MaxEncodedLen          int
	MaxGatherLen           int
	MaxPayloadLen          int
	MaxHeaderLen           int
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

	var s3Config *S3PluginConfig
	if rawConfig.Output.Plugins != nil {
		s3Config, err = NewS3Config(rawConfig.Output.Plugins.S3)
		if err != nil {
			return nil, fmt.Errorf("Could not parse S3 config in %s: %w", configFileName, err)
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
			Plugins: &PluginsConfig{
				S3: s3Config,
			},
		},
		TLS:                    rawConfig.TLS,
		Auth:                   rawConfig.Auth,
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
		MaxEncodedLen: s2.MaxEncodedLen(compressBlockSize * kilobyte),
		MaxGatherLen:  compressBlockSize * kilobyte,
		MaxPayloadLen: s2.MaxEncodedLen(compressBlockSize*kilobyte) + /*hdrData*/ 4 + /*payloadMarker*/ 4,
		MaxHeaderLen:  + /*hdrData*/ 4 + /*payloadMarker*/ 4,
	}

	return config, nil
}

func NewS3Config(rawS3Config *S3OutputRawConfig) (*S3PluginConfig, error) {
	var (
		totalFileSize   *bytesize.ByteSize
		uploadTimeout   time.Duration
		uploadChunkSize *bytesize.ByteSize
		cannedACL       string
		err             error
	)

	if rawS3Config == nil {
		return nil, nil
	}

	if rawS3Config.TotalFileSize != nil {
		t, err := bytesize.Parse(*rawS3Config.TotalFileSize)
		if err != nil {
			return nil, fmt.Errorf("could not parse the totalFileSize field %s: %w", *rawS3Config.TotalFileSize, err)
		}
		totalFileSize = &t
	} else {
		t := 10 * bytesize.MB
		totalFileSize = &t
	}

	if rawS3Config.UploadTimeout != nil {
		uploadTimeout, err = time.ParseDuration(*rawS3Config.UploadTimeout)
		if err != nil {
			return nil, fmt.Errorf("could not parse the uploadTimeout field %s: %w", *rawS3Config.UploadTimeout, err)
		}
	} else {
		uploadTimeout = time.Minute
	}

	if rawS3Config.UploadChunkSize != nil {
		u, err := bytesize.Parse(*rawS3Config.UploadChunkSize)
		if err != nil {
			return nil, fmt.Errorf("could not partse the uploadChunkSize field %s: %w", *rawS3Config.UploadChunkSize, err)
		}
		uploadChunkSize = &u
	} else {
		u := 5 * bytesize.MB
		totalFileSize = &u
	}

	if rawS3Config.CannedACL != nil {
		cannedACL = *rawS3Config.CannedACL
	} else {
		cannedACL = string(types.ObjectCannedACLBucketOwnerFullControl)
	}

	return &S3PluginConfig{
		Bucket:          rawS3Config.Bucket,
		Region:          rawS3Config.Region,
		TotalFileSize:   totalFileSize,
		UploadChunkSize: uploadChunkSize,
		UploadTimeout:   uploadTimeout,
		CannedACL:       cannedACL,
	}, nil
}
