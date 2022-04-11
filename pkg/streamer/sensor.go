package streamer

import (
	"encoding/binary"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket/pcap"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

func StartSensor(config *config.Config, mainSignalChannel chan bool) {
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
	go sensorOutput(config, agentOutputChan, mainSignalChannel)
	go processIntfCapture(config, agentOutputChan, mainSignalChannel)
}

func sensorOutput(config *config.Config, agentPktOutputChannel chan string, mainSignalChannel chan bool) {

	outputErr := 0
	payloadMarkerBuff := [...]byte{0x0, 0x0, 0x0, 0x0}
	dataToSend := make([]byte, config.MaxPayloadLen)
	copy(dataToSend[0:], hdrData[:])
	for {
		if outputErr == maxWriteAttempts {
			log.Printf("Error while writing %d packets to output. Giving up \n", maxWriteAttempts)
			break
		}
		tmpData, chanExitVal := <-agentPktOutputChannel
		if !chanExitVal {
			log.Println("Error while reading from output channel")
			break
		}
		outputData := []byte(tmpData)
		outputDataLen := len(outputData)
		startIdx := len(hdrData)
		binary.LittleEndian.PutUint32(payloadMarkerBuff[:], uint32(outputDataLen))
		copy(dataToSend[startIdx:], payloadMarkerBuff[:])
		startIdx += len(payloadMarkerBuff)
		copy(dataToSend[startIdx:], outputData[:])
		startIdx += outputDataLen
		if writeOutput(config, dataToSend[0:startIdx]) == 1 {
			break
		}
	}
	mainSignalChannel <- true
}

func gatherPkts(config *config.Config, pktGatherChannel, output chan string) {

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
			select {
			case output <- string(packetData[0:totalLen]):
			default:
				log.Println("Gather output queue is full. Discarding")
			}
			totalLen = 0
		}
		copy(packetData[totalLen:], tmpPacketData[0:currLen])
		totalLen += currLen
	}
}

func processIntfCapture(config *config.Config, agentPktOutputChannel chan string, mainSignalChannel chan bool) {

	pktGatherChannel := make(chan string, maxNumPkts*500)
	pktCompressChannel := make(chan string, maxNumPkts)

	var wg sync.WaitGroup
	go gatherPkts(config, pktGatherChannel, pktCompressChannel)
	go compressPkts(config, pktCompressChannel, agentPktOutputChannel)

	if len(config.CapturePorts) == 0 && len(config.CaptureInterfacesPorts) == 0 {
		captureHandles, err := initAllInterfaces(config)
		if err != nil {
			log.Printf("Unable to init interfaces:%v\n", err)
			os.Exit(1)
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
		toUpdate := grabInterface(config, mainSignalChannel)
		for {
			var intfPorts intfPorts
			select {
			case intfPorts = <-toUpdate:
			case <-mainSignalChannel:
				break
			}
			if capturing[intfPorts.name] == nil {
				handle, err := initInterface(config, intfPorts.name, intfPorts.ports)
				if err != nil {
					log.Printf("Unable to init interface%v: %v\n", intfPorts.name, err)
					os.Exit(1)
				}
				capturing[intfPorts.name] = handle
				wg.Add(1)
				go func(intf *pcap.Handle) {
					readPacketOnIntf(config, intf, pktGatherChannel)
					wg.Done()
				}(handle)
				log.Printf("New interface setup: %v\n", intfPorts.name)
			} else {
				filter := strings.Replace(createBpfString(config, intfPorts.ports), bpfParamInputDelimiter, bpfParamOutputDelimiter, -1)
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
	mainSignalChannel <- true
}
