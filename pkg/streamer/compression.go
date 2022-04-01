package streamer

import (
	"log"

	"github.com/klauspost/compress/s2"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

const (
	kilobyte = 1024
)

func compressPkts(config *config.Config, pktCompressChannel, output chan string) {
	var sizeForEncoding = (config.CompressBlockSize * kilobyte)
	var packetData = make([]byte, sizeForEncoding)

	for {
		inputData, chanExitVal := <-pktCompressChannel
		if !chanExitVal {
			log.Println("Error while reading from compression channel")
			break
		}
		compressedData := s2.Encode(packetData, []byte(inputData))
		select {
		case output <- string(compressedData):
		default:
			log.Println("Compression output queue is full. Discarding")
		}
	}
}

func decompressPkts(config *config.Config, pktUncompressChannel, output chan string) {

	var sizeForEncoding = (config.CompressBlockSize * kilobyte)
	var packetData = make([]byte, sizeForEncoding)

	for {
		decompressBuff, chanExitVal := <-pktUncompressChannel
		if chanExitVal == false {
			log.Println("Exiting uncompress channel")
			break
		}
		deCompressedData, err := s2.Decode(packetData, []byte(decompressBuff))
		if err != nil {
			log.Printf("Error while S2 decompress. Reason %s\n", err.Error())
			continue
		}
		select {
		case output <- string(deCompressedData):
		default:
			log.Println("Decompression output channel is full. Discarding")
		}
	}
}
