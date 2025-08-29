package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RateProfile defines different speed presets for scanning
type RateProfile struct {
	Name        string        `yaml:"name" json:"name"`
	Description string        `yaml:"description" json:"description"`
	Rate        int           `yaml:"rate" json:"rate"`               // packets per second
	Concurrency int           `yaml:"concurrency" json:"concurrency"` // concurrent workers
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`         // per-operation timeout
	Retries     int           `yaml:"retries" json:"retries"`         // retry attempts
}

// Config represents the persistent NetCrate configuration
type Config struct {
	Version         string                 `yaml:"version" json:"version"`
	LastUpdated     time.Time              `yaml:"last_updated" json:"last_updated"`
	
	// Rate profile settings
	CurrentRateProfile string             `yaml:"current_rate_profile" json:"current_rate_profile"`
	RateProfiles       map[string]RateProfile `yaml:"rate_profiles" json:"rate_profiles"`
	
	// User preferences
	Preferences        UserPreferences    `yaml:"preferences" json:"preferences"`
	
	// Session settings
	Session            SessionConfig      `yaml:"session" json:"session"`
}

// UserPreferences stores user configuration choices
type UserPreferences struct {
	DefaultOutputFormat   string `yaml:"default_output_format" json:"default_output_format"`
	ShowBanners          bool   `yaml:"show_banners" json:"show_banners"`
	ColorOutput          bool   `yaml:"color_output" json:"color_output"`
	VerboseMode          bool   `yaml:"verbose_mode" json:"verbose_mode"`
	AutoConfirmDangerous bool   `yaml:"auto_confirm_dangerous" json:"auto_confirm_dangerous"`
}

// SessionConfig stores session-specific settings
type SessionConfig struct {
	LastTemplate     string            `yaml:"last_template" json:"last_template"`
	LastTargets      []string          `yaml:"last_targets" json:"last_targets"`
	RecentTargets    []string          `yaml:"recent_targets" json:"recent_targets"`
	CustomProfiles   map[string]RateProfile `yaml:"custom_profiles" json:"custom_profiles"`
}

// ConfigManager handles configuration persistence
type ConfigManager struct {
	configPath string
	config     *Config
}

// Default rate profiles
var DefaultRateProfiles = map[string]RateProfile{
	"slow": {
		Name:        "slow",
		Description: "Conservative scanning for stealth and stability",
		Rate:        50,    // 50 pps
		Concurrency: 50,    // 50 workers
		Timeout:     3 * time.Second,
		Retries:     3,
	},
	"medium": {
		Name:        "medium",
		Description: "Balanced scanning for general use",
		Rate:        200,   // 200 pps
		Concurrency: 200,   // 200 workers
		Timeout:     2 * time.Second,
		Retries:     2,
	},
	"fast": {
		Name:        "fast",
		Description: "Aggressive scanning for speed",
		Rate:        1000,  // 1000 pps
		Concurrency: 500,   // 500 workers
		Timeout:     1 * time.Second,
		Retries:     1,
	},
	"ludicrous": {
		Name:        "ludicrous",
		Description: "Maximum speed scanning (use with caution)",
		Rate:        5000,  // 5000 pps
		Concurrency: 1000,  // 1000 workers
		Timeout:     500 * time.Millisecond,
		Retries:     0,
	},
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".netcrate")
	configPath := filepath.Join(configDir, "config.json")
	
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	cm := &ConfigManager{
		configPath: configPath,
	}
	
	// Load existing config or create default
	if err := cm.load(); err != nil {
		// Create default config if load fails
		cm.config = cm.createDefaultConfig()
		if err := cm.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}
	
	return cm, nil
}

// load reads configuration from disk
func (cm *ConfigManager) load() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file does not exist")
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	cm.config = &config
	return nil
}

// Save writes configuration to disk
func (cm *ConfigManager) Save() error {
	cm.config.LastUpdated = time.Now()
	cm.config.Version = "1.0"
	
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// createDefaultConfig creates a default configuration
func (cm *ConfigManager) createDefaultConfig() *Config {
	return &Config{
		Version:            "1.0",
		LastUpdated:        time.Now(),
		CurrentRateProfile: "medium", // Default to medium speed
		RateProfiles:       DefaultRateProfiles,
		Preferences: UserPreferences{
			DefaultOutputFormat:   "table",
			ShowBanners:          true,
			ColorOutput:          true,
			VerboseMode:          false,
			AutoConfirmDangerous: false,
		},
		Session: SessionConfig{
			RecentTargets:  make([]string, 0),
			CustomProfiles: make(map[string]RateProfile),
		},
	}
}

// GetCurrentRateProfile returns the currently active rate profile
func (cm *ConfigManager) GetCurrentRateProfile() RateProfile {
	profile, exists := cm.config.RateProfiles[cm.config.CurrentRateProfile]
	if !exists {
		// Fallback to medium if current profile doesn't exist
		profile = DefaultRateProfiles["medium"]
	}
	return profile
}

// SetCurrentRateProfile sets the active rate profile and persists it
func (cm *ConfigManager) SetCurrentRateProfile(profileName string) error {
	// Check if profile exists
	if _, exists := cm.config.RateProfiles[profileName]; !exists {
		return fmt.Errorf("rate profile '%s' does not exist", profileName)
	}
	
	cm.config.CurrentRateProfile = profileName
	return cm.Save()
}

// GetAvailableProfiles returns all available rate profiles
func (cm *ConfigManager) GetAvailableProfiles() map[string]RateProfile {
	return cm.config.RateProfiles
}

// AddCustomProfile adds a custom rate profile
func (cm *ConfigManager) AddCustomProfile(name string, profile RateProfile) error {
	if cm.config.Session.CustomProfiles == nil {
		cm.config.Session.CustomProfiles = make(map[string]RateProfile)
	}
	
	profile.Name = name
	cm.config.Session.CustomProfiles[name] = profile
	cm.config.RateProfiles[name] = profile
	
	return cm.Save()
}

// RemoveCustomProfile removes a custom rate profile
func (cm *ConfigManager) RemoveCustomProfile(name string) error {
	// Don't allow removal of default profiles
	if _, isDefault := DefaultRateProfiles[name]; isDefault {
		return fmt.Errorf("cannot remove default profile '%s'", name)
	}
	
	// Check if profile exists in custom profiles
	if _, exists := cm.config.Session.CustomProfiles[name]; !exists {
		return fmt.Errorf("custom profile '%s' does not exist", name)
	}
	
	delete(cm.config.Session.CustomProfiles, name)
	delete(cm.config.RateProfiles, name)
	
	// If we're removing the current profile, switch to medium
	if cm.config.CurrentRateProfile == name {
		cm.config.CurrentRateProfile = "medium"
	}
	
	return cm.Save()
}

// GetConfig returns the full configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// SetPreference sets a user preference
func (cm *ConfigManager) SetPreference(key string, value interface{}) error {
	switch key {
	case "output_format":
		if str, ok := value.(string); ok {
			cm.config.Preferences.DefaultOutputFormat = str
		}
	case "show_banners":
		if b, ok := value.(bool); ok {
			cm.config.Preferences.ShowBanners = b
		}
	case "color_output":
		if b, ok := value.(bool); ok {
			cm.config.Preferences.ColorOutput = b
		}
	case "verbose":
		if b, ok := value.(bool); ok {
			cm.config.Preferences.VerboseMode = b
		}
	case "auto_confirm_dangerous":
		if b, ok := value.(bool); ok {
			cm.config.Preferences.AutoConfirmDangerous = b
		}
	default:
		return fmt.Errorf("unknown preference: %s", key)
	}
	
	return cm.Save()
}

// AddRecentTarget adds a target to the recent targets list
func (cm *ConfigManager) AddRecentTarget(target string) error {
	// Remove target if it already exists
	for i, t := range cm.config.Session.RecentTargets {
		if t == target {
			cm.config.Session.RecentTargets = append(
				cm.config.Session.RecentTargets[:i],
				cm.config.Session.RecentTargets[i+1:]...)
			break
		}
	}
	
	// Add to front
	cm.config.Session.RecentTargets = append([]string{target}, cm.config.Session.RecentTargets...)
	
	// Keep only last 10
	if len(cm.config.Session.RecentTargets) > 10 {
		cm.config.Session.RecentTargets = cm.config.Session.RecentTargets[:10]
	}
	
	return cm.Save()
}

// GetRecentTargets returns the list of recent targets
func (cm *ConfigManager) GetRecentTargets() []string {
	return cm.config.Session.RecentTargets
}

// SetLastTemplate stores the last used template
func (cm *ConfigManager) SetLastTemplate(template string) error {
	cm.config.Session.LastTemplate = template
	return cm.Save()
}

// GetLastTemplate returns the last used template
func (cm *ConfigManager) GetLastTemplate() string {
	return cm.config.Session.LastTemplate
}

// PrintConfig prints the current configuration in a user-friendly format
func (cm *ConfigManager) PrintConfig() {
	fmt.Printf("NetCrate Configuration\n")
	fmt.Printf("======================\n")
	fmt.Printf("Config file: %s\n", cm.configPath)
	fmt.Printf("Version: %s\n", cm.config.Version)
	fmt.Printf("Last updated: %s\n\n", cm.config.LastUpdated.Format("2006-01-02 15:04:05"))
	
	fmt.Printf("Rate Profiles:\n")
	fmt.Printf("--------------\n")
	fmt.Printf("Current profile: %s\n\n", cm.config.CurrentRateProfile)
	
	current := cm.GetCurrentRateProfile()
	fmt.Printf("Active Settings:\n")
	fmt.Printf("  • Rate: %d packets/second\n", current.Rate)
	fmt.Printf("  • Concurrency: %d workers\n", current.Concurrency)
	fmt.Printf("  • Timeout: %v per operation\n", current.Timeout)
	fmt.Printf("  • Retries: %d attempts\n\n", current.Retries)
	
	fmt.Printf("Available Profiles:\n")
	for name, profile := range cm.config.RateProfiles {
		status := ""
		if name == cm.config.CurrentRateProfile {
			status = " (current)"
		}
		fmt.Printf("  • %s%s: %s\n", name, status, profile.Description)
		fmt.Printf("    Rate: %d pps, Concurrency: %d, Timeout: %v\n", 
			profile.Rate, profile.Concurrency, profile.Timeout)
	}
	
	fmt.Printf("\nPreferences:\n")
	fmt.Printf("------------\n")
	fmt.Printf("  • Output format: %s\n", cm.config.Preferences.DefaultOutputFormat)
	fmt.Printf("  • Show banners: %v\n", cm.config.Preferences.ShowBanners)
	fmt.Printf("  • Color output: %v\n", cm.config.Preferences.ColorOutput)
	fmt.Printf("  • Verbose mode: %v\n", cm.config.Preferences.VerboseMode)
	fmt.Printf("  • Auto-confirm dangerous: %v\n", cm.config.Preferences.AutoConfirmDangerous)
	
	if len(cm.config.Session.RecentTargets) > 0 {
		fmt.Printf("\nRecent Targets:\n")
		fmt.Printf("---------------\n")
		for i, target := range cm.config.Session.RecentTargets {
			fmt.Printf("  %d. %s\n", i+1, target)
		}
	}
	
	if cm.config.Session.LastTemplate != "" {
		fmt.Printf("\nLast Template: %s\n", cm.config.Session.LastTemplate)
	}
}