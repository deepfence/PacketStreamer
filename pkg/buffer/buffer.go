package buffer

import (
	"bufio"
	"fmt"
	"io"

	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/pcap"
)

// PacketBufWriter is a writer based on bufio.Writer with one difference: when
// the current write is going to exceed the buffer size, it flushes the buffer,
// so the whole packet can be written to the flushed buffer again.
type PacketBufWriter struct {
	config *config.Config

	*bufio.Writer
}

func NewPacketBufWriter(config *config.Config, w io.Writer, size int) *PacketBufWriter {
	return &PacketBufWriter{
		config: config,
		Writer: bufio.NewWriterSize(w, size),
	}
}

// Flush flushes the buffer and writes a pcap header for a new file.
func (b *PacketBufWriter) Flush() error {
	if err := b.Writer.Flush(); err != nil {
		return fmt.Errorf("could not flush the buffer: %w", err)
	}

	pcapBuffer, err := pcap.PcapHeaderBuffer(b.config)
	if err != nil {
		return err
	}

	if _, err := b.Writer.Write(pcapBuffer); err != nil {
		return fmt.Errorf("could not write pcap header: %w", err)
	}

	return nil
}

// Write writes the given packet buffer to the underlying writer. When the
// packet is going to exceed the buffer size, the buffer gets flushed before
// the write.
func (b *PacketBufWriter) Write(p []byte) (int, error) {
	if len(p) > b.Available() {
		if err := b.Flush(); err != nil {
			return 0, fmt.Errorf("could not flush the buffer before the next write: %w", err)
		}
	}
	return b.Writer.Write(p)
}
