# NetCrate Templates

This directory contains template files for common NetCrate scanning workflows. Templates allow you to create reusable scanning configurations with parameters for different scenarios.

## 📋 Available Templates

### Built-in Templates (`builtin/`)

| Template | Description | Use Case | Requires --dangerous |
|----------|-------------|----------|---------------------|
| `basic_scan.yaml` | Basic network and port scanning | General network assessment | No |

### Example Templates (`examples/`)

| Template | Description | Use Case | Requires --dangerous |
|----------|-------------|----------|---------------------|
| `network_discovery.yaml` | Network discovery and asset enumeration | Finding live hosts and basic services | No |
| `web_application_scan.yaml` | Web application security assessment | Testing web apps and services | Yes (if public) |
| `security_audit.yaml` | Comprehensive security audit | Internal network security review | No |
| `quick_recon.yaml` | Quick reconnaissance | Fast initial assessment | Depends on target |

## 🚀 Quick Start

### List Available Templates

```bash
netcrate templates list
```

### Run a Template

```bash
# Basic network scan
netcrate templates run basic_scan --targets 192.168.1.0/24

# Web application scan (requires --dangerous for public targets)
netcrate templates run web_application_scan --target https://example.com --dangerous

# Quick reconnaissance
netcrate templates run quick_recon --target 192.168.1.1 --speed fast
```

### Get Template Help

```bash
# Show template parameters and usage
netcrate templates run basic_scan --help
netcrate templates run web_application_scan --help
```

## 📝 Template Format

Templates use YAML format with Jinja2-style templating. Here's the basic structure:

```yaml
name: "template_name"
version: "1.0"
description: "Template description"
author: "Your Name"
tags: ["tag1", "tag2"]
require_dangerous: false  # Set to true if template may scan public networks

parameters:
  - name: "param_name"
    description: "Parameter description"
    type: "string"  # string, int, bool, cidr, ports, endpoint, list<string>
    required: true
    default: "default_value"  # Optional
    validation: "regex_pattern"  # Optional

steps:
  - name: "step_name"
    operation: "discover"  # discover, scan_ports, banner_grab, fingerprint
    with:
      targets: ["{{ .param_name }}"]
      # Operation-specific parameters
    depends_on: "previous_step"  # Optional
    on_empty: "continue"  # continue, fail, skip
    on_error: "continue"  # continue, fail, skip
```

## 🛠️ Template Operations

### Available Operations

| Operation | Description | Parameters |
|-----------|-------------|------------|
| `discover` | Host discovery | `targets`, `methods`, `timeout`, `concurrency` |
| `scan_ports` | Port scanning | `targets`, `ports`, `scan_type`, `service_detection` |
| `banner_grab` | Banner grabbing | `targets`, `ports`, `protocols`, `timeout` |
| `fingerprint` | Service fingerprinting | `targets`, `services`, `deep_scan` |

### Parameter Types

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text value | `"192.168.1.1"` |
| `int` | Integer value | `80` |
| `bool` | Boolean value | `true` |
| `cidr` | Network range | `"192.168.1.0/24"` |
| `ports` | Port specification | `"80,443,8000-8999"` |
| `endpoint` | URL or hostname | `"https://example.com"` |
| `list<string>` | Array of strings | `["host1", "host2"]` |

## 📚 Template Examples

### Example 1: Simple Discovery Template

```yaml
name: "simple_discovery"
version: "1.0" 
description: "Simple host discovery template"
author: "NetCrate User"
tags: ["discovery", "simple"]
require_dangerous: false

parameters:
  - name: "network"
    description: "Target network in CIDR notation"
    type: "cidr"
    required: true

steps:
  - name: "discover_hosts"
    operation: "discover"
    with:
      targets: ["{{ .network }}"]
      methods: ["ping"]
      timeout: "2s"
```

### Example 2: Multi-step Scanning Template

```yaml
name: "comprehensive_scan"
version: "1.0"
description: "Comprehensive scanning with multiple phases"
author: "NetCrate User"  
tags: ["comprehensive", "multi-phase"]
require_dangerous: false

parameters:
  - name: "target"
    description: "Target host or network"
    type: "string"
    required: true
  - name: "deep_scan"
    description: "Enable deep scanning"
    type: "bool"
    required: false
    default: false

steps:
  - name: "discovery"
    operation: "discover"
    with:
      targets: ["{{ .target }}"]
      methods: ["ping", "tcp"]
    on_empty: "fail"

  - name: "port_scan"
    operation: "scan_ports"
    with:
      targets: "{{ .discovery.live_hosts }}"
      ports: "{{ if .deep_scan }}top1000{{ else }}top100{{ end }}"
      scan_type: "connect"
      service_detection: true
    depends_on: "discovery"

  - name: "banner_grab"
    operation: "banner_grab"
    with:
      targets: "{{ .discovery.live_hosts }}"
      ports: "{{ .port_scan.open_ports }}"
    depends_on: "port_scan"
    on_error: "continue"
```

## 🔧 Creating Custom Templates

### Step 1: Plan Your Template

- Define the scanning workflow
- Identify required parameters
- Plan step dependencies
- Consider error handling

### Step 2: Create Template File

```bash
# Create new template file
touch my_custom_template.yaml

# Edit with your preferred editor
nano my_custom_template.yaml
```

### Step 3: Validate Template

```bash
# Check template syntax
netcrate templates validate my_custom_template.yaml

# Test with dry run (if supported)
netcrate templates run my_custom_template.yaml --dry-run
```

### Step 4: Test Template

```bash
# Run template with test parameters
netcrate templates run my_custom_template.yaml --param1 value1 --param2 value2

# Check results
netcrate output show --last
```

## 🎯 Best Practices

### Template Design

1. **Keep it focused**: One template per specific use case
2. **Use clear names**: Descriptive names for templates, parameters, and steps
3. **Add documentation**: Good descriptions and examples
4. **Handle errors**: Use `on_error` and `on_empty` appropriately
5. **Consider security**: Set `require_dangerous: true` if scanning public networks

### Parameter Design

1. **Required vs optional**: Mark parameters appropriately
2. **Sensible defaults**: Provide good default values
3. **Input validation**: Use regex validation for complex inputs
4. **Clear descriptions**: Help users understand what each parameter does

### Step Design

1. **Logical flow**: Order steps in logical sequence
2. **Use dependencies**: Ensure steps run in correct order
3. **Error handling**: Decide how to handle failures
4. **Resource efficiency**: Don't scan unnecessarily

## 🔍 Template Validation

### Syntax Validation

```bash
# Validate YAML syntax and template structure
netcrate templates validate template.yaml
```

### Common Validation Errors

- **Missing required fields**: `name`, `version`, `description`
- **Invalid parameter types**: Use supported types only
- **Circular dependencies**: Step A depends on B, B depends on A
- **Invalid operations**: Use supported operations only
- **Malformed templating**: Check Jinja2 syntax

### Testing Templates

```bash
# Test with minimal parameters
netcrate templates run template.yaml --help

# Test execution flow
netcrate templates run template.yaml --param value --dry-run

# Full test run
netcrate templates run template.yaml --param value --verbose
```

## 📦 Template Distribution

### Sharing Templates

1. **Local**: Save in current directory, NetCrate will find them
2. **User templates**: Save in `~/.netcrate/templates/`
3. **System templates**: Share via GitHub, package managers
4. **Team sharing**: Version control with your team

### Template Locations

NetCrate searches for templates in:

1. Current working directory (`./`)
2. Current directory templates (`./templates/`)
3. User template directory (`~/.netcrate/templates/`)
4. Built-in templates (included with NetCrate)

## 🤝 Contributing Templates

Want to contribute templates to the NetCrate project?

1. **Fork** the NetCrate repository
2. **Create** your template in `templates/examples/`
3. **Test** thoroughly with various parameters
4. **Document** in this README
5. **Submit** a pull request

### Template Submission Guidelines

- Follow the established format and style
- Include comprehensive parameter validation
- Add clear descriptions and examples
- Test on multiple environments
- Consider security implications

---

For more information, see the [User Guide](../docs/USER_GUIDE.md) and [Examples](../docs/EXAMPLES.md).