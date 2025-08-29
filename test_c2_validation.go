package main

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Template parameter types from the specification
type TemplateParameter struct {
	Name        string      `yaml:"name" json:"name"`
	Description string      `yaml:"description" json:"description"`
	Type        string      `yaml:"type" json:"type"` // string, int, bool, duration, cidr, ports, endpoint, list<string>
	Required    bool        `yaml:"required" json:"required"`
	Default     interface{} `yaml:"default" json:"default"`
	Validation  string      `yaml:"validation" json:"validation"`
}

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string
	Value     interface{}
	Message   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("parameter '%s' validation failed: %s (value: %v)", e.Parameter, e.Message, e.Value)
}

// ValidatorFunc defines the signature for parameter validators
type ValidatorFunc func(value interface{}, param TemplateParameter) error

// ParameterValidator handles validation of template parameters
type ParameterValidator struct {
	validators map[string]ValidatorFunc
}

// NewParameterValidator creates a new parameter validator
func NewParameterValidator() *ParameterValidator {
	validator := &ParameterValidator{
		validators: make(map[string]ValidatorFunc),
	}
	
	// Register built-in validators
	validator.RegisterValidator("cidr", validateCIDR)
	validator.RegisterValidator("port_range", validatePortRange)
	validator.RegisterValidator("endpoint", validateEndpoint)
	validator.RegisterValidator("duration", validateDuration)
	validator.RegisterValidator("int", validateInteger)
	validator.RegisterValidator("bool", validateBoolean)
	validator.RegisterValidator("string", validateString)
	
	return validator
}

// RegisterValidator registers a custom validator
func (v *ParameterValidator) RegisterValidator(name string, validator ValidatorFunc) {
	v.validators[name] = validator
}

// ValidateParameter validates a single parameter value
func (v *ParameterValidator) ValidateParameter(param TemplateParameter, value interface{}) error {
	// First validate by type
	if err := v.validateByType(param, value); err != nil {
		return err
	}
	
	// Then validate by specific validation rule if present
	if param.Validation != "" {
		if validator, exists := v.validators[param.Validation]; exists {
			if err := validator(value, param); err != nil {
				return ValidationError{
					Parameter: param.Name,
					Value:     value,
					Message:   err.Error(),
				}
			}
		} else {
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   fmt.Sprintf("unknown validation rule: %s", param.Validation),
			}
		}
	}
	
	return nil
}

// validateByType validates parameter by its declared type
func (v *ParameterValidator) validateByType(param TemplateParameter, value interface{}) error {
	switch param.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   "must be a string",
			}
		}
	case "int":
		switch val := value.(type) {
		case int, int32, int64:
			// Valid
		case float64:
			// JSON unmarshaling produces float64 for numbers
			if val != float64(int(val)) {
				return ValidationError{
					Parameter: param.Name,
					Value:     value,
					Message:   "must be an integer",
				}
			}
		default:
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   "must be an integer",
			}
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   "must be a boolean",
			}
		}
	case "duration":
		if _, err := parseDuration(value); err != nil {
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   "must be a valid duration (e.g., '5s', '10m', '1h')",
			}
		}
	case "cidr":
		if err := validateCIDR(value, param); err != nil {
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   err.Error(),
			}
		}
	case "ports":
		if err := validatePortRange(value, param); err != nil {
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   err.Error(),
			}
		}
	case "endpoint":
		if err := validateEndpoint(value, param); err != nil {
			return ValidationError{
				Parameter: param.Name,
				Value:     value,
				Message:   err.Error(),
			}
		}
	default:
		// Handle list types
		if strings.HasPrefix(param.Type, "list<") && strings.HasSuffix(param.Type, ">") {
			innerType := strings.TrimSuffix(strings.TrimPrefix(param.Type, "list<"), ">")
			if err := v.validateList(param, value, innerType); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// validateList validates list-type parameters
func (v *ParameterValidator) validateList(param TemplateParameter, value interface{}, innerType string) error {
	// Convert to slice
	var items []interface{}
	switch val := value.(type) {
	case []interface{}:
		items = val
	case []string:
		for _, item := range val {
			items = append(items, item)
		}
	default:
		return ValidationError{
			Parameter: param.Name,
			Value:     value,
			Message:   "must be a list",
		}
	}
	
	// Validate each item
	for i, item := range items {
		dummyParam := TemplateParameter{
			Name: fmt.Sprintf("%s[%d]", param.Name, i),
			Type: innerType,
		}
		if err := v.validateByType(dummyParam, item); err != nil {
			return err
		}
	}
	
	return nil
}

// Built-in validators

// validateCIDR validates CIDR notation
func validateCIDR(value interface{}, param TemplateParameter) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}
	
	// Handle special cases
	if str == "auto" {
		return nil // Special case for auto-detection
	}
	
	if _, _, err := net.ParseCIDR(str); err != nil {
		return fmt.Errorf("must be valid CIDR notation (e.g., '192.168.1.0/24')")
	}
	
	return nil
}

// validatePortRange validates port range specification
func validatePortRange(value interface{}, param TemplateParameter) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}
	
	// Handle special cases
	switch str {
	case "top100", "top1000", "all":
		return nil
	}
	
	// Parse port range: single port, comma-separated, or ranges
	parts := strings.Split(str, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		
		if strings.Contains(part, "-") {
			// Range format: "80-443"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return fmt.Errorf("invalid port range format: %s", part)
			}
			
			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil || start < 1 || start > 65535 {
				return fmt.Errorf("invalid start port: %s", rangeParts[0])
			}
			
			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil || end < 1 || end > 65535 {
				return fmt.Errorf("invalid end port: %s", rangeParts[1])
			}
			
			if start >= end {
				return fmt.Errorf("start port must be less than end port: %s", part)
			}
		} else {
			// Single port
			port, err := strconv.Atoi(part)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("invalid port: %s", part)
			}
		}
	}
	
	return nil
}

// validateEndpoint validates endpoint format (host:port)
func validateEndpoint(value interface{}, param TemplateParameter) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}
	
	// Split host and port
	host, portStr, err := net.SplitHostPort(str)
	if err != nil {
		return fmt.Errorf("must be in format 'host:port'")
	}
	
	// Validate host (can be IP or hostname)
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	
	// Validate port
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %s", portStr)
	}
	
	return nil
}

// validateDuration validates duration format
func validateDuration(value interface{}, param TemplateParameter) error {
	_, err := parseDuration(value)
	return err
}

// validateInteger validates integer values
func validateInteger(value interface{}, param TemplateParameter) error {
	switch val := value.(type) {
	case int, int32, int64:
		return nil
	case float64:
		if val != float64(int(val)) {
			return fmt.Errorf("must be an integer")
		}
		return nil
	default:
		return fmt.Errorf("must be an integer")
	}
}

// validateBoolean validates boolean values
func validateBoolean(value interface{}, param TemplateParameter) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("must be a boolean")
	}
	return nil
}

// validateString validates string values
func validateString(value interface{}, param TemplateParameter) error {
	if _, ok := value.(string); !ok {
		return fmt.Errorf("must be a string")
	}
	return nil
}

// Helper functions

// parseDuration parses duration from various formats
func parseDuration(value interface{}) (time.Duration, error) {
	switch val := value.(type) {
	case string:
		// Try Go duration format first
		if d, err := time.ParseDuration(val); err == nil {
			return d, nil
		}
		
		// Try common formats
		re := regexp.MustCompile(`^(\d+)(ms|s|m|h)$`)
		matches := re.FindStringSubmatch(val)
		if len(matches) == 3 {
			num, _ := strconv.Atoi(matches[1])
			switch matches[2] {
			case "ms":
				return time.Duration(num) * time.Millisecond, nil
			case "s":
				return time.Duration(num) * time.Second, nil
			case "m":
				return time.Duration(num) * time.Minute, nil
			case "h":
				return time.Duration(num) * time.Hour, nil
			}
		}
		
		return 0, fmt.Errorf("invalid duration format: %s", val)
	case int:
		// Assume milliseconds
		return time.Duration(val) * time.Millisecond, nil
	case float64:
		// Assume milliseconds
		return time.Duration(val) * time.Millisecond, nil
	default:
		return 0, fmt.Errorf("duration must be string or number")
	}
}

// Test Cases
func main() {
	fmt.Println("NetCrate Template Parameter Validation (C2) Test")
	fmt.Println("=================================================\n")
	
	validator := NewParameterValidator()
	
	// Test cases for each type and validation rule
	testCases := []struct {
		name      string
		param     TemplateParameter
		value     interface{}
		expectErr bool
		desc      string
	}{
		// String type tests
		{
			name: "string_valid",
			param: TemplateParameter{Name: "test", Type: "string"},
			value: "hello",
			expectErr: false,
			desc: "Valid string parameter",
		},
		{
			name: "string_invalid",
			param: TemplateParameter{Name: "test", Type: "string"},
			value: 123,
			expectErr: true,
			desc: "Invalid string parameter (number)",
		},
		
		// Integer type tests
		{
			name: "int_valid",
			param: TemplateParameter{Name: "test", Type: "int"},
			value: 42,
			expectErr: false,
			desc: "Valid integer parameter",
		},
		{
			name: "int_float_valid",
			param: TemplateParameter{Name: "test", Type: "int"},
			value: 42.0,
			expectErr: false,
			desc: "Valid integer as float64",
		},
		{
			name: "int_invalid",
			param: TemplateParameter{Name: "test", Type: "int"},
			value: 42.5,
			expectErr: true,
			desc: "Invalid integer (decimal)",
		},
		
		// Boolean type tests
		{
			name: "bool_valid",
			param: TemplateParameter{Name: "test", Type: "bool"},
			value: true,
			expectErr: false,
			desc: "Valid boolean parameter",
		},
		{
			name: "bool_invalid",
			param: TemplateParameter{Name: "test", Type: "bool"},
			value: "true",
			expectErr: true,
			desc: "Invalid boolean parameter (string)",
		},
		
		// Duration type tests
		{
			name: "duration_valid_string",
			param: TemplateParameter{Name: "test", Type: "duration"},
			value: "5s",
			expectErr: false,
			desc: "Valid duration string",
		},
		{
			name: "duration_valid_int",
			param: TemplateParameter{Name: "test", Type: "duration"},
			value: 1000,
			expectErr: false,
			desc: "Valid duration integer (ms)",
		},
		{
			name: "duration_invalid",
			param: TemplateParameter{Name: "test", Type: "duration"},
			value: "invalid",
			expectErr: true,
			desc: "Invalid duration string",
		},
		
		// CIDR validation tests
		{
			name: "cidr_valid",
			param: TemplateParameter{Name: "test", Type: "cidr", Validation: "cidr"},
			value: "192.168.1.0/24",
			expectErr: false,
			desc: "Valid CIDR notation",
		},
		{
			name: "cidr_auto",
			param: TemplateParameter{Name: "test", Type: "cidr", Validation: "cidr"},
			value: "auto",
			expectErr: false,
			desc: "Valid CIDR auto value",
		},
		{
			name: "cidr_invalid",
			param: TemplateParameter{Name: "test", Type: "cidr", Validation: "cidr"},
			value: "192.168.1.0/33",
			expectErr: true,
			desc: "Invalid CIDR notation",
		},
		
		// Port range validation tests
		{
			name: "ports_single",
			param: TemplateParameter{Name: "test", Type: "ports", Validation: "port_range"},
			value: "80",
			expectErr: false,
			desc: "Valid single port",
		},
		{
			name: "ports_range",
			param: TemplateParameter{Name: "test", Type: "ports", Validation: "port_range"},
			value: "80-443",
			expectErr: false,
			desc: "Valid port range",
		},
		{
			name: "ports_list",
			param: TemplateParameter{Name: "test", Type: "ports", Validation: "port_range"},
			value: "22,80,443,8080",
			expectErr: false,
			desc: "Valid port list",
		},
		{
			name: "ports_top100",
			param: TemplateParameter{Name: "test", Type: "ports", Validation: "port_range"},
			value: "top100",
			expectErr: false,
			desc: "Valid top100 ports",
		},
		{
			name: "ports_invalid",
			param: TemplateParameter{Name: "test", Type: "ports", Validation: "port_range"},
			value: "70000",
			expectErr: true,
			desc: "Invalid port (out of range)",
		},
		
		// Endpoint validation tests
		{
			name: "endpoint_valid",
			param: TemplateParameter{Name: "test", Type: "endpoint", Validation: "endpoint"},
			value: "localhost:8080",
			expectErr: false,
			desc: "Valid endpoint",
		},
		{
			name: "endpoint_ip",
			param: TemplateParameter{Name: "test", Type: "endpoint", Validation: "endpoint"},
			value: "192.168.1.1:80",
			expectErr: false,
			desc: "Valid IP endpoint",
		},
		{
			name: "endpoint_invalid",
			param: TemplateParameter{Name: "test", Type: "endpoint", Validation: "endpoint"},
			value: "invalid_endpoint",
			expectErr: true,
			desc: "Invalid endpoint format",
		},
		
		// List type tests
		{
			name: "list_string_valid",
			param: TemplateParameter{Name: "test", Type: "list<string>"},
			value: []interface{}{"a", "b", "c"},
			expectErr: false,
			desc: "Valid string list",
		},
		{
			name: "list_string_invalid",
			param: TemplateParameter{Name: "test", Type: "list<string>"},
			value: []interface{}{"a", 123, "c"},
			expectErr: true,
			desc: "Invalid string list (contains number)",
		},
	}
	
	// Run tests
	passed := 0
	failed := 0
	
	for _, tc := range testCases {
		err := validator.ValidateParameter(tc.param, tc.value)
		
		if (err != nil) != tc.expectErr {
			failed++
			if tc.expectErr {
				fmt.Printf("❌ %s: Expected error but got none - %s\n", tc.name, tc.desc)
			} else {
				fmt.Printf("❌ %s: Unexpected error: %v - %s\n", tc.name, err, tc.desc)
			}
		} else {
			passed++
			fmt.Printf("✅ %s: %s\n", tc.name, tc.desc)
		}
	}
	
	fmt.Printf("\nC2 Validation Test Results:\n")
	fmt.Printf("===========================\n")
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total:  %d\n", passed+failed)
	
	if failed == 0 {
		fmt.Printf("\n🎉 All C2 parameter validation tests passed!\n")
		fmt.Printf("C2 DoD achieved: ✅ 支持 string/int/bool/duration/cidr/ports/endpoint/list<string> 类型\n")
	} else {
		fmt.Printf("\n⚠️  Some tests failed. Review implementation.\n")
	}
}