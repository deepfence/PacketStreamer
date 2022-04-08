package output

import (
	"context"
	"io"

	"golang.org/x/sync/errgroup"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

type Outputs struct {
	outputFds []io.WriteCloser
}

func NewOutputs(config *config.Config) (*Outputs, error) {
	outputFds := make([]io.WriteCloser, 0)

	if config.Output.File != nil {
		fileFd, err := newFileOutput(config)
		if err != nil {
			return nil, err
		}
		outputFds = append(outputFds, fileFd)
	} else if config.Output.Server != nil {
		serverFd, err := newServerOutput(config)
		if err != nil {
			return nil, err
		}
		outputFds = append(outputFds, serverFd)
	}

	return &Outputs{outputFds: outputFds}, nil
}

func (o *Outputs) WriteAll(ctx context.Context, data []byte) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, outputFd := range o.outputFds {
		g.Go(func() error {
			_, err := outputFd.Write(data)
			return err
		})
	}
	return g.Wait()
}
