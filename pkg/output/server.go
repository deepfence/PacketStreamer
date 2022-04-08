package output

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/deepfence/PacketStreamer/pkg/auth"
	"github.com/deepfence/PacketStreamer/pkg/config"
)

const (
	maxWriteAttempts = 10
)

type ServerOutput struct {
	conn   io.WriteCloser
	config *config.Config
}

func newConnection(config *config.Config) (io.WriteCloser, error) {
	addr := config.Output.Server.Address
	if config.Output.Server.Port != nil {
		addr = fmt.Sprintf("%s:%d", config.Output.Server.Address, *config.Output.Server.Port)
	}
	var (
		serverFd io.WriteCloser
		authConn auth.AuthConnIntf
	)
	if config.TLS.Enable {
		tlsConfig, err := auth.GetTlsConfig(config.TLS.CertFile, config.TLS.KeyFile, "")
		if err != nil {
			return nil, err
		}
		conn, err := tls.Dial(*config.Output.Server.Protocol, addr, tlsConfig)
		if err != nil {
			return nil, err
		}
		err = conn.Handshake()
		if err != nil {
			return nil, err
		}
		serverFd = conn
		authConn = conn
	} else {
		conn, err := net.Dial(*config.Output.Server.Protocol, addr)
		if err != nil {
			return nil, err
		}
		log.Println("Connection established, TLS disabled: ", *config.Output.Server.Protocol, conn.RemoteAddr())
		serverFd = conn
		authConn = conn
	}
	if config.Auth.Enable {
		err := auth.HandleClientAuth(authConn, config.Auth.Key)
		if err != nil {
			return nil, err
		}
	}
	return serverFd, nil
}

func newServerOutput(config *config.Config) (*ServerOutput, error) {
	serverFd, err := newConnection(config)
	if err != nil {
		return nil, err
	}
	return &ServerOutput{
		conn:   serverFd,
		config: config,
	}, nil
}

func (o *ServerOutput) reconnect() error {
	o.conn.Close()
	
	serverFd, err := newConnection(o.config)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	o.conn = serverFd

	return nil
}

func (o *ServerOutput) Write(p []byte) (int, error) {
	numAttempts := 0
	reconnectAttempt := false
	dataLen := len(p)
	totalBytesWritten := 0
	for {
		if numAttempts == maxWriteAttempts {
			if !reconnectAttempt {
				reconnectAttempt = true
				if err := o.reconnect(); err != nil {
					return 0, err
				}
				log.Printf("Tried to write for %d times. Reconnecting once. \n", numAttempts)
				numAttempts = 0
				continue
			}
			return 0, fmt.Errorf("tried to write for %d times, bailing out", numAttempts)
		}

		bytesWritten, err := o.conn.Write(p[totalBytesWritten:])

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

		return bytesWritten, nil
	}
}

func (o *ServerOutput) Close() error {
	return o.conn.Close()
}
