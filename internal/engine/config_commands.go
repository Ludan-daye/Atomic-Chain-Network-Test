package engine

import (
	"fmt"
	"strconv"
	"time"

	"github.com/netcrate/netcrate/internal/config"
	"github.com/spf13/cobra"
)

// NewConfigCommand creates the config management command
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage NetCrate configuration",
		Long: `Configure NetCrate settings including rate profiles, preferences, and session options.
		
Rate profiles control the speed and aggressiveness of scans:
- slow: Conservative scanning (50 pps, 50 workers)
- medium: Balanced scanning (200 pps, 200 workers) 
- fast: Aggressive scanning (1000 pps, 500 workers)
- ludicrous: Maximum speed (5000 pps, 1000 workers)`,
	}

	// Add subcommands
	cmd.AddCommand(NewConfigShowCommand())
	cmd.AddCommand(NewConfigSetCommand())
	cmd.AddCommand(NewConfigRateCommand())

	return cmd
}

// NewConfigShowCommand shows current configuration
func NewConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current NetCrate configuration including rate profiles and preferences.",
		RunE:  runConfigShow,
	}
}

// NewConfigSetCommand sets configuration values
func NewConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Long: `Set configuration values. Available keys:
- output_format: table, json, yaml
- show_banners: true, false  
- color_output: true, false
- verbose: true, false
- auto_confirm_dangerous: true, false`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}
}

// NewConfigRateCommand manages rate profiles
func NewConfigRateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate",
		Short: "Manage rate profiles",
		Long:  "Manage scanning rate profiles for different speed/stealth requirements.",
	}

	cmd.AddCommand(NewConfigRateListCommand())
	cmd.AddCommand(NewConfigRateSetCommand())
	cmd.AddCommand(NewConfigRateCreateCommand())
	cmd.AddCommand(NewConfigRateDeleteCommand())

	return cmd
}

// NewConfigRateListCommand lists available rate profiles
func NewConfigRateListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available rate profiles",
		Long:  "Show all available rate profiles with their settings.",
		RunE:  runConfigRateList,
	}
}

// NewConfigRateSetCommand sets the current rate profile
func NewConfigRateSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <profile-name>",
		Short: "Set current rate profile",
		Long: `Set the active rate profile. Available profiles:
- slow: Conservative scanning for stealth
- medium: Balanced scanning for general use
- fast: Aggressive scanning for speed  
- ludicrous: Maximum speed scanning`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigRateSet,
	}
}

// NewConfigRateCreateCommand creates a custom rate profile
func NewConfigRateCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create custom rate profile",
		Long:  "Create a new custom rate profile with specified parameters.",
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigRateCreate,
	}

	cmd.Flags().Int("rate", 100, "Packets per second")
	cmd.Flags().Int("concurrency", 100, "Number of concurrent workers")  
	cmd.Flags().Duration("timeout", 2*time.Second, "Per-operation timeout")
	cmd.Flags().Int("retries", 1, "Number of retry attempts")
	cmd.Flags().String("description", "", "Profile description")

	return cmd
}

// NewConfigRateDeleteCommand deletes a custom rate profile
func NewConfigRateDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <profile-name>",
		Short: "Delete custom rate profile",
		Long:  "Delete a custom rate profile. Built-in profiles cannot be deleted.",
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigRateDelete,
	}
}

// Command implementations

func runConfigShow(cmd *cobra.Command, args []string) error {
	cm, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	cm.PrintConfig()
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cm, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Parse value based on key
	var parsedValue interface{}
	switch key {
	case "output_format":
		parsedValue = value
	case "show_banners", "color_output", "verbose", "auto_confirm_dangerous":
		parsedValue, err = strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := cm.SetPreference(key, parsedValue); err != nil {
		return fmt.Errorf("failed to set preference: %w", err)
	}

	fmt.Printf("✅ Configuration updated: %s = %v\n", key, parsedValue)
	return nil
}

func runConfigRateList(cmd *cobra.Command, args []string) error {
	cm, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	current := cm.GetCurrentRateProfile()
	profiles := cm.GetAvailableProfiles()

	fmt.Printf("Rate Profiles\n")
	fmt.Printf("=============\n")
	fmt.Printf("Current profile: %s\n\n", current.Name)

	for name, profile := range profiles {
		status := ""
		if name == current.Name {
			status = " (current)"
		}

		fmt.Printf("• %s%s\n", name, status)
		fmt.Printf("  Description: %s\n", profile.Description)
		fmt.Printf("  Rate: %d pps | Concurrency: %d workers | Timeout: %v | Retries: %d\n\n",
			profile.Rate, profile.Concurrency, profile.Timeout, profile.Retries)
	}

	return nil
}

func runConfigRateSet(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	cm, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cm.SetCurrentRateProfile(profileName); err != nil {
		return fmt.Errorf("failed to set rate profile: %w", err)
	}

	profile := cm.GetCurrentRateProfile()
	fmt.Printf("✅ Rate profile set to: %s\n", profileName)
	fmt.Printf("Settings: %d pps, %d workers, %v timeout, %d retries\n",
		profile.Rate, profile.Concurrency, profile.Timeout, profile.Retries)

	return nil
}

func runConfigRateCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	rate, _ := cmd.Flags().GetInt("rate")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	retries, _ := cmd.Flags().GetInt("retries")
	description, _ := cmd.Flags().GetString("description")

	if description == "" {
		description = fmt.Sprintf("Custom profile: %d pps, %d workers", rate, concurrency)
	}

	profile := config.RateProfile{
		Name:        name,
		Description: description,
		Rate:        rate,
		Concurrency: concurrency,
		Timeout:     timeout,
		Retries:     retries,
	}

	cm, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cm.AddCustomProfile(name, profile); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	fmt.Printf("✅ Custom rate profile '%s' created\n", name)
	fmt.Printf("Settings: %d pps, %d workers, %v timeout, %d retries\n",
		rate, concurrency, timeout, retries)

	return nil
}

func runConfigRateDelete(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	cm, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cm.RemoveCustomProfile(profileName); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	fmt.Printf("✅ Custom rate profile '%s' deleted\n", profileName)
	return nil
}