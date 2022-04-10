package streamer

import (
	"errors"
	"testing"

	"github.com/foxcpp/go-mockdns"

	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/utils"
)

func TestCreateBpfString(t *testing.T) {
	resolver := mockdns.Resolver{
		Zones: map[string]mockdns.Zone{
			"packetstreamer.io.": {
				A: []string{"172.68.142.37"},
			},
		},
	}

	for _, tt := range []struct {
		testName      string
		expectedError error
		config        *config.Config
		portList      []int
		expected      string
	}{
		{
			testName:      "no server, no ports",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: nil,
				},
				PcapMode: config.Allow,
			},
			portList: nil,
			expected: "",
		},
		{
			testName:      "no server, pcap allow",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: nil,
				},
				PcapMode: config.Allow,
			},
			portList: []int{8000, 8001, 8002},
			expected: "port 8000 or port 8001 or port 8002",
		},
		{
			testName:      "no server, pcap deny",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: nil,
				},
				PcapMode: config.Deny,
			},
			portList: []int{8000, 8001, 8002},
			expected: "not ( port 8000 or port 8001 or port 8002 )",
		},
		{
			testName:      "no server, pcap all",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: nil,
				},
				PcapMode: config.All,
			},
			portList: []int{8000, 8001, 8002},
			expected: "",
		},
		{
			testName:      "server, no ports",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "192.168.0.30",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.Allow,
			},
			portList: nil,
			expected: "not ( dst host 192.168.0.30 and port 9000 )",
		},
		{
			testName:      "server, pcap allow",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "192.168.0.30",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.Allow,
			},
			portList: []int{8000, 8001, 8002},
			expected: "not ( dst host 192.168.0.30 and port 9000 ) and port 8000 or port 8001 or port 8002",
		},
		{
			testName:      "server, pcap deny",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "192.168.0.30",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.Deny,
			},
			portList: []int{8000, 8001, 8002},
			expected: "not ( dst host 192.168.0.30 and port 9000 ) and ( not ( port 8000 or port 8001 or port 8002 ) )",
		},
		{
			testName:      "server, pcap all",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "192.168.0.30",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.All,
			},
			portList: []int{8000, 8001, 8002},
			expected: "not ( dst host 192.168.0.30 and port 9000 )",
		},
		{
			testName:      "server domain, no ports",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "packetstreamer.io",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.Allow,
			},
			portList: nil,
			expected: "not ( dst host 172.68.142.37 and port 9000 )",
		},
		{
			testName:      "server domain, pcap allow",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "packetstreamer.io",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.Allow,
			},
			portList: []int{8000, 8001, 8002},
			expected: "not ( dst host 172.68.142.37 and port 9000 ) and port 8000 or port 8001 or port 8002",
		},
		{
			testName:      "server domain, pcap deny",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "packetstreamer.io",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.Deny,
			},
			portList: []int{8000, 8001, 8002},
			expected: "not ( dst host 172.68.142.37 and port 9000 ) and ( not ( port 8000 or port 8001 or port 8002 ) )",
		},
		{
			testName:      "server domain, pcap all",
			expectedError: nil,
			config: &config.Config{
				Output: config.OutputConfig{
					Server: &config.ServerOutputConfig{
						Address: "packetstreamer.io",
						Port:    utils.IntPtr(9000),
					},
				},
				PcapMode: config.All,
			},
			portList: []int{8000, 8001, 8002},
			expected: "not ( dst host 172.68.142.37 and port 9000 )",
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			bpfString, err := createBpfString(tt.config, &resolver, tt.portList)
			if err != nil && !errors.Is(err, tt.expectedError) {
				t.Fatalf("Unexpected error: %v", err)
			}
			if bpfString != tt.expected {
				t.Fatalf("expected: '%s', got '%s'", tt.expected, bpfString)
			}
		})
	}
}
