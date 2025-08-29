package templates

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParameterValidator handles validation of template parameters
type ParameterValidator struct {
	validators map[string]ValidatorFunc
}

// ValidatorFunc defines the signature for parameter validators
type ValidatorFunc func(value interface{}, param TemplateParameter) error

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string
	Value     interface{}
	Message   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("parameter '%s' validation failed: %s (value: %v)", e.Parameter, e.Message, e.Value)
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

// ValidateTemplate validates all parameters in a template
func (v *ParameterValidator) ValidateTemplate(template *Template, parameters map[string]interface{}) []error {
	var errors []error
	
	// Check required parameters
	for _, param := range template.Parameters {
		value, exists := parameters[param.Name]
		
		if !exists {
			if param.Required {
				errors = append(errors, ValidationError{
					Parameter: param.Name,
					Value:     nil,
					Message:   "required parameter missing",
				})
			} else if param.Default != nil {
				// Use default value
				parameters[param.Name] = param.Default
				value = param.Default
			} else {
				continue // Optional parameter not provided
			}
		}
		
		// Validate the parameter
		if err := v.ValidateParameter(param, value); err != nil {
			errors = append(errors, err)
		}
	}
	
	return errors
}