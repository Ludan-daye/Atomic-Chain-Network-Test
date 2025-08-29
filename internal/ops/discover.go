// Package ops provides atomic network operations
package ops

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/netcrate/netcrate/internal/netenv"
	"github.com/netcrate/netcrate/internal/privileges"
)

// DiscoverOptions contains configuration for host discovery
type DiscoverOptions struct {
	Targets     []string  `json:"targets"`
	Methods     []string  `json:"methods"`
	Interface   string    `json:"interface"`
	Rate        int       `json:"rate"`
	Timeout     time.Duration `json:"timeout"`
	Concurrency int       `json:"concurrency"`
	TCPPorts    []int     `json:"tcp_ports"`
	ResolveHostnames bool `json:"resolve_hostnames"`
}

// DiscoverResult represents the result of host discovery
type DiscoverResult struct {
	Host      string            `json:"host"`
	Status    string            `json:"status"` // "up", "down", "timeout", "error"
	RTT       float64           `json:"rtt"`    // milliseconds
	Method    string            `json:"method"` // "icmp", "tcp", "arp"
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time         `json:"timestamp"`
	Hostname  string            `json:"hostname,omitempty"`
}

// DiscoverSummary provides summary statistics
type DiscoverSummary struct {
	RunID            string            `json:"run_id"`
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	Duration         float64           `json:"duration"`
	TargetsInput     string            `json:"targets_input"`
	TargetsResolved  int               `json:"targets_resolved"`
	HostsDiscovered  int               `json:"hosts_discovered"`
	SuccessRate      float64           `json:"success_rate"`
	MethodUsed       []string          `json:"method_used"`
	InterfaceUsed    string            `json:"interface_used"`
	Results          []DiscoverResult  `json:"results"`
	Stats            DiscoverStats     `json:"stats"`
	PrivilegeMode    string            `json:"privilege_mode"`
	FallbackReasons  []string          `json:"fallback_reasons,omitempty"`
	PrivilegeSummary map[string]interface{} `json:"privilege_summary,omitempty"`
}

// DiscoverStats provides detailed statistics
type DiscoverStats struct {
	Sent            int                    `json:"sent"`
	Received        int                    `json:"received"`
	Errors          int                    `json:"errors"`
	Timeouts        int                    `json:"timeouts"`
	MethodBreakdown map[string]MethodStats `json:"method_breakdown"`
}

// MethodStats provides per-method statistics
type MethodStats struct {
	Sent     int `json:"sent"`
	Received int `json:"received"`
}

// Discover performs host discovery on the specified targets
func Discover(opts DiscoverOptions) (*DiscoverSummary, error) {
	startTime := time.Now()
	runID := fmt.Sprintf("discover_%d", startTime.Unix())

	// Initialize privilege manager for capability detection
	pm := privileges.NewPrivilegeManager()

	// Parse and expand targets
	targets, err := parseTargets(opts.Targets)
	if err != nil {
		return nil, fmt.Errorf("failed to parse targets: %w", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets specified")
	}

	// Set defaults
	if opts.Rate == 0 {
		opts.Rate = 100
	}
	if opts.Timeout == 0 {
		opts.Timeout = 1000 * time.Millisecond
	}
	if opts.Concurrency == 0 {
		opts.Concurrency = 200
	}
	if len(opts.Methods) == 0 {
		// Use optimal methods based on privileges
		if pm.HasCapability(privileges.CapabilityICMP) {
			opts.Methods = []string{"icmp", "tcp"}
		} else if pm.HasCapability(privileges.CapabilitySystemPing) {
			opts.Methods = []string{"ping", "tcp"}
		} else {
			opts.Methods = []string{"tcp"}
		}
	}
	if len(opts.TCPPorts) == 0 {
		opts.TCPPorts = []int{80, 443, 22}
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Rate limiter
	rateLimiter := time.NewTicker(time.Second / time.Duration(opts.Rate))
	defer rateLimiter.Stop()

	// Results channel
	results := make(chan DiscoverResult, opts.Concurrency)
	
	// Semaphore for concurrency control
	sem := make(chan struct{}, opts.Concurrency)

	var wg sync.WaitGroup
	var stats DiscoverStats
	stats.MethodBreakdown = make(map[string]MethodStats)

	// Start discovery workers
	for _, target := range targets {
		wg.Add(1)
		
		go func(target string) {
			defer wg.Done()
			
			// Rate limiting
			select {
			case <-rateLimiter.C:
			case <-ctx.Done():
				return
			}

			// Concurrency control
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			result := discoverSingleTarget(ctx, target, opts)
			
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}
		}(target)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []DiscoverResult
	for result := range results {
		allResults = append(allResults, result)
		
		// Update stats
		stats.Sent++
		if result.Status == "up" {
			stats.Received++
		} else if result.Status == "timeout" {
			stats.Timeouts++
		} else if result.Status == "error" {
			stats.Errors++
		}

		// Update method stats
		methodStats := stats.MethodBreakdown[result.Method]
		methodStats.Sent++
		if result.Status == "up" {
			methodStats.Received++
		}
		stats.MethodBreakdown[result.Method] = methodStats
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Calculate success rate
	var successRate float64
	if len(allResults) > 0 {
		successRate = float64(stats.Received) / float64(len(allResults))
	}

	summary := &DiscoverSummary{
		RunID:            runID,
		StartTime:        startTime,
		EndTime:          endTime,
		Duration:         duration.Seconds(),
		TargetsInput:     strings.Join(opts.Targets, ","),
		TargetsResolved:  len(targets),
		HostsDiscovered:  stats.Received,
		SuccessRate:      successRate,
		MethodUsed:       opts.Methods,
		InterfaceUsed:    opts.Interface,
		Results:          allResults,
		Stats:            stats,
		PrivilegeMode:    pm.GetLevel().String(),
		FallbackReasons:  pm.GetFallbackReasons(),
		PrivilegeSummary: pm.GetPrivilegeSummary(),
	}

	return summary, nil
}

func parseTargets(targets []string) ([]string, error) {
	var result []string

	for _, target := range targets {
		switch {
		case target == "auto":
			// Auto-detect current network
			interfaces, err := netenv.GetActiveInterfaces()
			if err != nil {
				return nil, fmt.Errorf("failed to auto-detect network: %w", err)
			}
			
			for _, iface := range interfaces {
				if iface.Type != "loopback" && len(iface.Addresses) > 0 {
					for _, addr := range iface.Addresses {
						if strings.Contains(addr.Network, "/") {
							expanded, err := expandCIDR(addr.Network)
							if err != nil {
								continue
							}
							result = append(result, expanded...)
							break // Only use first address per interface
						}
					}
					break // Only use first suitable interface
				}
			}

		case strings.Contains(target, "/"):
			// CIDR notation
			expanded, err := expandCIDR(target)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %s: %w", target, err)
			}
			result = append(result, expanded...)

		case strings.Contains(target, "-"):
			// IP range (e.g., 192.168.1.1-100)
			expanded, err := expandRange(target)
			if err != nil {
				return nil, fmt.Errorf("invalid range %s: %w", target, err)
			}
			result = append(result, expanded...)

		case strings.HasPrefix(target, "file:"):
			// File reference - TODO: implement file reading
			return nil, fmt.Errorf("file targets not yet implemented")

		default:
			// Single IP or hostname
			if net.ParseIP(target) != nil || isValidHostname(target) {
				result = append(result, target)
			} else {
				return nil, fmt.Errorf("invalid target: %s", target)
			}
		}
	}

	return result, nil
}

func expandCIDR(cidr string) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		// Skip network and broadcast addresses for /24 and larger
		if ip.Equal(ipnet.IP) || ip.Equal(getBroadcastAddress(ipnet)) {
			continue
		}
		ips = append(ips, ip.String())
		
		// Limit to prevent memory issues
		if len(ips) >= 65535 {
			break
		}
	}

	return ips, nil
}

func expandRange(rangeStr string) ([]string, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format")
	}

	startIP := net.ParseIP(parts[0])
	if startIP == nil {
		return nil, fmt.Errorf("invalid start IP")
	}

	// Handle cases like "192.168.1.1-100" or "192.168.1.1-192.168.1.100"
	var endIP net.IP
	if net.ParseIP(parts[1]) != nil {
		endIP = net.ParseIP(parts[1])
	} else {
		// Extract base IP and append the end number
		ipParts := strings.Split(parts[0], ".")
		if len(ipParts) != 4 {
			return nil, fmt.Errorf("invalid IP format")
		}
		endNum := parts[1]
		endIPStr := strings.Join(ipParts[:3], ".") + "." + endNum
		endIP = net.ParseIP(endIPStr)
	}

	if endIP == nil {
		return nil, fmt.Errorf("invalid end IP")
	}

	var ips []string
	for ip := startIP; !ip.Equal(endIP); incrementIP(ip) {
		ips = append(ips, ip.String())
		if len(ips) >= 65535 {
			break
		}
	}
	ips = append(ips, endIP.String())

	return ips, nil
}

func discoverSingleTarget(ctx context.Context, target string, opts DiscoverOptions) DiscoverResult {
	result := DiscoverResult{
		Host:      target,
		Status:    "down",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Try each method in order until one succeeds
	for _, method := range opts.Methods {
		var success bool
		var rtt time.Duration
		var details map[string]interface{}

		switch method {
		case "icmp":
			success, rtt, details = tryICMP(ctx, target, opts.Timeout)
		case "ping":
			success, rtt, details = trySystemPing(ctx, target, opts.Timeout)
		case "tcp":
			success, rtt, details = tryTCP(ctx, target, opts.TCPPorts, opts.Timeout)
		case "arp":
			success, rtt, details = tryARP(ctx, target, opts.Timeout)
		default:
			continue
		}

		result.Method = method
		if details != nil {
			result.Details = details
		}

		if success {
			result.Status = "up"
			result.RTT = float64(rtt) / float64(time.Millisecond)
			
			// Resolve hostname if requested
			if opts.ResolveHostnames {
				if names, err := net.LookupAddr(target); err == nil && len(names) > 0 {
					result.Hostname = names[0]
				}
			}
			
			return result
		}
	}

	return result
}

func tryICMP(ctx context.Context, target string, timeout time.Duration) (bool, time.Duration, map[string]interface{}) {
	// Try native ICMP socket first
	// This would require raw socket implementation
	// For now, fall back to system ping
	return trySystemPing(ctx, target, timeout)
}

func trySystemPing(ctx context.Context, target string, timeout time.Duration) (bool, time.Duration, map[string]interface{}) {
	start := time.Now()
	
	// Use system ping command
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", fmt.Sprintf("%d", int(timeout/time.Millisecond)), target)
	output, err := cmd.Output()
	
	rtt := time.Since(start)
	
	if err != nil {
		return false, rtt, map[string]interface{}{"error": err.Error(), "fallback_used": "system_ping"}
	}

	// Parse RTT from ping output
	if realRTT := parseRTTFromPing(string(output)); realRTT > 0 {
		rtt = realRTT
	}

	details := map[string]interface{}{
		"method": "ping",
		"fallback_used": "system_ping",
		"output": strings.TrimSpace(string(output)),
	}

	return true, rtt, details
}

func tryTCP(ctx context.Context, target string, ports []int, timeout time.Duration) (bool, time.Duration, map[string]interface{}) {
	var lastErr error
	
	for _, port := range ports {
		start := time.Now()
		
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", target, port), timeout)
		rtt := time.Since(start)
		
		if err != nil {
			lastErr = err
			continue
		}
		
		conn.Close()
		
		details := map[string]interface{}{
			"method":   "tcp",
			"tcp_port": port,
		}
		
		return true, rtt, details
	}

	details := map[string]interface{}{
		"method": "tcp",
		"error":  lastErr.Error(),
	}
	
	return false, 0, details
}

func tryARP(ctx context.Context, target string, timeout time.Duration) (bool, time.Duration, map[string]interface{}) {
	// ARP is only for local network targets
	// This is a simplified implementation
	start := time.Now()
	
	// Use system arp command
	cmd := exec.CommandContext(ctx, "arp", "-n", target)
	output, err := cmd.Output()
	rtt := time.Since(start)
	
	details := map[string]interface{}{
		"method": "arp",
	}
	
	if err != nil {
		details["error"] = err.Error()
		return false, rtt, details
	}
	
	// Check if we got a valid MAC address
	if strings.Contains(string(output), ":") {
		details["output"] = strings.TrimSpace(string(output))
		return true, rtt, details
	}
	
	return false, rtt, details
}

// Helper functions

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func getBroadcastAddress(ipnet *net.IPNet) net.IP {
	broadcast := make(net.IP, len(ipnet.IP))
	for i := range ipnet.IP {
		broadcast[i] = ipnet.IP[i] | ^ipnet.Mask[i]
	}
	return broadcast
}

func parseRTTFromPing(output string) time.Duration {
	// Parse RTT from ping output like "time=1.234 ms"
	re := regexp.MustCompile(`time=([0-9.]+)\s*ms`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		if rtt, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return time.Duration(rtt * float64(time.Millisecond))
		}
	}
	return 0
}

func isValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}
	
	// Basic hostname validation
	re := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	return re.MatchString(hostname)
}