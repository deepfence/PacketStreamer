package streamer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"github.com/deepfence/PacketStreamer/pkg/plugins"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

const (
	maxNumPkts       = 100
	connTimeout      = 60
)

func readDataFromSocket(hostConn net.Conn, dataBuff []byte, bytesToRead int) error {

	totalBytesRead := 0

	for {
		deadLineErr := hostConn.SetReadDeadline(time.Now().Add(connTimeout * time.Second))
		if deadLineErr != nil {
			log.Println(fmt.Sprintf("Unable to set timeout for connection from %s. Reason %v",
				hostConn.RemoteAddr(), deadLineErr))
		}
		bytesRead, readErr := hostConn.Read(dataBuff[totalBytesRead:])
		if (readErr != nil) && (readErr != io.EOF) && !os.IsTimeout(readErr) {
			return fmt.Errorf("Client %s closed connection. Reason = %v", hostConn.RemoteAddr(), readErr)
		}
		if (readErr == io.EOF) && (totalBytesRead != bytesToRead) {
			return fmt.Errorf("Client %s closed connection abruptly. Got EOF", hostConn.RemoteAddr())
		}
		if os.IsTimeout(readErr) && (totalBytesRead != bytesToRead) {
			return readErr
		}
		if (bytesRead == 0) && (readErr != nil) {
			log.Printf("Zero bytes received from client. Error reason %v\n", readErr)
			return nil
		}
		if (bytesRead == 0) && (readErr == nil) {
			log.Println("Zero bytes received from client. No errors")
			return nil
		}
		totalBytesRead += bytesRead
		if totalBytesRead == bytesToRead {
			return nil
		}
	}
}

func readPkts(clientConn net.Conn, config *config.Config, pktUncompressChannel chan string, sizeChannel chan int) {

	var dataBuff = make([]byte, config.MaxPayloadLen)
	hdrDataLen := len(hdrData)
	var totalHdrLen = config.MaxHeaderLen

	for {
		err := readDataFromSocket(clientConn, dataBuff[0:totalHdrLen], totalHdrLen)
		if err != nil {
			if !os.IsTimeout(err) {
				log.Printf("Unable to read data from connection. %v\n", err)
			}
			clientConn.Close()
			close(pktUncompressChannel)
			return
		}
		compareRes := bytes.Compare(dataBuff[0:hdrDataLen], hdrData[:])
		if compareRes != 0 {
			log.Printf("Illegal data received from client")
			clientConn.Close()
			close(pktUncompressChannel)
			return
		}
		compressedDataLen := binary.LittleEndian.Uint32(dataBuff[hdrDataLen:])
		if int(compressedDataLen) > (config.MaxEncodedLen - totalHdrLen) {
			log.Printf("Invalid buffer length %d obtained from client", compressedDataLen)
			clientConn.Close()
			close(pktUncompressChannel)
			return
		}
		err = readDataFromSocket(clientConn, dataBuff[totalHdrLen:(totalHdrLen+int(compressedDataLen))], int(compressedDataLen))
		if err != nil {
			log.Printf("Unable to read data from connection. %s\n", err)
			clientConn.Close()
			close(pktUncompressChannel)
			return
		}
		select {
		case pktUncompressChannel <- string(dataBuff[totalHdrLen:(int(compressedDataLen) + totalHdrLen)]):
		default:
			log.Println("Uncompress queue is full. Discarding")
		}
		select {
		case sizeChannel <- (totalHdrLen + int(compressedDataLen)):
		default:
			log.Println("Size queue is full. Discarding")
		}
	}
}

func receiverOutput(ctx context.Context, config *config.Config, consolePktOutputChannel chan string, pluginChan chan<- string) {
	for {
		select {
		case tmpData, chanExitVal := <-consolePktOutputChannel:
			if !chanExitVal {
				log.Println("Error while reading from output channel")
				break
			}

			if pluginChan != nil {
				pluginChan <- tmpData
			}

			if writeOutput(config, []byte(tmpData)) == 1 {
				break
			}
		case <-ctx.Done():
			break
		}
	}
}

func processHost(config *config.Config, consolePktOutputChannel chan string, proto string) {

	var err error
	var listener net.Listener
	addr := config.Input.Address
	if config.Input.Port != nil {
		addr = fmt.Sprintf("%s:%d", config.Input.Address, *config.Input.Port)
	}

	if config.TLS.Enable {
		config, err := getTlsConfig(config.TLS.CertFile, config.TLS.KeyFile, "")
		if err != nil {
			log.Println("Unable to start TLS listener: " + err.Error())
			return
		}
		listener, err = tls.Listen(proto, addr, config)
		if err != nil {
			log.Println("Unable to start TLS listener socket "+err.Error(), proto, addr, config)
			return
		}
	} else {
		listener, err = net.Listen(proto, addr)
		if err != nil {
			log.Println("Unable to start listener socket "+err.Error(), proto, addr)
			return
		}
	}

	sizeChannel := make(chan int, maxNumPkts)
	go calculateDataSize(sizeChannel)

	for {
		hostConn, cerr := listener.Accept()
		if cerr != nil {
			log.Println("Unable to accept connections on socket " + cerr.Error())
			break
		} else {
			log.Println("Accepted connection on socket: ", proto, hostConn.RemoteAddr())
		}
		if config.Auth.Enable {
			go func() {
				if handleServerAuth(hostConn) {
					pktUncompressChannel := make(chan string, maxNumPkts)
					go decompressPkts(config, pktUncompressChannel, consolePktOutputChannel)
					go readPkts(hostConn, config, pktUncompressChannel, sizeChannel)
				}
			}()
			continue
		}
		pktUncompressChannel := make(chan string, maxNumPkts)
		go decompressPkts(config, pktUncompressChannel, consolePktOutputChannel)
		go readPkts(hostConn, config, pktUncompressChannel, sizeChannel)
	}
}

func StartReceiver(ctx context.Context, config *config.Config, proto string) {
	ticker := time.NewTicker(1 * time.Minute)
	consolePktOutputChannel := make(chan string, maxNumPkts*10)

	pluginChan, err := plugins.Start(ctx, config)
	if err != nil {
		// log but carry on, we still might want to see the receiver output despite the broken plugins
		log.Println(err)
	}
	go receiverOutput(ctx, config, consolePktOutputChannel, pluginChan)
	go processHost(config, consolePktOutputChannel, proto)

	go func() {
		for {
			select {
			case <-ticker.C:
				printDataSize()
			case <-ctx.Done():
				if pluginChan != nil {
					close(pluginChan)
				}
				return
			}
		}
	}()
}
