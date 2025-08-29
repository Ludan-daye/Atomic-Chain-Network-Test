package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Template represents a NetCrate template
type Template struct {
	Name            string                 `yaml:"name" json:"name"`
	Version         string                 `yaml:"version" json:"version"`
	Description     string                 `yaml:"description" json:"description"`
	Author          string                 `yaml:"author" json:"author"`
	Tags            []string               `yaml:"tags" json:"tags"`
	RequireDangerous bool                  `yaml:"require_dangerous" json:"require_dangerous"`
	Parameters      []TemplateParameter    `yaml:"parameters" json:"parameters"`
	Steps           []TemplateStep         `yaml:"steps" json:"steps"`
	
	// Runtime metadata
	Path     string    `yaml:"-" json:"path"`
	Source   string    `yaml:"-" json:"source"` // "user", "builtin", "env"
	LoadTime time.Time `yaml:"-" json:"load_time"`
}

// TemplateParameter defines a parameter for the template
type TemplateParameter struct {
	Name        string      `yaml:"name" json:"name"`
	Description string      `yaml:"description" json:"description"`
	Type        string      `yaml:"type" json:"type"` // string, int, bool, duration, cidr, ports, endpoint, list<string>
	Required    bool        `yaml:"required" json:"required"`
	Default     interface{} `yaml:"default" json:"default"`
	Validation  string      `yaml:"validation" json:"validation"`
}

// TemplateStep defines a step in the template execution
type TemplateStep struct {
	Name      string                 `yaml:"name" json:"name"`
	Operation string                 `yaml:"operation" json:"operation"`
	With      map[string]interface{} `yaml:"with" json:"with"`
	DependsOn string                 `yaml:"depends_on" json:"depends_on"`
	OnEmpty   string                 `yaml:"on_empty" json:"on_empty"`
	OnError   string                 `yaml:"on_error" json:"on_error"` // continue, skip, fail (default)
}

// Registry manages template discovery and caching
type Registry struct {
	searchPaths    []string
	templates      map[string]*Template
	indexPath      string
	lastIndexTime  time.Time
}

// NewRegistry creates a new template registry
func NewRegistry() *Registry {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".netcrate", "cache")
	os.MkdirAll(cacheDir, 0755)
	
	registry := &Registry{
		templates: make(map[string]*Template),
		indexPath: filepath.Join(cacheDir, "templates.index.json"),
	}
	
	// Setup search paths with priority order
	registry.setupSearchPaths()
	
	return registry
}

// setupSearchPaths configures template search paths in priority order
func (r *Registry) setupSearchPaths() {
	homeDir, _ := os.UserHomeDir()
	
	// Priority 1: User directory ~/.netcrate/templates/
	userTemplatesDir := filepath.Join(homeDir, ".netcrate", "templates")
	if _, err := os.Stat(userTemplatesDir); err == nil {
		r.searchPaths = append(r.searchPaths, userTemplatesDir)
	}
	
	// Priority 2: Environment variable NETCRATE_TEMPLATES
	if envPaths := os.Getenv("NETCRATE_TEMPLATES"); envPaths != "" {
		for _, path := range strings.Split(envPaths, ":") {
			path = strings.TrimSpace(path)
			if path != "" && filepath.IsAbs(path) {
				if _, err := os.Stat(path); err == nil {
					r.searchPaths = append(r.searchPaths, path)
				}
			}
		}
	}
	
	// Priority 3: Project builtin templates/builtin/
	builtinPath := filepath.Join("templates", "builtin")
	if _, err := os.Stat(builtinPath); err == nil {
		r.searchPaths = append(r.searchPaths, builtinPath)
	}
}

// LoadTemplates loads templates from all search paths
func (r *Registry) LoadTemplates() error {
	// Try to load from cache first
	if r.loadFromCache() {
		return nil
	}
	
	// Clear existing templates
	r.templates = make(map[string]*Template)
	
	// Load from each search path
	for i, searchPath := range r.searchPaths {
		source := r.getSourceName(i, searchPath)
		err := r.loadFromPath(searchPath, source)
		if err != nil {
			fmt.Printf("[WARN] Failed to load templates from %s: %v\n", searchPath, err)
		}
	}
	
	// Save to cache
	r.saveToCache()
	
	return nil
}

// loadFromCache attempts to load templates from cache
func (r *Registry) loadFromCache() bool {
	if _, err := os.Stat(r.indexPath); os.IsNotExist(err) {
		return false
	}
	
	// Check if any search path is newer than cache
	stat, err := os.Stat(r.indexPath)
	if err != nil {
		return false
	}
	
	cacheTime := stat.ModTime()
	for _, searchPath := range r.searchPaths {
		if r.isPathNewer(searchPath, cacheTime) {
			return false
		}
	}
	
	// Load from cache
	data, err := os.ReadFile(r.indexPath)
	if err != nil {
		return false
	}
	
	var templates map[string]*Template
	if err := json.Unmarshal(data, &templates); err != nil {
		return false
	}
	
	r.templates = templates
	r.lastIndexTime = cacheTime
	return true
}

// isPathNewer checks if any file in path is newer than the reference time
func (r *Registry) isPathNewer(searchPath string, refTime time.Time) bool {
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
			if info.ModTime().After(refTime) {
				return fmt.Errorf("newer file found") // Use error to break walk
			}
		}
		return nil
	})
	return err != nil
}

// saveToCache saves templates to cache
func (r *Registry) saveToCache() {
	data, err := json.MarshalIndent(r.templates, "", "  ")
	if err != nil {
		return
	}
	
	os.WriteFile(r.indexPath, data, 0644)
	r.lastIndexTime = time.Now()
}

// loadFromPath loads templates from a specific path
func (r *Registry) loadFromPath(searchPath, source string) error {
	return filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			template, err := r.loadTemplate(path, source)
			if err != nil {
				fmt.Printf("[WARN] Failed to load template %s: %v\n", path, err)
				return nil // Continue walking
			}
			
			// User templates override builtin ones with same name
			if existing, exists := r.templates[template.Name]; exists {
				if source == "user" || (source == "env" && existing.Source != "user") {
					fmt.Printf("[INFO] Template %s: %s overrides %s\n", template.Name, source, existing.Source)
					r.templates[template.Name] = template
				}
			} else {
				r.templates[template.Name] = template
			}
		}
		
		return nil
	})
}

// loadTemplate loads a single template from file
func (r *Registry) loadTemplate(filePath, source string) (*Template, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var template Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, err
	}
	
	template.Path = filePath
	template.Source = source
	template.LoadTime = time.Now()
	
	return &template, nil
}

// getSourceName determines the source name for a search path
func (r *Registry) getSourceName(index int, path string) string {
	homeDir, _ := os.UserHomeDir()
	userTemplatesDir := filepath.Join(homeDir, ".netcrate", "templates")
	
	if path == userTemplatesDir {
		return "user"
	} else if strings.Contains(path, "builtin") {
		return "builtin"
	} else {
		return "env"
	}
}

// List returns all loaded templates
func (r *Registry) List() []*Template {
	var templates []*Template
	for _, template := range r.templates {
		templates = append(templates, template)
	}
	
	// Sort by name
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	
	return templates
}

// Get retrieves a template by name
func (r *Registry) Get(name string) (*Template, bool) {
	template, exists := r.templates[name]
	return template, exists
}

// PrintIndex prints debug information about the registry
func (r *Registry) PrintIndex() {
	fmt.Printf("Template Registry Index\n")
	fmt.Printf("=======================\n\n")
	
	fmt.Printf("Search Paths (%d):\n", len(r.searchPaths))
	for i, path := range r.searchPaths {
		source := r.getSourceName(i, path)
		fmt.Printf("  %d. %s (%s)\n", i+1, path, source)
	}
	fmt.Printf("\n")
	
	fmt.Printf("Loaded Templates (%d):\n", len(r.templates))
	for name, template := range r.templates {
		fmt.Printf("  • %s v%s (%s) - %s\n", name, template.Version, template.Source, template.Description)
	}
	fmt.Printf("\n")
	
	fmt.Printf("Cache: %s\n", r.indexPath)
	if !r.lastIndexTime.IsZero() {
		fmt.Printf("Last indexed: %s\n", r.lastIndexTime.Format("2006-01-02 15:04:05"))
	}
}

// EnsureUserTemplateDir creates the user template directory if it doesn't exist
func EnsureUserTemplateDir() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	userTemplatesDir := filepath.Join(homeDir, ".netcrate", "templates")
	return os.MkdirAll(userTemplatesDir, 0755)
}