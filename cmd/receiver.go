package cmd

import (
	"log"

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

		mainSignalChannel := make(chan bool)

		proto := "tcp"
		if err := streamer.InitOutput(cfg, proto); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		log.Println("Start receiving")
		streamer.StartReceiver(cfg, proto, mainSignalChannel)
		log.Println("Now waiting in main")
		<-mainSignalChannel
	},
}

func init() {
	rootCmd.AddCommand(receiverCmd)
}
