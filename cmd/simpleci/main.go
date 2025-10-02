package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbrake/simpleci"
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
	fmt.Fprintf(os.Stderr, "  run <config-file>    Run simpleci with the given config file\n")
	fmt.Fprintf(os.Stderr, "  install              Install simpleci as a systemd service\n")
}

func cmdInstall(_ []string) {
	if err := simpleci.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Installation completed successfully")
}

func cmdRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s run <config-file>\n", os.Args[0])
		os.Exit(1)
	}

	configFile := args[0]

	// Load configuration
	config, err := simpleci.LoadConfig(configFile)
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
	orchestrator := simpleci.NewOrchestrator(units)

	if err := orchestrator.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error running orchestrator: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("All units completed successfully")
}
