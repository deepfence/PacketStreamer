package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

type FileOutput struct {
	fd io.WriteCloser
}

func (o *FileOutput) Write(data []byte) (int, error) {
	return o.fd.Write(data)
}

func newFileOutput(config *config.Config) (io.WriteCloser, error) {
	var pcapBuffer bytes.Buffer
	pcapWriter := pcapgo.NewWriter(&pcapBuffer)

	fileFd, err := os.OpenFile(config.Output.File.Path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", config.Output.File.Path, err)
	}

	pcapWriter.WriteFileHeader(uint32(config.InputPacketLen), layers.LinkTypeEthernet)
	fileFd.Write(pcapBuffer.Bytes())

	return fileFd, nil
}