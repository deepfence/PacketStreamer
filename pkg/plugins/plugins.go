package plugins

import (
	"context"
	"fmt"
	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/plugins/s3"
)

//Start uses the provided config to start the execution of any plugin outputs that have been defined.
//Packets that are written to the returned channel will be fanned out to N configured plugins.
func Start(ctx context.Context, config *config.Config) (chan<- string, error) {
	if !pluginsAreDefined(config.Output.Plugins) {
		return nil, nil
	}

	var plugins []chan<- string

	if config.Output.Plugins.S3 != nil {
		s3plugin, err := s3.NewPlugin(ctx, config.Output.Plugins.S3)

		if err != nil {
			return nil, fmt.Errorf("error starting S3 plugin, %v", err)
		}

		s3Chan := s3plugin.Start(ctx)
		plugins = append(plugins, s3Chan)
	}

	inputChan := make(chan string)
	go func() {
		defer func() {
			for _, p := range plugins {
				close(p)
			}
		}()

		for {
			select {
			case pkt := <-inputChan:
				for _, p := range plugins {
					p <- pkt
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return inputChan, nil
}

func pluginsAreDefined(pluginsConfig *config.PluginsConfig) bool {
	return pluginsConfig != nil && pluginsConfig.S3 != nil
}
