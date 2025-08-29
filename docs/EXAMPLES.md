# NetCrate Examples

This document provides practical examples for common NetCrate use cases. Each example includes the command, expected output, and explanations.

## 📋 Table of Contents

- [Getting Started Examples](#getting-started-examples)
- [Network Discovery Examples](#network-discovery-examples)
- [Port Scanning Examples](#port-scanning-examples)
- [Template Examples](#template-examples)
- [Configuration Examples](#configuration-examples)
- [Advanced Scenarios](#advanced-scenarios)
- [Troubleshooting Examples](#troubleshooting-examples)

## 🚀 Getting Started Examples

### Example 1: First Time Setup

```bash
# Install NetCrate (macOS/Linux)
brew tap netcrate/tap
brew install netcrate

# Verify installation
netcrate --version
# Output: NetCrate 1.0.0 (abc1234) built on 2023-12-01T10:00:00Z by goreleaser with go1.21.0 for darwin/arm64

# Check initial configuration
netcrate config show
# Output shows default settings, medium rate profile, empty recent targets
```

### Example 2: Quick Network Scan

```bash
# Auto-detect and scan your local network
netcrate quick

# Expected output:
# 🔍 NetCrate Quick Mode
# ====================
# Auto-detected network: 192.168.1.0/24
# Privilege mode: degraded (using system ping fallback)
# 
# 📡 Discovery Results:
# • 192.168.1.1 (up) - Router/Gateway
# • 192.168.1.10 (up) - Desktop-PC
# • 192.168.1.20 (up) - Laptop
# 
# 🔍 Port Scan Results:
# 192.168.1.1:
#   • 22/tcp open (ssh)
#   • 80/tcp open (http)
#   • 443/tcp open (https)
# 
# Scan completed in 15.2s
```

## 🌐 Network Discovery Examples

### Example 3: Discover Local Network

```bash
# Basic discovery
netcrate ops discover 192.168.1.0/24

# Expected output:
# 🔍 Network Discovery
# ===================
# Target: 192.168.1.0/24 (254 hosts)
# Methods: ping, tcp
# Privilege mode: degraded
# 
# Results:
# ✅ 192.168.1.1    (5.2ms)   - method: ping
# ✅ 192.168.1.10   (2.1ms)   - method: ping  
# ✅ 192.168.1.20   (1.8ms)   - method: ping
# ❌ 192.168.1.50   (timeout) - method: ping
# 
# Summary: 3/254 hosts up (1.18%), completed in 8.5s
```

### Example 4: Discovery with Different Methods

```bash
# Try ICMP first (requires sudo)
sudo netcrate ops discover 192.168.1.0/24 --methods icmp

# TCP-based discovery (no privileges needed)
netcrate ops discover 192.168.1.0/24 --methods tcp --tcp-ports 22,80,443

# ARP discovery for local network (requires sudo)
sudo netcrate ops discover 192.168.1.0/24 --methods arp
```

### Example 5: Enhanced Discovery with Adaptive Rates

```bash
# Enable advanced features
netcrate ops discover 192.168.1.0/24 --enhanced --target-pruning --adaptive-rate

# Expected output shows:
# 🚀 Enhanced Discovery Mode
# =========================
# • Target pruning: Enabled
# • Adaptive rate: Enabled  
# • Method fallback: Enabled
# 
# Phase 1: Sampling 5% of targets (12 hosts)
# Phase 2: Full scan with optimized rate (782 pps)
# Phase 3: Verification of edge cases
```

## 🔍 Port Scanning Examples

### Example 6: Basic Port Scanning

```bash
# Scan common ports
netcrate ops scan ports --targets 192.168.1.1 --ports top100

# Expected output:
# 🔍 Port Scanning
# ================
# Target: 192.168.1.1
# Ports: top100 (100 ports)
# Scan type: connect
# 
# Results:
# ✅ 22/tcp    open     ssh      (OpenSSH 8.9)
# ✅ 53/tcp    open     domain   (dnsmasq 2.85)  
# ✅ 80/tcp    open     http     (nginx 1.20.1)
# ✅ 443/tcp   open     https    (nginx 1.20.1)
# ❌ 21/tcp    closed   ftp
# ❌ 23/tcp    filtered telnet
# 
# Summary: 4 open, 2 closed, 94 filtered
# Scan completed in 12.3s
```

### Example 7: Custom Port Ranges

```bash
# Scan specific ports
netcrate ops scan ports --targets 192.168.1.1 --ports 22,80,443,8080,8443

# Scan port range
netcrate ops scan ports --targets 192.168.1.1 --ports 1000-2000

# Combine multiple specifications
netcrate ops scan ports --targets 192.168.1.1 --ports 22,80,443,8000-8999
```

### Example 8: UDP Scanning

```bash
# Common UDP ports
netcrate ops scan ports --targets 192.168.1.1 --ports 53,67,123,161 --scan-type udp

# Expected output:
# 🔍 UDP Port Scanning
# ====================
# Note: UDP scanning results may show "open|filtered" due to protocol limitations
# 
# Results:
# ✅ 53/udp   open     domain
# ❓ 67/udp   open|filtered dhcps
# ✅ 123/udp  open     ntp
# ❓ 161/udp  open|filtered snmp
```

### Example 9: SYN Scanning (Requires Root)

```bash
# Fast SYN scan (requires sudo)
sudo netcrate ops scan ports --targets 192.168.1.1 --ports top1000 --scan-type syn

# Shows faster results with privilege escalation
# Also displays privilege status:
# 🔓 Privilege Status: full (raw socket available)
```

## 📝 Template Examples

### Example 10: Using Built-in Templates

```bash
# List available templates
netcrate templates list

# Output:
# 📋 Available Templates
# =====================
# 
# Built-in Templates:
# • basic_scan - Basic network and port scanning
# • web_application_scan - Web application security assessment  
# • network_discovery - Network discovery and asset enumeration
# • security_audit - Comprehensive security audit
# • quick_recon - Quick reconnaissance
# 
# User Templates:
# • custom_template.yaml (in current directory)

# Run basic scan template
netcrate templates run basic_scan --targets 192.168.1.0/24
```

### Example 11: Web Application Scanning

```bash
# Scan a web application (requires --dangerous for public targets)
netcrate templates run web_application_scan \
  --target https://example.com \
  --ports "80,443,8080,8443" \
  --rate_profile fast \
  --dangerous

# Expected confirmation prompt:
# ⚠️ COMPLIANCE WARNING ⚠️
# ==========================================
# You are about to scan PUBLIC NETWORK targets:
#   • example.com
# 
# 🚨 IMPORTANT SECURITY NOTICE:
# • Only scan networks you own or have explicit permission to test
# • Unauthorized scanning may violate laws and policies
# 
# Command: netcrate templates run web_application_scan
# Template: web_application_scan  
# Risk Level: high
# 
# ⚠️ Type 'YES' to proceed, or anything else to abort: YES

# Scan proceeds with comprehensive web app assessment
```

### Example 12: Security Audit Template

```bash
# Deep security audit of internal network
netcrate templates run security_audit \
  --target_range 10.0.1.0/24 \
  --audit_depth deep \
  --include_udp true

# Shows multi-phase scanning:
# Phase 1: Host Discovery (ping sweep)
# Phase 2: TCP Port Scanning (1-65535)  
# Phase 3: UDP Port Scanning (common ports)
# Phase 4: Service Detection & Banner Grabbing
# Phase 5: Service Fingerprinting
```

## ⚙️ Configuration Examples

### Example 13: Rate Profile Management

```bash
# View current rate profiles
netcrate config rate list

# Output:
# Rate Profiles
# =============
# Current profile: medium
# 
# • slow
#   Description: Conservative scanning for stealth and stability
#   Rate: 50 pps | Concurrency: 50 workers | Timeout: 3s | Retries: 3
# 
# • medium (current)  
#   Description: Balanced scanning for general use
#   Rate: 200 pps | Concurrency: 200 workers | Timeout: 2s | Retries: 2
# 
# • fast
#   Description: Aggressive scanning for speed
#   Rate: 1000 pps | Concurrency: 500 workers | Timeout: 1s | Retries: 1

# Switch to fast profile
netcrate config rate set fast
# Output: ✅ Rate profile set to: fast
#         Settings: 1000 pps, 500 workers, 1s timeout, 1 retries

# Create custom profile
netcrate config rate create myprofile \
  --rate 600 \
  --concurrency 300 \
  --timeout 1500ms \
  --retries 2 \
  --description "Balanced profile for corporate network"

# Output: ✅ Custom rate profile 'myprofile' created
#         Settings: 600 pps, 300 workers, 1.5s timeout, 2 retries
```

### Example 14: General Configuration

```bash
# View all configuration
netcrate config show

# Set output preferences  
netcrate config set output_format json
netcrate config set color_output false
netcrate config set verbose true

# Check configuration file location
# ~/.netcrate/config.json contains:
# {
#   "current_rate_profile": "fast",
#   "preferences": {
#     "default_output_format": "json",
#     "color_output": false,
#     "verbose_mode": true
#   }
# }
```

## 🎯 Advanced Scenarios

### Example 15: Large Network Assessment

```bash
# Efficient scanning of large network (Class B)
netcrate config rate set ludicrous
netcrate ops discover 172.16.0.0/16 --enhanced --target-pruning

# Expected intelligent behavior:
# 🧠 Smart Scanning Detected
# ==========================
# Large network detected (65,534 hosts)
# • Enabling sampling mode (1% sample = 655 hosts)
# • Adaptive rate scaling activated
# • Target pruning enabled
# 
# Phase 1: Statistical sampling (655 hosts)
# Phase 2: Dense subnet identification  
# Phase 3: Full scan of active subnets only
# 
# Estimated time: 8-12 minutes (vs 4+ hours without optimization)
```

### Example 16: Privilege-Aware Scanning

```bash
# Check privilege status
netcrate ops discover 192.168.1.1 --help
# Shows available methods based on current privileges

# Without sudo (shows fallback behavior)
netcrate ops discover 192.168.1.0/24

# Output includes privilege info:
# 🔒 Privilege Status Report
# ==========================
# Level: degraded
# Root/Admin: false
# 
# Capabilities:
# ✅ system_ping  
# ✅ tcp_connect
# ❌ icmp (fallback: system ping)
# ❌ syn_scan (fallback: connect scan)

# With sudo (full capabilities)  
sudo netcrate ops discover 192.168.1.0/24

# Output shows:
# Level: full
# Root/Admin: true
# ✅ icmp
# ✅ raw_socket
# ✅ syn_scan
```

### Example 17: Integration with Other Tools

```bash
# Export to JSON for programmatic processing
netcrate ops discover 192.168.1.0/24 --json > discovery.json

# Extract live hosts with jq
cat discovery.json | jq -r '.results[] | select(.status=="up") | .host'

# Feed to nmap for detailed scanning
netcrate ops discover 192.168.1.0/24 --json | \
  jq -r '.results[] | select(.status=="up") | .host' | \
  xargs -I {} nmap -sV {}

# Generate CSV report
netcrate output export --format csv --output network_scan.csv
```

## 🔧 Troubleshooting Examples

### Example 18: Permission Issues

```bash
# Error: Raw socket creation failed
netcrate ops discover 192.168.1.0/24 --methods icmp
# Error: operation not permitted

# Solution 1: Use sudo
sudo netcrate ops discover 192.168.1.0/24 --methods icmp

# Solution 2: Use fallback methods  
netcrate ops discover 192.168.1.0/24 --methods ping

# Solution 3: Check what's available
netcrate ops discover 192.168.1.0/24
# Auto-selects best available methods
```

### Example 19: Slow Scanning Issues

```bash
# Problem: Scanning is too slow
netcrate ops scan ports --targets 192.168.1.1 --ports top1000
# Takes 10+ minutes

# Solution 1: Increase rate profile
netcrate config rate set fast
netcrate ops scan ports --targets 192.168.1.1 --ports top1000
# Now takes 2-3 minutes

# Solution 2: Reduce timeout
netcrate ops scan ports --targets 192.168.1.1 --ports top1000 --timeout 500ms

# Solution 3: Increase concurrency  
netcrate ops scan ports --targets 192.168.1.1 --ports top1000 --concurrency 1000
```

### Example 20: Template Validation Issues

```bash
# Problem: Template fails to run
netcrate templates run my_template.yaml --targets 192.168.1.1
# Error: invalid parameter type

# Solution: Validate template first
netcrate templates validate my_template.yaml
# Output shows specific syntax errors

# Check required parameters
netcrate templates run my_template.yaml --help
# Shows all required and optional parameters

# Run with verbose output
netcrate templates run my_template.yaml --targets 192.168.1.1 --verbose
# Shows detailed execution steps
```

### Example 21: Network Connectivity Issues

```bash
# Problem: No hosts found in discovery
netcrate ops discover 192.168.1.0/24
# Summary: 0/254 hosts up (0%)

# Diagnosis 1: Check your own connectivity
ping 192.168.1.1
netstat -rn | grep default

# Diagnosis 2: Try different methods
netcrate ops discover 192.168.1.0/24 --methods tcp --tcp-ports 22,80,443

# Diagnosis 3: Increase timeout
netcrate ops discover 192.168.1.0/24 --timeout 5s

# Diagnosis 4: Check a single known host
netcrate ops discover 192.168.1.1 --verbose
```

### Example 22: Public Network Compliance

```bash
# Problem: Want to scan public network
netcrate ops discover 8.8.8.8
# Error: Compliance violation: public network targets require --dangerous flag

# Solution: Use --dangerous flag (with caution)
netcrate ops discover 8.8.8.8 --dangerous
# Prompts for "YES" confirmation

# Check compliance logs
netcrate output show --compliance
# Shows audit trail of all scans
```

## 📊 Output Examples

### Example 23: Different Output Formats

```bash
# Table format (default)
netcrate ops scan ports --targets 192.168.1.1 --ports 22,80,443

# JSON format
netcrate ops scan ports --targets 192.168.1.1 --ports 22,80,443 --json

# Export to HTML report
netcrate output export --format html --output report.html
```

### Example 24: Filtering and Processing Results

```bash
# Show only open ports
netcrate output show --filter "status=open"

# Show specific hosts
netcrate output show --filter "host=192.168.1.1"

# Show last scan with details
netcrate output show --last --format json | jq '.results[] | select(.status=="open")'
```

## 🎓 Learning Exercises

### Exercise 1: Basic Network Assessment

1. Discover your local network
2. Identify live hosts  
3. Scan common ports on each host
4. Generate a summary report

```bash
# Step 1
netcrate ops discover auto

# Step 2  
netcrate output show --last --format json | jq -r '.results[] | select(.status=="up") | .host'

# Step 3
netcrate ops scan ports --targets $(previous_command_output) --ports top100

# Step 4
netcrate output export --format html --output my_network_assessment.html
```

### Exercise 2: Template Creation

Create a custom template for your specific scanning needs:

1. Start with basic_scan template
2. Modify parameters for your environment
3. Add custom port lists
4. Test and validate

### Exercise 3: Performance Optimization  

Find the optimal settings for your network:

1. Test different rate profiles
2. Measure scan times
3. Create custom profile
4. Document your results

---

These examples should help you get started with NetCrate and understand its capabilities. Remember to always scan only networks you own or have permission to test!