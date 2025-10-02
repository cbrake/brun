package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbrake/simpleci"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-file>\n", os.Args[0])
		os.Exit(1)
	}

	configFile := os.Args[1]

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

	// Run all trigger units
	ctx := context.Background()
	for _, unit := range units {
		fmt.Printf("Running unit '%s' (type: %s)\n", unit.Name(), unit.Type())
		if err := unit.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error running unit '%s': %v\n", unit.Name(), err)
			os.Exit(1)
		}
	}

	fmt.Println("All units completed successfully")
}
