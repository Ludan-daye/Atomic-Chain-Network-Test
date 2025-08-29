# NetCrate User Guide

Welcome to NetCrate! This comprehensive guide will help you get started with network security testing using NetCrate's powerful toolkit.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Basic Commands](#basic-commands)
- [Configuration Management](#configuration-management)
- [Templates and Workflows](#templates-and-workflows)
- [Security and Compliance](#security-and-compliance)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)
- [Examples](#examples)

## 🚀 Quick Start

### Installation

Choose your preferred installation method:

**Homebrew (Recommended for macOS/Linux):**
```bash
brew tap netcrate/tap
brew install netcrate
```

**Direct Binary Download:**
1. Visit [GitHub Releases](https://github.com/netcrate/netcrate/releases)
2. Download the appropriate binary for your platform
3. Extract and move to your PATH

**From Source:**
```bash
git clone https://github.com/netcrate/netcrate.git
cd netcrate
make build
make install
```

### Your First Scan

Start with NetCrate's intelligent quick mode:

```bash
# Auto-detect your network and perform a quick scan
netcrate quick

# Or scan a specific network (requires --dangerous for public networks)
netcrate quick --targets 192.168.1.0/24
```

### Verify Installation

```bash
# Check version and build info
netcrate --version

# View available commands
netcrate --help

# Check your current configuration
netcrate config show
```

## 🧩 Core Concepts

### Rate Profiles

NetCrate uses rate profiles to balance speed and stealth:

- **slow** (50 pps): Conservative, stealthy scanning
- **medium** (200 pps): Balanced performance (default)
- **fast** (1000 pps): Aggressive scanning
- **ludicrous** (5000 pps): Maximum speed

```bash
# View available profiles
netcrate config rate list

# Set your preferred profile
netcrate config rate set fast

# Create custom profile
netcrate config rate create myprofile --rate 500 --concurrency 300 --timeout 2s
```

### Privilege Levels

NetCrate automatically detects your privileges and adapts:

- **Full**: Raw socket access, all features available
- **Degraded**: Limited privileges, automatic fallbacks (ICMP→ping, SYN→connect)
- **Restricted**: Minimal privileges, TCP connect only

### Compliance System

NetCrate enforces security compliance automatically:

- **Private networks**: 192.168.x.x, 10.x.x.x, 172.16-31.x.x (allowed by default)
- **Public networks**: Require `--dangerous` flag and "YES" confirmation
- **Audit trail**: All scans logged to `~/.netcrate/compliance/`

## 🛠️ Basic Commands

### Network Discovery

```bash
# Discover live hosts on local network
netcrate ops discover auto

# Discover specific network
netcrate ops discover 192.168.1.0/24

# Use specific discovery method
netcrate ops discover 192.168.1.0/24 --methods ping,tcp

# Advanced discovery with adaptive rates
netcrate ops discover 192.168.1.0/24 --enhanced --target-pruning
```

### Port Scanning

```bash
# Scan common ports
netcrate ops scan ports --targets 192.168.1.1 --ports top100

# Scan specific ports
netcrate ops scan ports --targets 192.168.1.1 --ports 22,80,443,8080

# Scan port range
netcrate ops scan ports --targets 192.168.1.1 --ports 1-1000

# UDP scanning
netcrate ops scan ports --targets 192.168.1.1 --ports 53,161,162 --scan-type udp

# Fast SYN scan (requires privileges)
sudo netcrate ops scan ports --targets 192.168.1.1 --ports top1000 --scan-type syn
```

### Template-Based Scanning

```bash
# List available templates
netcrate templates list

# Run a template
netcrate templates run basic_scan --targets 192.168.1.0/24

# Run with parameters
netcrate templates run web_application_scan --target https://example.com --rate_profile fast

# Validate template syntax
netcrate templates validate ./my_template.yaml
```

### Output Management

```bash
# View recent scan results
netcrate output show

# Show specific scan
netcrate output show --last

# Export to different formats
netcrate output export --format json --output scan_results.json
netcrate output export --format html --output report.html

# Filter results
netcrate output show --filter "port=443,status=open"
```

## ⚙️ Configuration Management

### Basic Configuration

```bash
# View current configuration
netcrate config show

# Set preferences
netcrate config set output_format json
netcrate config set color_output false
netcrate config set verbose true
```

### Rate Profile Management

```bash
# List profiles
netcrate config rate list

# Set active profile
netcrate config rate set fast

# Create custom profile
netcrate config rate create custom \
  --rate 800 \
  --concurrency 400 \
  --timeout 1500ms \
  --description "Custom balanced profile"

# Delete custom profile
netcrate config rate delete custom
```

### Configuration Files

NetCrate stores configuration in `~/.netcrate/`:

- `config.json`: Main configuration file
- `compliance/compliance.json`: Compliance logs
- `output/`: Scan results and reports

## 📝 Templates and Workflows

### Using Built-in Templates

NetCrate includes several pre-built templates:

```bash
# Basic network scan
netcrate templates run basic_scan --targets 192.168.1.0/24

# Web application assessment
netcrate templates run web_application_scan --target https://webapp.example.com --dangerous

# Security audit
netcrate templates run security_audit --target_range 10.0.1.0/24 --audit_depth deep

# Quick reconnaissance
netcrate templates run quick_recon --target example.com --speed ludicrous --dangerous
```

### Creating Custom Templates

Templates use YAML format with Jinja2-style templating:

```yaml
name: "my_custom_scan"
version: "1.0"
description: "My custom scanning template"
author: "Your Name"
tags: ["custom", "example"]
require_dangerous: false

parameters:
  - name: "target"
    description: "Target to scan"
    type: "string"
    required: true

steps:
  - name: "discovery"
    operation: "discover"
    with:
      targets: ["{{ .target }}"]
      methods: ["ping"]
```

### Template Parameters

Common parameter types:

- `string`: Text values
- `int`: Numeric values  
- `bool`: True/false values
- `cidr`: Network ranges (e.g., 192.168.1.0/24)
- `ports`: Port specifications (e.g., 80,443 or 1-1000)
- `endpoint`: URLs or hostnames
- `list<string>`: Arrays of strings

## 🔒 Security and Compliance

### Public Network Scanning

When scanning public networks, NetCrate requires explicit confirmation:

```bash
# This will prompt for confirmation
netcrate ops discover 8.8.8.8 --dangerous

# You'll see:
# ⚠️ COMPLIANCE WARNING ⚠️
# You are about to scan PUBLIC NETWORK targets:
# • 8.8.8.8
# Type 'YES' to proceed, or anything else to abort:
```

### Legal Compliance

**⚠️ IMPORTANT**: Only use NetCrate on:

- Networks you own
- Networks where you have explicit written permission
- Authorized penetration testing environments
- Internal lab setups for learning

### Compliance Logging

All scans are automatically logged:

```bash
# View compliance history
netcrate output show --compliance

# Check compliance status
ls ~/.netcrate/compliance/
```

## 🎯 Advanced Usage

### Privilege Management

Check your current privileges:

```bash
netcrate ops discover --help  # Shows available methods based on privileges

# With sudo (full privileges)
sudo netcrate ops discover 192.168.1.0/24 --methods icmp,syn

# Without sudo (degraded mode, automatic fallbacks)
netcrate ops discover 192.168.1.0/24  # Uses ping instead of ICMP
```

### Performance Tuning

Optimize for your environment:

```bash
# Conservative (for slow networks or stealth)
netcrate config rate set slow
netcrate ops scan ports --targets host --ports top100 --timeout 5s

# Aggressive (for fast internal networks)
netcrate config rate set ludicrous  
netcrate ops scan ports --targets network --ports top1000 --timeout 500ms

# Custom tuning
netcrate ops discover network --rate 2000 --concurrency 1000 --timeout 1s
```

### Output Processing

Integrate NetCrate with other tools:

```bash
# JSON output for programmatic processing
netcrate ops discover 192.168.1.0/24 --json | jq '.results[] | select(.status=="up")'

# CSV for spreadsheet import
netcrate output export --format csv --filter "status=open" --output results.csv

# Detailed HTML reports
netcrate output export --format html --template detailed --output full_report.html
```

## 🔧 Troubleshooting

### Common Issues

**"Permission denied" errors:**
- Use `sudo` for raw socket operations (ICMP, SYN scans)
- Or use fallback methods: `--methods ping,tcp` instead of `icmp,syn`

**"No targets found":**
- Check network connectivity: `ping target`
- Verify target format: `192.168.1.0/24` not `192.168.1.1/24`
- Try different discovery methods: `--methods tcp`

**Slow scanning:**
- Increase rate profile: `netcrate config rate set fast`
- Reduce timeout: `--timeout 1s`
- Increase concurrency: `--concurrency 500`

**Template errors:**
- Validate syntax: `netcrate templates validate template.yaml`
- Check required parameters: `netcrate templates run template --help`

### Debug Mode

Enable verbose output for troubleshooting:

```bash
# Enable verbose mode permanently
netcrate config set verbose true

# Or use for single command
netcrate ops discover target --verbose
```

### Getting Help

```bash
# Command-specific help
netcrate ops discover --help
netcrate templates run --help

# List all available commands
netcrate --help

# Version and build information
netcrate --version
```

## 📚 Examples

### Example 1: Basic Network Assessment

Assess your local network:

```bash
# Step 1: Discover live hosts
netcrate ops discover 192.168.1.0/24

# Step 2: Scan common ports on live hosts
netcrate ops scan ports --targets $(netcrate output show --last --format json | jq -r '.results[] | select(.status=="up") | .host' | tr '\n' ',')

# Step 3: Generate HTML report
netcrate output export --format html --output network_assessment.html
```

### Example 2: Web Application Security Test

Test a web application (with permission):

```bash
# Use the web application template
netcrate templates run web_application_scan \
  --target https://webapp.example.com \
  --ports "80,443,8080,8443" \
  --rate_profile medium \
  --dangerous

# Review results
netcrate output show --last
```

### Example 3: Comprehensive Security Audit

Perform a thorough security audit:

```bash
# Deep security audit with UDP scanning
netcrate templates run security_audit \
  --target_range 10.0.1.0/24 \
  --audit_depth deep \
  --include_udp true

# Export detailed report
netcrate output export --format html --template comprehensive --output security_audit.html
```

### Example 4: Quick Reconnaissance

Rapid assessment of a target:

```bash
# Fast reconnaissance
netcrate templates run quick_recon \
  --target example.com \
  --speed ludicrous \
  --dangerous

# View summary
netcrate output show --last --format table
```

### Example 5: Custom Configuration

Set up NetCrate for your environment:

```bash
# Create custom rate profile for your network
netcrate config rate create mynetwork \
  --rate 300 \
  --concurrency 150 \
  --timeout 2s \
  --description "Optimized for my corporate network"

# Set preferences
netcrate config set output_format json
netcrate config set color_output true
netcrate config rate set mynetwork

# Verify configuration
netcrate config show
```

## 🤝 Getting Support

- **Documentation**: [GitHub Wiki](https://github.com/netcrate/netcrate/wiki)
- **Issues**: [GitHub Issues](https://github.com/netcrate/netcrate/issues)
- **Discussions**: [GitHub Discussions](https://github.com/netcrate/netcrate/discussions)
- **Security**: See [SECURITY.md](../SECURITY.md) for vulnerability reporting

## 📄 License

NetCrate is released under the MIT License. See [LICENSE](../LICENSE) for details.

---

**Remember**: Always ensure you have proper authorization before scanning any network. NetCrate is a powerful tool - use it responsibly!