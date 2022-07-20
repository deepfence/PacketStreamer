package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/deepfence/PacketStreamer/pkg/config"
	pb "github.com/deepfence/PacketStreamer/proto"
	"google.golang.org/grpc"
)

const (
	SOCKET_RETRIES = 10
)

type Plugin struct {
	client   pb.PcapForwarderClient
	stopChan chan struct{}
}

func NewPlugin(conf *config.AgentPluginConfig) (*Plugin, error) {

	_, errw := os.Stat(conf.SocketPath)
	tries := 1
	for errors.Is(errw, os.ErrNotExist) {
		_, errw = os.Stat(conf.SocketPath)
		time.Sleep(1 * time.Second)
		if tries == SOCKET_RETRIES {
			return nil, fmt.Errorf("No socket found: %v", errw)
		}
		tries += 1
	}

	conn, err := grpc.Dial("unix://"+conf.SocketPath, grpc.WithAuthority("dummy"), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Plugin{
		client:   pb.NewPcapForwarderClient(conn),
		stopChan: make(chan struct{}, 1),
	}, nil
}

func (p *Plugin) Start(ctx context.Context) (chan<- string, error) {

	stream, err := p.client.SendPackets(ctx)

	if err != nil {
		return nil, err
	}

	recvChan := make(chan string, 10)

	go func() {
	loop:
		for {
			select {
			case packet := <-recvChan:
				stream.Send(&pb.Packet{Payload: []byte(packet)})
			case <-p.stopChan:
				break loop
			}
		}
		stream.CloseAndRecv()
	}()

	return recvChan, nil
}

func (p *Plugin) Stop() {
	p.stopChan <- struct{}{}
}
