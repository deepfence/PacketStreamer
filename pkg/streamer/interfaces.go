package streamer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"

	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/network"
)

var (
	interfaceToPortMap map[string][]int
)

const (
	bpfParamInputDelimiter  = ";"
	bpfParamOutputDelimiter = "  "
	pktCaptureTimeout       = 5
	dnsResolveTimeout       = 10
	maxReadErrCnt           = 10
	timeoutErrString        = "timeout expired"
	ioTimeoutString         = "i/o timeout"

	PROCESS_SCAN_FREQUENCY = 10 * time.Second
)

type intfPorts struct {
	name  string
	ports []int
}

func getUpInterfaces(interfaceList []net.Interface) []net.Interface {
	var upInterfaces = make([]net.Interface, 0)
	for _, interfaces := range interfaceList {
		if strings.Contains(strings.ToLower(interfaces.Flags.String()), "up") &&
			!strings.Contains(strings.ToLower(interfaces.Flags.String()), "loopback") {
			upInterfaces = append(upInterfaces, interfaces)
		}
	}
	return upInterfaces
}

func findAllInterfaces() error {
	interfaces, errVal := net.Interfaces()
	if errVal != nil {
		return errVal
	}
	upInterfaces := getUpInterfaces(interfaces)
	for _, upInterface := range upInterfaces {
		formInterfacePortMap(upInterface.Name, []int{})
	}
	return nil
}

func formInterfacePortMap(interfaceName string, portsList []int) {
	if interfaceToPortMap == nil {
		interfaceToPortMap = make(map[string][]int)
	}
	interfaceToPortMap[interfaceName] = append(interfaceToPortMap[interfaceName], portsList...)
}

func initAllInterfaces(config *config.Config) ([]*pcap.Handle, error) {
	err := findAllInterfaces()
	if err != nil {
		return nil, err
	}
	var intfPtr []*pcap.Handle
	for interfaceName, portList := range interfaceToPortMap {
		intf, err := initInterface(config, interfaceName, portList)
		if err != nil {
			return nil, err
		}
		intfPtr = append(intfPtr, intf)

	}
	return intfPtr, nil
}

func grabInterface(config *config.Config, mainSignalChannel chan bool) chan intfPorts {
	res := make(chan intfPorts)
	ticker := time.NewTicker(PROCESS_SCAN_FREQUENCY)
	go func() {
		for {
			oldMap := interfaceToPortMap
			interfaceToPortMap = map[string][]int{}
			err := setupInterfacesAndPortMappings(config)
			if err != nil {
				select {
				case <-mainSignalChannel:
					break
				case <-ticker.C:
				}
				continue
			}

			for interf, ports := range interfaceToPortMap {
				if !compareIntSets(ports, oldMap[interf]) {
					res <- intfPorts{
						interf,
						ports,
					}
				}
			}
			select {
			case <-mainSignalChannel:
				break
			case <-ticker.C:
			}
		}
	}()
	return res
}

func initInterface(config *config.Config, intfName string, portList []int) (*pcap.Handle, error) {

	if intfName == "" {
		return nil, errors.New("no interface specified")
	}

	packetHandle, err := pcap.OpenLive(intfName, int32(config.InputPacketLen), false, pktCaptureTimeout*time.Second)

	if err != nil {
		return nil, err
	}

	bpfString, err := createBpfString(config, net.DefaultResolver, portList)
	if err != nil {
		return nil, fmt.Errorf("could not generate BPF filter: %w", err)
	}
	intfBpf := strings.Replace(bpfString, bpfParamInputDelimiter, bpfParamOutputDelimiter, -1)

	if intfBpf != "" {
		bpfStrings := strings.Replace(intfBpf, bpfParamInputDelimiter, bpfParamOutputDelimiter, -1)
		err = packetHandle.SetBPFFilter(bpfStrings)
		if err != nil {
			return nil, err
		}
	}
	return packetHandle, nil
}

func readPacketOnIntf(config *config.Config, intf *pcap.Handle, pktGatherChannel chan string) {
	pktsRead := 0
	errCntr := 0
	var pcapBuffer bytes.Buffer
	var pcapWriter = pcapgo.NewWriter(&pcapBuffer)
	for {
		pcapBuffer.Reset()
		if errCntr == maxReadErrCnt {
			log.Println("Maximum packet read error reached. Exiting")
			break
		}
		pktData, pktCi, pktErr := intf.ZeroCopyReadPacketData()

		if pktErr != nil {
			if !strings.Contains(strings.ToLower(pktErr.Error()), ioTimeoutString) &&
				!strings.Contains(strings.ToLower(pktErr.Error()), timeoutErrString) {
				log.Printf("Error while reading packets. Reason = %s\n", pktErr.Error())
				errCntr += 1
				continue
			}
			continue
		}
		pktsRead = (pktsRead + 1) % config.SamplingRate.MaxTotalPkts
		if pktsRead >= config.SamplingRate.MaxPktsToWrite {
			continue
		}
		err := pcapWriter.WritePacket(pktCi, pktData)
		if err != nil {
			log.Printf("Unable to convert packet to byte buffer. Reason %v\n", err)
			continue
		}
		errCntr = 0
		select {
		case pktGatherChannel <- pcapBuffer.String():
		default:
			log.Println("Gather queue is full. Discarding ")
		}
	}
}

func resolveHost(resolver network.Resolver, host string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*dnsResolveTimeout)
	defer cancel()
	ips, err := resolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("could not resolve host %s: %w", host, err)
	}
	return ips, nil
}

/* this creates a bpf string from the list of ports */
func createBpfString(c *config.Config, resolver network.Resolver, portList []int) (string, error) {
	var portString []string = make([]string, 0)
	for _, port := range portList {
		portVal := strconv.Itoa(port)
		portVal = "port " + portVal
		portString = append(portString, portVal)
	}

	if c.Output.Server == nil {
		if len(portList) == 0 {
			return "", nil
		}

		switch c.PcapMode {
		case config.Allow:
			return strings.Join(portString, " or "), nil
		case config.Deny:
			return "not ( " + strings.Join(portString, " or ") + " )", nil
		default:
			/* this must be the all-processes mode */
			return "", nil
		}
	} else {
		var hostIPs []string
		if net.ParseIP(c.Output.Server.Address) == nil {
			ips, err := resolveHost(resolver, c.Output.Server.Address)
			if err != nil {
				return "", fmt.Errorf("unable to resolve host %s: %w", c.Output.Server.Address, err)
			}
			hostIPs = append(hostIPs, ips...)
		} else {
			hostIPs = append(hostIPs, c.Output.Server.Address)
		}

		defaultBpfString := ""
		for i, ip := range hostIPs {
			defaultBpfString += fmt.Sprintf("not ( dst host %s and port %d )", ip, *c.Output.Server.Port)
			if i != len(hostIPs)-1 {
				defaultBpfString += " and "
			}
		}

		if len(portList) == 0 {
			return defaultBpfString, nil
		}

		switch c.PcapMode {
		case config.Allow:
			return defaultBpfString + " and " + strings.Join(portString, " or "), nil
		case config.Deny:
			return defaultBpfString + " and " + "( not ( " + strings.Join(portString, " or ") + " ) )", nil
		default:
			return defaultBpfString, nil
		}
	}
}

func setupInterfacesAndPortMappings(c *config.Config) error {
	/* if it is a deny mode, and no ports have been selected, run
	 * capture on all interfaces */
	if (c.PcapMode == config.Deny && len(c.CapturePorts) == 0) || c.PcapMode == config.All {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}
		upInterfaces := getUpInterfaces(interfaces)
		for _, upInterface := range upInterfaces {
			formInterfacePortMap(upInterface.Name, []int{})
		}
		/* this is for deny mode and some ports must actually be denied */
	} else if c.PcapMode == config.Deny && len(c.CapturePorts) != 0 {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}
		upInterfaces := getUpInterfaces(interfaces)
		for _, upInterface := range upInterfaces {
			if len(c.CapturePorts) == 0 {
				formInterfacePortMap(upInterface.Name, []int{})
			} else {
				formInterfacePortMap(upInterface.Name, c.CapturePorts)
			}
		}
		for iface, ports := range c.CaptureInterfacesPorts {
			formInterfacePortMap(iface, ports)
		}
		/* this is for allow */
	} else {
		if len(c.CapturePorts) != 0 {
			interfaces, err := net.Interfaces()
			if err != nil {
				return err
			}
			upInterfaces := getUpInterfaces(interfaces)
			for _, upInterface := range upInterfaces {
				formInterfacePortMap(upInterface.Name, c.CapturePorts)
			}
		}
		for iface, ports := range c.CaptureInterfacesPorts {
			formInterfacePortMap(iface, ports)
		}
	}
	removeDuplicatePortsFromMap()
	return nil
}

func removeDuplicatePortsFromMap() {
	for interfaceName, portsList := range interfaceToPortMap {
		interfaceToPortMap[interfaceName] = Uniques(portsList)
	}
}

func Uniques(s []int) []int {
	if len(s) == 0 {
		return s
	}
	seen := make([]int, 0, len(s))
slice:
	for i, n := range s {
		if i == 0 {
			s = s[:0]
		}
		for _, t := range seen {
			if n == t {
				continue slice
			}
		}
		seen = append(seen, n)
		s = append(s, n)
	}
	return s
}

func compareIntSets(X, Y []int) bool {
	if len(X) != len(Y) {
		return false
	}
	counts := make(map[int]bool)
	for _, val := range X {
		counts[val] = true
	}
	for _, val := range Y {
		if ok := counts[val]; !ok {
			return false
		}
	}
	return true
}
