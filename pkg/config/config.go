package config

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/inhies/go-bytesize"

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

const (
	defaultClientId = "packetstreamer"
	defaultTopic    = "packetstreamer"
	defaultAcks     = "all"
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

type KafkaPluginConfig struct {
	Brokers     []string
	ClientId    string             `yaml:"clientId,omitempty"`
	Topic       string             `yaml:"topic,omitempty"`
	MessageSize *bytesize.ByteSize `yaml:"messageSize,omitempty"`
	Acks        string             `yaml:"acks,omitempty"`
	FileSize    *bytesize.ByteSize `yaml:"fileSize,omitempty"`
	Timeout     time.Duration      `yaml:"timeout,omitempty"`
}

type PluginsConfig struct {
	S3    *S3PluginConfig
	Kafka *KafkaPluginConfig
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

type KafkaOutputRawConfig struct {
	Brokers     string
	ClientId    *string       `yaml:"clientId,omitempty"`
	Topic       *string       `yaml:"topic,omitempty"`
	MessageSize *string       `yaml:"messageSize,omitempty"`
	Acks        *string       `yaml:"acks,omitempty"`
	FileSize    *string       `yaml:"fileSize,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty"`
}

type PluginsRawConfig struct {
	S3    *S3OutputRawConfig
	Kafka *KafkaOutputRawConfig
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
	Output                 *OutputRawConfig
	TLS                    TLSConfig
	Auth                   AuthConfig
	CompressBlockSize      *int             `yaml:"compressBlockSize,omitempty"`
	InputPacketLen         *int             `yaml:"inputPacketLen,omitempty"`
	GatherMaxWaitSec       *int             `yaml:"gatherMaxWaitSec,omitempty"`
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
	MaxGatherWait          time.Duration
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
	var kafkaConfig *KafkaPluginConfig
	if rawConfig.Output != nil && rawConfig.Output.Plugins != nil {

		s3Config, err = populateS3Config(rawConfig)

		if err != nil {
			return nil, err
		}

		kafkaConfig, err = populateKafkaConfig(rawConfig)

		if err != nil {
			return nil, err
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

	gatherMaxWaitSec := 5
	if rawConfig.GatherMaxWaitSec != nil {
		gatherMaxWaitSec = *rawConfig.GatherMaxWaitSec
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
				S3:    s3Config,
				Kafka: kafkaConfig,
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
		MaxGatherWait: time.Duration(gatherMaxWaitSec) * time.Second,
		MaxPayloadLen: s2.MaxEncodedLen(compressBlockSize*kilobyte) + /*hdrData*/ 4 + /*payloadMarker*/ 4,
		MaxHeaderLen:  + /*hdrData*/ 4 + /*payloadMarker*/ 4,
	}

	return config, nil
}

func populateKafkaConfig(rawConfig RawConfig) (*KafkaPluginConfig, error) {
	if rawConfig.Output.Plugins.Kafka == nil {
		return nil, nil
	}

	var (
		clientId    string
		topic       string
		acks        string
		messageSize *bytesize.ByteSize
		fileSize    *bytesize.ByteSize
	)

	rawKafkaConfig := rawConfig.Output.Plugins.Kafka

	if rawKafkaConfig.ClientId != nil {
		clientId = *rawKafkaConfig.ClientId
	} else {
		clientId = defaultClientId
	}

	if rawKafkaConfig.Topic != nil {
		topic = *rawKafkaConfig.Topic
	} else {
		topic = defaultTopic
	}

	if rawKafkaConfig.MessageSize != nil {
		ms, err := bytesize.Parse(*rawKafkaConfig.MessageSize)
		if err != nil {
			return nil, fmt.Errorf("could not parse the messageSize field %s: %w", *rawKafkaConfig.MessageSize, err)
		}
		messageSize = &ms
	} else {
		ms := 65 * bytesize.KB
		messageSize = &ms
	}

	if rawKafkaConfig.Acks != nil {
		acks = *rawKafkaConfig.Acks
	} else {
		acks = defaultAcks
	}

	if rawKafkaConfig.FileSize != nil {
		fs, err := bytesize.Parse(*rawKafkaConfig.FileSize)
		if err != nil {
			return nil, fmt.Errorf("could not parse the fileSize field %s: %w", *rawKafkaConfig.FileSize, err)
		}
		fileSize = &fs
	} else {
		fs := 1 * bytesize.MB
		fileSize = &fs
	}

	return &KafkaPluginConfig{
		Brokers:     strings.Split(rawConfig.Output.Plugins.Kafka.Brokers, ","),
		ClientId:    clientId,
		Topic:       topic,
		MessageSize: messageSize,
		Acks:        acks,
		FileSize:    fileSize,
		Timeout:     rawConfig.Output.Plugins.Kafka.Timeout,
	}, nil
}

func populateS3Config(rawConfig RawConfig) (*S3PluginConfig, error) {
	if rawConfig.Output.Plugins.S3 == nil {
		return nil, nil
	}

	var (
		totalFileSize   *bytesize.ByteSize
		uploadTimeout   time.Duration
		uploadChunkSize *bytesize.ByteSize
		cannedACL       string
	)

	if rawConfig.Output.Plugins.S3.TotalFileSize != nil {
		t, err := bytesize.Parse(*rawConfig.Output.Plugins.S3.TotalFileSize)
		if err != nil {
			return nil, fmt.Errorf("could not parse the totalFileSize field %s: %w", *rawConfig.Output.Plugins.S3.TotalFileSize, err)
		}
		totalFileSize = &t
	} else {
		t := 10 * bytesize.MB
		totalFileSize = &t
	}

	if rawConfig.Output.Plugins.S3.UploadTimeout != nil {
		var err error
		uploadTimeout, err = time.ParseDuration(*rawConfig.Output.Plugins.S3.UploadTimeout)
		if err != nil {
			return nil, fmt.Errorf("could not parse the uploadTimeout field %s: %w", *rawConfig.Output.Plugins.S3.UploadTimeout, err)
		}
	} else {
		uploadTimeout = time.Minute
	}

	if rawConfig.Output.Plugins.S3.UploadChunkSize != nil {
		u, err := bytesize.Parse(*rawConfig.Output.Plugins.S3.UploadChunkSize)
		if err != nil {
			return nil, fmt.Errorf("could not parse the uploadChunkSize field %s: %w", *rawConfig.Output.Plugins.S3.UploadChunkSize, err)
		}
		uploadChunkSize = &u
	} else {
		u := 5 * bytesize.MB
		totalFileSize = &u
	}

	if rawConfig.Output.Plugins.S3.CannedACL != nil {
		cannedACL = *rawConfig.Output.Plugins.S3.CannedACL
	} else {
		cannedACL = string(types.ObjectCannedACLBucketOwnerFullControl)
	}

	return &S3PluginConfig{
		Bucket:          rawConfig.Output.Plugins.S3.Bucket,
		Region:          rawConfig.Output.Plugins.S3.Region,
		TotalFileSize:   totalFileSize,
		UploadChunkSize: uploadChunkSize,
		UploadTimeout:   uploadTimeout,
		CannedACL:       cannedACL,
	}, nil
}
