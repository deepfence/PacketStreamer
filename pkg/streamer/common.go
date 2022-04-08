package streamer

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

const (
	maxWriteAttempts = 10
)

var (
	outputFd      io.Writer
	pktsRead      uint64
	totalDataSize uint64
)

func writeOutput(config *config.Config, tmpData []byte) int {

	var numAttempts = 0
	reconnectAttempt := false
	dataLen := len(tmpData)
	totalBytesWritten := 0
	for {
		if numAttempts == maxWriteAttempts {
			if !reconnectAttempt {
				reconnectAttempt = true
				err := InitOutput(config, "tcp")
				if err != nil {
					log.Printf("Tried to reconnect but got: %v\n", err)
					return 1
				}
				log.Printf("Tried to write for %d times. Reconnecting once. \n", numAttempts)
				numAttempts = 0
				continue
			}
			log.Printf("Tried to write for %d times. Bailing out. \n", numAttempts)
			return 1
		}

		bytesWritten, err := outputFd.Write(tmpData[totalBytesWritten:])

		if err != nil {
			log.Printf("Error while writing data to output. Reason = %s \n", err.Error())
			numAttempts += 1
			continue
		}
		if (totalBytesWritten + bytesWritten) != dataLen {
			log.Printf("Not all bytes written to output. Wanted to write  %d, but wrote only %d \n", (dataLen - totalBytesWritten), bytesWritten)
			totalBytesWritten += bytesWritten
			numAttempts += 1
			continue
		}

		return 0
	}
}

func InitOutput(config *config.Config, proto string) error {

	if config.Output.File != nil {
		var pcapBuffer bytes.Buffer
		pcapWriter := pcapgo.NewWriter(&pcapBuffer)
		fileFd := os.Stdout
		if config.Output.File.Path != "stdout" {
			var err error
			fileFd, err = os.OpenFile(config.Output.File.Path, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				return err
			}
		}
		pcapWriter.WriteFileHeader(uint32(config.InputPacketLen), layers.LinkTypeEthernet)
		fileFd.Write(pcapBuffer.Bytes())
		outputFd = fileFd
	} else if config.Output.Server != nil {

		addr := config.Output.Server.Address
		if config.Output.Server.Port != nil {
			addr = fmt.Sprintf("%s:%d", config.Output.Server.Address, *config.Output.Server.Port)
		}
		var authConn authConnIntf
		if config.TLS.Enable {
			tlsConfig, err := getTlsConfig(config.TLS.CertFile, config.TLS.KeyFile, "")
			if err != nil {
				return err
			}
			conn, err := tls.Dial(proto, addr, tlsConfig)
			if err != nil {
				return err
			}
			err = conn.Handshake()
			if err != nil {
				return err
			}
			outputFd = conn
			authConn = conn
		} else {
			conn, err := net.Dial(proto, addr)
			if err != nil {
				return err
			}
			log.Println("Connection established, TLS disabled: ", proto, conn.RemoteAddr())
			outputFd = conn
			authConn = conn
		}
		if config.Auth.Enable {
			err := handleClientAuth(authConn, config.Auth.Key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func calculateDataSize(sizeChannel chan int) {
	for {
		dataSize := <-sizeChannel
		totalDataSize += uint64(dataSize)
	}
}

func printDataSize() {
	currSize := totalDataSize
	v := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	l := 0
	for ; currSize > kilobyte; currSize = currSize / kilobyte {
		l++
	}
	log.Printf("Total data transfer size is %d %s\n", currSize, v[l])
}

func printPacketCount() {
	log.Printf("Total packets read from interface is %d\n", pktsRead)
}
