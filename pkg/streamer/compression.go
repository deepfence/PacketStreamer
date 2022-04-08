package streamer

import (
	"log"

	"github.com/klauspost/compress/s2"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

const (
	kilobyte = 1024
)

type CompressData struct {
	Data string
	IsCompressed bool
}

func compressPkts(config *config.Config, pktCompressChannel chan string, output chan CompressData) {
	var sizeForEncoding = (config.CompressBlockSize * kilobyte)
	var packetData = make([]byte, sizeForEncoding)

	for {
		inputData, chanExitVal := <-pktCompressChannel
		if !chanExitVal {
			log.Println("Error while reading from compression channel")
			break
		}
		compressedData := s2.Encode(packetData, []byte(inputData))
		var dataToSend CompressData
		if len(compressedData) > sizeForEncoding {
			dataToSend = CompressData{
				Data: inputData,
				IsCompressed: false,
			}
		} else {
			dataToSend = CompressData{
				Data: string(compressedData),
				IsCompressed: true,
			}
		}

		select {
		case output <- dataToSend:
		default:
			log.Println("Compression output queue is full. Discarding")
		}
	}
}

func decompressPkts(config *config.Config, pktUncompressChannel chan CompressData, output chan string) {

	var sizeForEncoding = (config.CompressBlockSize * kilobyte)
	var packetData = make([]byte, sizeForEncoding)

	for {
		decompressBuff, chanExitVal := <-pktUncompressChannel
		if chanExitVal == false {
			// log.Println("Exiting uncompress channel")
			break
		}
		var dataToSend string
		if decompressBuff.IsCompressed {
			deCompressedData, err := s2.Decode(packetData, []byte(decompressBuff.Data))
			if err != nil {
				log.Printf("Error while S2 decompress. Reason %s\n", err.Error())
				continue
			}
			dataToSend = string(deCompressedData)
		} else {
			dataToSend = decompressBuff.Data
		}
		select {
		case output <- dataToSend:
		default:
			log.Println("Decompression output channel is full. Discarding")
		}
	}
}
