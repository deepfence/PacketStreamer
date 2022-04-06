package pcap

import (
	"bytes"
	"fmt"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"

	"github.com/deepfence/PacketStreamer/pkg/config"
)


func PcapHeaderBuffer(config *config.Config) ([]byte, error) {
	var pcapBuffer bytes.Buffer
	pcapWriter := pcapgo.NewWriter(&pcapBuffer)
	err := pcapWriter.WriteFileHeader(uint32(config.InputPacketLen), layers.LinkTypeEthernet)
	if err != nil {
		return nil, fmt.Errorf("could not write pcap header: %w", err)
	}
	return pcapBuffer.Bytes(), nil
}