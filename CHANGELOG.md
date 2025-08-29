# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive compliance system with public network protection
- Privilege detection and automatic fallback mechanisms (ICMP→ping, SYN→connect)
- Rate profile persistence system with multiple speed presets (slow/medium/fast/ludicrous)
- Configuration management with persistent settings in ~/.netcrate/config.json
- Multi-platform build support with GoReleaser
- Template system for reusable scan configurations
- Enhanced discovery with adaptive rates and method fallback
- Banner grabbing and service fingerprinting
- Comprehensive output and result management
- Security-focused design with built-in compliance checks
- Version management and build information injection

### Changed
- Improved error handling and user feedback
- Enhanced privilege detection across platforms (Linux, macOS, Windows)
- Optimized scanning performance with configurable rate profiles
- Updated project structure with proper package organization

### Security
- Added mandatory --dangerous flag for public network scanning
- Implemented user confirmation prompts for high-risk operations ("YES" confirmation required)
- Enhanced compliance logging and audit trails
- Automatic detection of public vs private networks (RFC 1918)
- Privilege-aware operation selection to prevent failures

---

## Release Notes

### Version Numbering
- **0.x.x**: MVP development phase, API may change
- **1.0.0**: Stable release with backward compatibility promise
- **x.y.z**: Patch releases for bug fixes only

### Template Compatibility
- Templates must declare minimum supported engine version
- Engine maintains forward compatibility with older template versions
- Breaking changes are reflected in major version increments