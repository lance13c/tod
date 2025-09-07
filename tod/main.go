package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ciciliostudio/tod/cmd"
)

var version = "dev"

func main() {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Run the command in a goroutine
	done := make(chan bool, 1)
	go func() {
		cmd.SetVersion(version)
		cmd.Execute()
		done <- true
	}()
	
	// Wait for either completion or interrupt signal
	select {
	case <-sigChan:
		// Handle Ctrl+C gracefully
		os.Exit(0)
	case <-done:
		// Normal completion
	}
}