/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

const (
	optChip  = "chip"
	optSize  = "size"
	optOrg   = "org"
	optPort  = "port"
	optXor   = "xor"
	optErase = "erase"

	defaultChip = 66
	defaultSize = 512
	defaultOrg  = 8
	defaultPort = "COM8"

	opWrite = "w"
	opRead  = "r"
	opErase = "e"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "eep",
	Short:        "eeprom tool",
	Long:         `a CLI to interface with the arduino eeprom programmer`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	ctx, cancel := context.WithCancel(context.TODO())

	go func() {
		<-sig
		cancel()
		time.Sleep(10 * time.Second)
		os.Exit(1)
	}()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()

	pf.Uint8P(optChip, "c", defaultChip, "chip type")
	pf.Uint16P(optSize, "s", defaultSize, "chip size")
	pf.Uint8P(optOrg, "o", defaultOrg, "chip org")
	pf.StringP(optPort, "p", defaultPort, "com port")
	pf.BytesHexP(optXor, "x", []byte{0x00}, "xor output")
	pf.BoolP(optErase, "e", false, "erase before write (default false")

	//cobra.MarkFlagRequired(pf, optChip)
	//cobra.MarkFlagRequired(pf, optSize)
	//cobra.MarkFlagRequired(pf, optOrg)

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getFlags() (uint8, uint16, uint8, string, error) {
	chip, err := rootCmd.PersistentFlags().GetUint8(optChip)
	if err != nil {
		return 0, 0, 0, "", err
	}
	size, err := rootCmd.PersistentFlags().GetUint16(optSize)
	if err != nil {
		return 0, 0, 0, "", err
	}
	org, err := rootCmd.PersistentFlags().GetUint8(optOrg)
	if err != nil {
		return 0, 0, 0, "", err
	}
	port, err := rootCmd.PersistentFlags().GetString(optPort)
	if err != nil {
		return 0, 0, 0, "", err
	}
	return chip, size, org, port, nil
}

func waitAck(stream serial.Port, char byte) error {
	start := time.Now()
	readBuffer := make([]byte, 1)
	for {
		n, err := stream.Read(readBuffer)
		if err != nil {
			return err
		}
		if time.Since(start) > 2*time.Second {
			return errors.New("got no ack")
		}
		if n == 0 {
			continue
			//return errors.New("got no ack")
		}
		if readBuffer[0] == char {
			log.Println("got ack")
			return nil
		}

	}

}
