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
	fmt.Fprintf(os.Stderr, "                                  -unit <unit name>: run a single unit (useful for debugging)\n")
	fmt.Fprintf(os.Stderr, "                                   Triggers are not executed.\n")
	fmt.Fprintf(os.Stderr, "                                  -trigger <unit name>: trigger the named unit and execute\n")
	fmt.Fprintf(os.Stderr, "                                   triggers.\n")
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
		fmt.Fprintf(os.Stderr, "Usage: %s run <config-file> [-daemon] [-unit <unit name>] [-trigger <unit name>]\n", os.Args[0])
		os.Exit(1)
	}

	configFile := args[0]
	daemonMode := false
	singleUnit := ""
	triggerUnit := ""

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-daemon":
			daemonMode = true
		case "-unit":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: -unit requires a unit name\n")
				os.Exit(1)
			}
			singleUnit = args[i+1]
			i++ // Skip the next argument as it's the unit name
		case "-trigger":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: -trigger requires a unit name\n")
				os.Exit(1)
			}
			triggerUnit = args[i+1]
			i++ // Skip the next argument as it's the unit name
		default:
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
			os.Exit(1)
		}
	}

	// Validate mutually exclusive flags
	if singleUnit != "" && triggerUnit != "" {
		fmt.Fprintf(os.Stderr, "Error: -unit and -trigger cannot be used together\n")
		os.Exit(1)
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

	// Handle single unit execution (no triggers)
	if singleUnit != "" {
		fmt.Printf("Running single unit: %s (triggers disabled)\n", singleUnit)
		if err := orchestrator.RunSingleUnit(ctx, singleUnit, false); err != nil {
			fmt.Fprintf(os.Stderr, "Error running unit '%s': %v\n", singleUnit, err)
			os.Exit(1)
		}
		fmt.Printf("Unit '%s' completed successfully\n", singleUnit)
		return
	}

	// Handle trigger unit execution (with triggers)
	if triggerUnit != "" {
		fmt.Printf("Triggering unit: %s (triggers enabled)\n", triggerUnit)
		if err := orchestrator.RunSingleUnit(ctx, triggerUnit, true); err != nil {
			fmt.Fprintf(os.Stderr, "Error triggering unit '%s': %v\n", triggerUnit, err)
			os.Exit(1)
		}
		fmt.Printf("Unit '%s' and its triggers completed successfully\n", triggerUnit)
		return
	}

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
