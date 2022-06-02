package streamer

import (
	"context"
	"encoding/binary"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket/pcap"

	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/plugins"
)

func StartSensor(ctx context.Context, config *config.Config) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				printPacketCount()
			}
		}
	}()
	agentOutputChan := make(chan string, maxNumPkts)
	pluginChan, err := plugins.Start(ctx, config)
	if err != nil {
		// log but carry on, we still might want to see the receiver output despite the broken plugins
		log.Println(err)
	}
	go sensorOutput(ctx, config, agentOutputChan)
	go processIntfCapture(ctx, config, agentOutputChan, pluginChan)
}

func sensorOutput(ctx context.Context, config *config.Config, agentPktOutputChan chan string) {
	outputErr := 0
	payloadMarkerBuff := [...]byte{0x0, 0x0, 0x0, 0x0}
	dataToSend := make([]byte, config.MaxPayloadLen)
	copy(dataToSend[0:], hdrData[:])

loop:
	for {
		if outputErr == maxWriteAttempts {
			log.Printf("Error while writing %d packets to output. Giving up \n", maxWriteAttempts)
			break
		}
		select {
		case tmpData, chanExitVal := <-agentPktOutputChan:
			if !chanExitVal {
				log.Println("Error while reading from output channel")
				break loop
			}

			outputData := []byte(tmpData)
			outputDataLen := len(outputData)
			startIdx := len(hdrData)
			binary.LittleEndian.PutUint32(payloadMarkerBuff[:], uint32(outputDataLen))
			copy(dataToSend[startIdx:], payloadMarkerBuff[:])
			startIdx += len(payloadMarkerBuff)
			copy(dataToSend[startIdx:], outputData[:])
			startIdx += outputDataLen
			if err := writeOutput(config, dataToSend[0:startIdx]); err != nil {
				log.Printf("Error while writing to output: %s\n", err)
				break loop
			}
		case <-ctx.Done():
			break loop
		}
	}
}

func gatherPkts(config *config.Config, pktGatherChannel, compressChan chan string,
	pluginChan chan<- string) {

	var totalLen = 0
	var currLen = 0
	var packetData = make([]byte, config.MaxGatherLen)
	var tmpPacketData []byte

	for {
		tmpChanData, chanExitVal := <-pktGatherChannel
		if !chanExitVal {
			log.Println("Error while reading from gather channel")
			break
		}
		pktsRead += 1
		tmpPacketData = []byte(tmpChanData)
		currLen = len(tmpPacketData)
		if (totalLen + currLen) > config.MaxGatherLen {
			// NOTE(vadorovsky): Currently we output an uncompressed packet to
			// two channels:
			// * `compressChan` - to output the compressed packets to an another
			//    PacketStreamer server
			// * `pluginChan` - to output the raw packets to plugins
			// TODO(vadorovsky): We eventually want to compress plugin outputs
			// as well. But there is no CLI tool for uncompressing S2. Probably
			// the best thing to do would be providing a CLI in PacketStreamer
			// to read S2-compressed pcap files.
			select {
			case compressChan <- string(packetData[:totalLen]):
			default:
				log.Println("Gather compression queue is full. Discarding")
			}
			select {
			case pluginChan <- string(packetData[:totalLen]):
			default:
				log.Println("Gather output queue is full. Discarding")
			}
			totalLen = 0
		}
		copy(packetData[totalLen:], tmpPacketData[:currLen])
		totalLen += currLen
	}
}

func processIntfCapture(ctx context.Context, config *config.Config,
	agentPktOutputChannel chan string, pluginChan chan<- string) {

	pktGatherChannel := make(chan string, maxNumPkts*500)
	pktCompressChannel := make(chan string, maxNumPkts)

	var wg sync.WaitGroup
	go gatherPkts(config, pktGatherChannel, pktCompressChannel, pluginChan)
	go compressPkts(config, pktCompressChannel, agentPktOutputChannel)

	if len(config.CapturePorts) == 0 && len(config.CaptureInterfacesPorts) == 0 {
		captureHandles, err := initAllInterfaces(config)
		if err != nil {
			log.Fatalf("Unable to init interfaces:%v\n", err)
		}
		for _, intf := range captureHandles {
			wg.Add(1)
			go func(intf *pcap.Handle) {
				readPacketOnIntf(config, intf, pktGatherChannel)
				wg.Done()
			}(intf)
		}
	} else {
		capturing := make(map[string]*pcap.Handle)
		toUpdate := grabInterface(ctx, config)
		for {
			var intfPorts intfPorts
			select {
			case intfPorts = <-toUpdate:
			case <-ctx.Done():
				break
			}
			if capturing[intfPorts.name] == nil {
				handle, err := initInterface(config, intfPorts.name, intfPorts.ports)
				if err != nil {
					log.Fatalf("Unable to init interface %v: %v\n", intfPorts.name, err)
				}
				capturing[intfPorts.name] = handle
				wg.Add(1)
				go func(intf *pcap.Handle) {
					readPacketOnIntf(config, intf, pktGatherChannel)
					wg.Done()
				}(handle)
				log.Printf("New interface setup: %v\n", intfPorts.name)
			} else {
				bpfString, err := createBpfString(config, net.DefaultResolver, intfPorts.ports)
				if err != nil {
					log.Fatalf("Could not generate BPF filter: %v\n", err)
				}
				filter := strings.Replace(bpfString, bpfParamInputDelimiter, bpfParamOutputDelimiter, -1)
				if filter != "" {
					log.Printf("Existing interface %v updated with: %v\n", intfPorts.name, filter)
					capturing[intfPorts.name].SetBPFFilter(filter)
				}
			}
		}

	}
	wg.Wait()
	close(pktGatherChannel)
	close(pktCompressChannel)
}
