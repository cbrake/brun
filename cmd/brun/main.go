package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbrake/brun"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "install":
		cmdInstall(args)
	case "run":
		cmdRun(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [args]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  run <config-file> [-daemon]    Run brun with the given config file\n")
	fmt.Fprintf(os.Stderr, "                                  -daemon: run in daemon mode (continuous monitoring)\n")
	fmt.Fprintf(os.Stderr, "  install                        Install brun as a systemd service\n")
}

func cmdInstall(_ []string) {
	if err := brun.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Installation completed successfully")
}

func cmdRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s run <config-file> [-daemon]\n", os.Args[0])
		os.Exit(1)
	}

	configFile := args[0]
	daemonMode := false

	// Check for -daemon flag
	if len(args) > 1 && args[1] == "-daemon" {
		daemonMode = true
	}

	// Load configuration
	config, err := brun.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create units from configuration
	units, err := config.CreateUnits()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating units: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d unit(s)\n", len(units))

	// Create orchestrator and run units
	ctx := context.Background()
	orchestrator := brun.NewOrchestrator(units)

	if daemonMode {
		fmt.Println("Running in daemon mode (press Ctrl+C to stop)...")
		if err := orchestrator.RunDaemon(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error running orchestrator: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := orchestrator.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error running orchestrator: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All units completed successfully")
	}
}
