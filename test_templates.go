package main

import (
	"fmt"
	"os"
	
	"github.com/netcrate/netcrate/internal/templates"
)

func main() {
	fmt.Println("Testing Template Registry...")
	
	// Create registry
	registry := templates.NewRegistry()
	
	// Load templates
	fmt.Println("Loading templates...")
	err := registry.LoadTemplates()
	if err != nil {
		fmt.Printf("Error loading templates: %v\n", err)
		return
	}
	
	// Print debug info
	registry.PrintIndex()
	
	// List templates
	fmt.Println("\nListing templates:")
	templateList := registry.List()
	for _, tmpl := range templateList {
		fmt.Printf("- %s v%s (%s): %s\n", tmpl.Name, tmpl.Version, tmpl.Source, tmpl.Description)
	}
	
	// Test specific template retrieval
	fmt.Println("\nTesting template retrieval:")
	if tmpl, exists := registry.Get("basic_scan"); exists {
		fmt.Printf("Found basic_scan: %s (source: %s)\n", tmpl.Description, tmpl.Source)
	}
	
	if tmpl, exists := registry.Get("custom_scan"); exists {
		fmt.Printf("Found custom_scan: %s (source: %s)\n", tmpl.Description, tmpl.Source)
	}
}