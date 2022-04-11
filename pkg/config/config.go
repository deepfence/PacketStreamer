package config

import (
	"fmt"
	"io/ioutil"

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

type OutputConfig struct {
	File   *FileOutputConfig
	Server *ServerOutputConfig
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
	Output                 OutputConfig
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
		Input:                  rawConfig.Input,
		Output:                 rawConfig.Output,
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
		MaxGatherLen: compressBlockSize * kilobyte,
		MaxPayloadLen: s2.MaxEncodedLen(compressBlockSize * kilobyte) + /*hdrData*/4 + /*payloadMarker*/4,
		MaxHeaderLen: + /*hdrData*/4 + /*payloadMarker*/4,
	}

	return config, nil
}
