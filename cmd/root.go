package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

var (
	cfgFile string
	cfg     *config.Config

	rootCmd = &cobra.Command{
		Use:   "packetstreamer",
		Short: "A tool for streaming packets from one server to another",
		Long: `A simple tool that helps us to stream packets (network traffic)
from one server to another. The servers could be hosted in cloud environments,
internal data centers, or regular desktops.`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}

func initConfig() {
	if cfgFile == "" {
		log.Fatalf("Configuration file not provided")
	}

	var err error
	cfg, err = config.NewConfig(cfgFile)
	if err != nil {
		log.Fatalf("Could not retrieve configuration: %v", err)
	}
}
