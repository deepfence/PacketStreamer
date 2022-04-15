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

var receiverCmd = &cobra.Command{
	Use:   "receiver",
	Short: "Receiver (server) which retrieves packets from sensors",
	Long: `Receiver (server) which retrieves packets from sensors (clients) via
TCP and is able to store them (i.e. output to file) or pass them further.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.ValidateReceiverConfig(cfg); err != nil {
			log.Fatalf("Invalid configuration: %v", err)
		}

		proto := "tcp"
		if err := streamer.InitOutput(cfg, proto); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		ctx, cancel := context.WithCancel(context.Background())

		log.Println("Start receiving")
		streamer.StartReceiver(ctx, cfg, proto)
		log.Println("Now waiting in main")
		<-sigs
		cancel()
	},
}

func init() {
	rootCmd.AddCommand(receiverCmd)
}
