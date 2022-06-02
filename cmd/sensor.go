package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/streamer"
)

var sensorCmd = &cobra.Command{
	Use:   "sensor",
	Short: "Sensor which broadcasts locally captured packets",
	Long: `Sensor which broadcasts locally captured packets to another server
(receiver).`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.ValidateSensorConfig(cfg); err != nil {
			log.Fatalf("Invalid configuration: %v", err)
		}

		proto := "tcp"
		if err := streamer.InitOutput(cfg, proto); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		ctx, cancel := context.WithCancel(context.Background())

		log.Println("Start sending")
		streamer.StartSensor(ctx, cfg)
		log.Println("Now waiting in main")
		<-sigs
		cancel()
	},
}

func init() {
	rootCmd.AddCommand(sensorCmd)
}
