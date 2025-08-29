# NetCrate

[![CI](https://github.com/netcrate/netcrate/workflows/CI/badge.svg)](https://github.com/netcrate/netcrate/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/netcrate/netcrate)](https://goreportcard.com/report/github.com/netcrate/netcrate)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Security Scan](https://github.com/netcrate/netcrate/workflows/Security%20Scan/badge.svg)](https://github.com/netcrate/netcrate/actions)

**NetCrate** is a comprehensive network security testing toolkit designed for authorized penetration testing, network diagnostics, and security research. Built with Go for exceptional performance and cross-platform compatibility.

## ⚠️ Legal Notice

**This tool is intended for authorized security testing only.** Please read our [legal compliance guide](LEGAL.md) before use. Users are responsible for ensuring their activities comply with applicable laws and regulations.

## 🚀 Quick Start

### Installation

#### Homebrew (macOS - Recommended)
```bash
brew tap netcrate/tap
brew install netcrate
```

#### Direct Download
Download the latest release for your platform from [GitHub Releases](https://github.com/netcrate/netcrate/releases).

#### Build from Source
```bash
git clone https://github.com/netcrate/netcrate.git
cd netcrate
go build -o netcrate ./cmd/netcrate
```

### First Run

```bash
# Interactive quick scan wizard
netcrate quick

# Show help
netcrate --help

# Show version
netcrate --version
```

## 🎯 Features

### Core Capabilities
- **Host Discovery**: Fast network host discovery with multiple probe methods
- **Port Scanning**: TCP/UDP port scanning with service detection
- **Packet Crafting**: Custom packet generation and analysis
- **Template System**: Reusable testing workflows via YAML templates
- **Interactive CLI**: User-friendly wizard-based interface

### Built-in Security
- **Private Network Focus**: Defaults to private IP ranges only
- **Rate Limiting**: Configurable speed controls to prevent network disruption
- **Compliance Checking**: Built-in validation against security best practices
- **Audit Logging**: Complete operation logging for accountability

### Operational Modes

#### 🧙‍♂️ Quick Mode (Beginner-Friendly)
```bash
netcrate quick
```
Step-by-step wizard that guides you through:
1. Network interface selection
2. Target range confirmation
3. Host discovery
4. Port scanning
5. Service probing

#### ⚡ Ops Mode (Advanced Users)
```bash
# Individual operations
netcrate discover auto
netcrate scan ports --targets 192.168.1.0/24 --ports top100
netcrate packet send --template syn --to 192.168.1.1:80
```

#### 📋 Template Mode (Automation)
```bash
# Run predefined workflows
netcrate templates run basic_scan --param target_range=192.168.1.0/24
netcrate templates ls
netcrate templates view basic_scan
```

## 📚 Documentation

### User Guides
- [Installation Guide](docs/INSTALLATION.md) - Detailed installation instructions
- [User Manual](docs/USER_GUIDE.md) - Complete feature documentation
- [Template Guide](docs/TEMPLATES.md) - Creating and using templates
- [Configuration](docs/CONFIGURATION.md) - Settings and customization

### Technical Documentation
- [API Reference](docs/API.md) - Programming interface
- [Architecture](docs/ARCHITECTURE.md) - Internal design and components
- [Contributing](CONTRIBUTING.md) - Development guidelines
- [Security](docs/SECURITY.md) - Security considerations and best practices

## 🛡️ Security & Compliance

### Default Safety Features
- **Private Network Only**: Restricts operations to RFC 1918 private networks by default
- **Rate Limiting**: Safe defaults (≤100 pps, ≤200 concurrent connections)
- **Permission Checks**: Automatic fallback when elevated privileges unavailable
- **Interactive Confirmations**: Prompts for potentially impactful operations

### Compliance Profiles

| Profile | Max Rate | Max Concurrent | Use Case |
|---------|----------|----------------|----------|
| Safe (Default) | 100 pps | 200 | Daily diagnostics, small tests |
| Fast | 400 pps | 800 | Professional penetration testing |
| Custom | User-defined | User-defined | Expert configuration |

### Public Network Access
Public network testing requires:
- `--dangerous` flag
- Interactive confirmation
- Reduced rate limits
- Additional legal warnings

## 🔧 Configuration

### Config File Locations
```
~/.netcrate/config.yaml          # User configuration
/etc/netcrate/config.yaml        # System-wide configuration
./netcrate.yaml                  # Project-specific configuration
```

### Example Configuration
```yaml
rate_limits:
  profile: "safe"
  
compliance:
  allow_public: false
  require_confirmation: true
  
output:
  workspace_dir: "~/.netcrate/runs"
  retention_days: 30
```

## 📊 Output & Results

### Output Formats
- **Human-readable**: Interactive tables and progress bars
- **JSON/NDJSON**: Machine-readable for automation
- **CSV**: Spreadsheet-compatible exports

### Result Management
```bash
# View recent results
netcrate output show

# List all saved results
netcrate output list

# Export specific run
netcrate output export --run <id> --out results.json
```

## 🧪 Examples

### Basic Network Discovery
```bash
# Auto-detect network and discover hosts
netcrate quick

# Manual network specification  
netcrate discover 192.168.1.0/24
```

### Port Scanning
```bash
# Scan top 100 ports
netcrate scan ports --targets 192.168.1.0/24 --ports top100

# Custom port ranges
netcrate scan ports --targets file:hosts.txt --ports 22,80,443,8000-9000
```

### Custom Packet Testing
```bash
# TCP SYN probe
netcrate packet send --template syn --to 192.168.1.1:80

# HTTP request
netcrate packet send --template http --to 192.168.1.1:80 --param path=/admin

# DNS query
netcrate packet send --template dns --to 8.8.8.8:53 --param domain=example.com
```

### Template-based Workflows
```bash
# Run comprehensive scan template
netcrate templates run comprehensive_scan --param target=192.168.1.0/24

# Create custom template
netcrate templates new my_scan --based-on basic_scan
```

## 🚧 Development Status

### Current Version: 0.1.0-dev
This is early development software. APIs and features may change.

### Roadmap
- [ ] **v0.1.0**: Basic CLI structure and core operations
- [ ] **v0.2.0**: Template system and interactive wizards  
- [ ] **v0.3.0**: Advanced packet crafting and analysis
- [ ] **v0.4.0**: Reporting and visualization features
- [ ] **v1.0.0**: Stable API and production readiness

### Contributing
We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) for details on:
- Development setup
- Code standards
- Testing requirements
- Security considerations

## 📄 License

NetCrate is released under the [MIT License](LICENSE).

## 🤝 Community

- **Issues**: [GitHub Issues](https://github.com/netcrate/netcrate/issues)
- **Discussions**: [GitHub Discussions](https://github.com/netcrate/netcrate/discussions)
- **Security**: Report vulnerabilities to security@netcrate.dev

## 🙏 Acknowledgments

NetCrate builds upon the excellent work of:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- Go standard library networking packages
- The broader network security research community

---

**Built with ❤️ for the security community**

*Remember: With great power comes great responsibility. Use NetCrate ethically and legally.*