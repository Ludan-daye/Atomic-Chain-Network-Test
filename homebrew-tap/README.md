# NetCrate Homebrew Tap

This is the official Homebrew tap for NetCrate, a network security testing toolkit with compliance controls.

## Installation

### Add the tap

```bash
brew tap netcrate/tap
```

### Install NetCrate

```bash
brew install netcrate
```

### Or install directly

```bash
brew install netcrate/tap/netcrate
```

## Usage

Once installed, you can use NetCrate commands:

```bash
# Quick network scan
netcrate quick

# Configure rate profiles  
netcrate config rate set fast

# Network discovery
netcrate ops discover 192.168.1.0/24

# Port scanning
netcrate ops scan ports --targets 192.168.1.1 --ports top100

# View help
netcrate --help
```

## Requirements

- macOS 10.15+ or Linux
- Optional: `nmap` for enhanced functionality

## Configuration

NetCrate stores its configuration in `~/.netcrate/`:
- `config.json`: Rate profiles and user preferences  
- `compliance/`: Compliance logs and audit trails
- `output/`: Scan results and reports

## Security Notice

⚠️ **IMPORTANT**: NetCrate is a security testing tool. Only use it on:
- Networks you own
- Networks where you have explicit written permission to test
- Internal lab environments for learning

Unauthorized network scanning may violate laws, policies, and terms of service.

## Features

- **Compliance Controls**: Automatic public network detection with mandatory confirmation
- **Privilege Detection**: Automatic fallback when raw sockets unavailable  
- **Rate Profiles**: Configurable speed presets (slow/medium/fast/ludicrous)
- **Multi-platform**: Works on macOS, Linux, and Windows
- **Template System**: Reusable scan configurations
- **Comprehensive Output**: JSON, table, and HTML report formats

## Support

- **Documentation**: [GitHub Wiki](https://github.com/netcrate/netcrate/wiki)
- **Issues**: [GitHub Issues](https://github.com/netcrate/netcrate/issues)  
- **Discussions**: [GitHub Discussions](https://github.com/netcrate/netcrate/discussions)

## Development

To install from source:

```bash
# Clone the repository
git clone https://github.com/netcrate/netcrate.git
cd netcrate

# Build and install
make build
make install
```

To test the Homebrew formula locally:

```bash
# Audit the formula
brew audit --strict --online netcrate/tap/netcrate

# Test installation
brew install --build-from-source netcrate/tap/netcrate

# Run tests
brew test netcrate/tap/netcrate
```

## License

MIT License - see [LICENSE](https://github.com/netcrate/netcrate/blob/main/LICENSE) for details.