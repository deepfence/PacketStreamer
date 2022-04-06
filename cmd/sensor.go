package cmd

import (
	"log"

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

		mainSignalChannel := streamer.NewSignalChannel()
		done := make(chan bool)

		proto := "tcp"
		if err := streamer.InitOutput(cfg, proto); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		log.Println("Start sending")
		streamer.StartSensor(cfg, done)
		log.Println("Now waiting in main")

		go func() {
			<-mainSignalChannel
			done <- true
		}()

		<-done
		streamer.FlushAndCloseOutput()
	},
}

func init() {
	rootCmd.AddCommand(sensorCmd)
}
