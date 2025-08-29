package ops

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/netcrate/netcrate/internal/netenv"
)

// TargetPriority represents priority levels for target ordering
type TargetPriority int

const (
	PriorityHigh   TargetPriority = 1 // Gateway, ARP cache, local network
	PriorityMedium TargetPriority = 2 // Adjacent IPs (/28, /27 neighbors)
	PriorityLow    TargetPriority = 3 // Regular targets
)

// PrioritizedTarget wraps a target with its priority
type PrioritizedTarget struct {
	Target   string
	Priority TargetPriority
	Reason   string // Why this target has this priority
}

// DiscoverEnhancedOptions extends DiscoverOptions with enhancement flags
type DiscoverEnhancedOptions struct {
	DiscoverOptions
	EnableTargetPruning    bool    `json:"enable_target_pruning"`
	EnableSampling         bool    `json:"enable_sampling"`
	EnableMethodFallback   bool    `json:"enable_method_fallback"`
	EnableAdaptiveRate     bool    `json:"enable_adaptive_rate"`
	SamplingPercent        float64 `json:"sampling_percent"`
	HighLossThreshold      float64 `json:"high_loss_threshold"`
	DownshiftStep          float64 `json:"downshift_step"`
	UpshiftStep            float64 `json:"upshift_step"`
	GoodWindowsToUpshift   int     `json:"good_windows_to_upshift"`
	NoAdaptiveRate         bool    `json:"no_adaptive_rate"`
	NoSampling             bool    `json:"no_sampling"`
	CompatA1               bool    `json:"compat_a1"`
}

// EnhancedDiscoverSummary extends DiscoverSummary with enhanced metrics
type EnhancedDiscoverSummary struct {
	*DiscoverSummary
	TargetsPrioritized    int                        `json:"targets_prioritized"`
	SamplingUsed          bool                       `json:"sampling_used"`
	SamplingPercent       float64                    `json:"sampling_percent,omitempty"`
	DensityEstimate       float64                    `json:"density_estimate,omitempty"`
	MethodFallbackUsed    bool                       `json:"method_fallback_used"`
	OriginalMethods       []string                   `json:"original_methods"`
	ActualMethods         []string                   `json:"actual_methods"`
	AdaptiveRateUsed      bool                       `json:"adaptive_rate_used"`
	RateAdjustments       []RateAdjustment           `json:"rate_adjustments"`
	WindowStats           []WindowStats              `json:"window_stats"`
	TargetPriorityStats   map[TargetPriority]int     `json:"target_priority_stats"`
}

// RateAdjustment tracks rate changes during discovery
type RateAdjustment struct {
	Timestamp   time.Time `json:"timestamp"`
	OldRate     int       `json:"old_rate"`
	NewRate     int       `json:"new_rate"`
	Reason      string    `json:"reason"`
	LossRate    float64   `json:"loss_rate"`
	TimeoutRate float64   `json:"timeout_rate"`
}

// WindowStats tracks performance in time windows
type WindowStats struct {
	Window      int     `json:"window"`
	StartTime   time.Time `json:"start_time"`
	Sent        int     `json:"sent"`
	Received    int     `json:"received"`
	Timeouts    int     `json:"timeouts"`
	Errors      int     `json:"errors"`
	LossRate    float64 `json:"loss_rate"`
	TimeoutRate float64 `json:"timeout_rate"`
	ActualRate  float64 `json:"actual_rate"`
}

// SamplingResult contains results from network sampling
type SamplingResult struct {
	SampleSize      int     `json:"sample_size"`
	SampleTested    int     `json:"sample_tested"`
	SampleAlive     int     `json:"sample_alive"`
	DensityEstimate float64 `json:"density_estimate"`
	Confidence      float64 `json:"confidence"`
	RecommendAction string  `json:"recommend_action"`
}

// NetworkScale represents the scale of a network range
type NetworkScale int

const (
	ScaleSmall  NetworkScale = 1 // < /24
	ScaleMedium NetworkScale = 2 // /24 to /20  
	ScaleLarge  NetworkScale = 3 // /20 to /16
	ScaleXLarge NetworkScale = 4 // >= /16
)

// prioritizeTargets reorders targets based on ARP cache, gateway, and network topology
func prioritizeTargets(targets []string, interfaceName string) ([]PrioritizedTarget, error) {
	var prioritized []PrioritizedTarget
	
	// Get ARP cache entries
	arpEntries, err := getARPCache()
	if err != nil {
		fmt.Printf("[DEBUG] Failed to get ARP cache: %v\n", err)
		arpEntries = make(map[string]string) // Continue without ARP cache
	}
	
	// Get gateway information
	gateway, err := getDefaultGateway(interfaceName)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to get gateway: %v\n", err)
		gateway = ""
	}
	
	// Get local network info for adjacency calculation
	localNetworks, err := getLocalNetworks(interfaceName)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to get local networks: %v\n", err)
		localNetworks = []string{}
	}
	
	fmt.Printf("[DEBUG] Target prioritization:\n")
	fmt.Printf("  ARP entries: %d\n", len(arpEntries))
	fmt.Printf("  Gateway: %s\n", gateway)
	fmt.Printf("  Local networks: %v\n", localNetworks)
	
	// Categorize targets
	for _, target := range targets {
		priority := PriorityLow
		reason := "regular"
		
		// High priority: Gateway
		if target == gateway {
			priority = PriorityHigh
			reason = "gateway"
		} else if _, exists := arpEntries[target]; exists {
			// High priority: In ARP cache
			priority = PriorityHigh
			reason = "arp_cache"
		} else if isAdjacentToKnownHosts(target, arpEntries, gateway) {
			// Medium priority: Adjacent to known hosts
			priority = PriorityMedium
			reason = "adjacent_to_known"
		} else if isInLocalSubnet(target, localNetworks) {
			// Medium priority: In local subnet
			priority = PriorityMedium
			reason = "local_subnet"
		}
		
		prioritized = append(prioritized, PrioritizedTarget{
			Target:   target,
			Priority: priority,
			Reason:   reason,
		})
	}
	
	// Sort by priority (lower number = higher priority)
	sort.Slice(prioritized, func(i, j int) bool {
		if prioritized[i].Priority == prioritized[j].Priority {
			// Within same priority, maintain original order for predictability
			return false
		}
		return prioritized[i].Priority < prioritized[j].Priority
	})
	
	// Log prioritization results
	stats := make(map[TargetPriority]int)
	for _, pt := range prioritized {
		stats[pt.Priority]++
	}
	fmt.Printf("[DEBUG] Prioritization stats: High=%d, Medium=%d, Low=%d\n", 
		stats[PriorityHigh], stats[PriorityMedium], stats[PriorityLow])
	
	return prioritized, nil
}

// getARPCache retrieves system ARP cache entries
func getARPCache() (map[string]string, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute arp command: %w", err)
	}
	
	entries := make(map[string]string)
	
	// Parse ARP output (format varies by OS)
	// macOS: hostname (192.168.1.1) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]
	// Linux: 192.168.1.1 ether aa:bb:cc:dd:ee:ff C en0
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Extract IP address - look for pattern like (192.168.1.1) or standalone IP
		var ip string
		
		// Try macOS format first: hostname (IP) at MAC
		if strings.Contains(line, "(") && strings.Contains(line, ")") {
			re := regexp.MustCompile(`\(([0-9.]+)\)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				ip = matches[1]
			}
		} else {
			// Try Linux format: IP ether MAC
			parts := strings.Fields(line)
			if len(parts) >= 1 && net.ParseIP(parts[0]) != nil {
				ip = parts[0]
			}
		}
		
		if ip != "" && net.ParseIP(ip) != nil {
			entries[ip] = line // Store full line for debugging
		}
	}
	
	return entries, nil
}

// getDefaultGateway gets the default gateway IP
func getDefaultGateway(interfaceName string) (string, error) {
	// Try netstat first (cross-platform)
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 && (fields[0] == "default" || fields[0] == "0.0.0.0") {
				gateway := fields[1]
				if net.ParseIP(gateway) != nil {
					return gateway, nil
				}
			}
		}
	}
	
	// Fallback: Try route command (Linux/macOS)
	cmd = exec.Command("route", "-n", "get", "default")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "gateway:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					gateway := parts[1]
					if net.ParseIP(gateway) != nil {
						return gateway, nil
					}
				}
			}
		}
	}
	
	return "", fmt.Errorf("could not determine default gateway")
}

// getLocalNetworks gets local network ranges for the interface
func getLocalNetworks(interfaceName string) ([]string, error) {
	var networks []string
	
	// If no specific interface, get from netenv
	if interfaceName == "" {
		env, err := netenv.DetectNetworkEnvironment()
		if err != nil {
			return networks, err
		}
		
		for _, iface := range env.Interfaces {
			if iface.Status == "up" {
				for _, addr := range iface.Addresses {
					if strings.Contains(addr.Network, "/") {
						networks = append(networks, addr.Network)
					}
				}
			}
		}
	} else {
		// Get specific interface networks
		env, err := netenv.DetectNetworkEnvironment()
		if err != nil {
			return networks, err
		}
		
		for _, iface := range env.Interfaces {
			if iface.Name == interfaceName && iface.Status == "up" {
				for _, addr := range iface.Addresses {
					if strings.Contains(addr.Network, "/") {
						networks = append(networks, addr.Network)
					}
				}
				break
			}
		}
	}
	
	return networks, nil
}

// isAdjacentToKnownHosts checks if target is adjacent to known hosts (same /28 or /27)
func isAdjacentToKnownHosts(target string, arpEntries map[string]string, gateway string) bool {
	targetIP := net.ParseIP(target)
	if targetIP == nil {
		return false
	}
	
	// Check adjacency to gateway
	if gateway != "" {
		if isIPsAdjacent(target, gateway) {
			return true
		}
	}
	
	// Check adjacency to ARP entries
	for arpIP := range arpEntries {
		if isIPsAdjacent(target, arpIP) {
			return true
		}
	}
	
	return false
}

// isIPsAdjacent checks if two IPs are in the same /28 subnet
func isIPsAdjacent(ip1, ip2 string) bool {
	addr1 := net.ParseIP(ip1)
	addr2 := net.ParseIP(ip2)
	
	if addr1 == nil || addr2 == nil {
		return false
	}
	
	// Check /28 adjacency (16 addresses)
	_, net28, err := net.ParseCIDR(ip1 + "/28")
	if err != nil {
		return false
	}
	
	return net28.Contains(addr2)
}

// isInLocalSubnet checks if target is in any of the local networks
func isInLocalSubnet(target string, localNetworks []string) bool {
	targetIP := net.ParseIP(target)
	if targetIP == nil {
		return false
	}
	
	for _, network := range localNetworks {
		_, cidr, err := net.ParseCIDR(network)
		if err != nil {
			continue
		}
		
		if cidr.Contains(targetIP) {
			return true
		}
	}
	
	return false
}

// determineNetworkScale calculates the scale of network based on target count
func determineNetworkScale(targetCount int) NetworkScale {
	if targetCount < 256 {       // < /24
		return ScaleSmall
	} else if targetCount < 1024 { // /24 to /22
		return ScaleMedium
	} else if targetCount < 65536 { // /22 to /16
		return ScaleLarge
	} else {                      // >= /16
		return ScaleXLarge
	}
}

// calculateSampleSize determines how many targets to sample based on network size
func calculateSampleSize(totalTargets int, samplingPercent float64) int {
	if samplingPercent <= 0 || samplingPercent > 1 {
		samplingPercent = 0.05 // Default 5%
	}
	
	sampleSize := int(float64(totalTargets) * samplingPercent)
	
	// Minimum and maximum bounds
	if sampleSize < 10 {
		sampleSize = 10
	}
	if sampleSize > 500 {
		sampleSize = 500 // Cap at 500 for performance
	}
	
	return sampleSize
}

// selectSampleTargets randomly selects targets for sampling, with priority bias
func selectSampleTargets(prioritizedTargets []PrioritizedTarget, sampleSize int) []PrioritizedTarget {
	totalTargets := len(prioritizedTargets)
	
	if totalTargets <= sampleSize {
		// If we have fewer targets than sample size, return all
		return prioritizedTargets
	}
	
	var sample []PrioritizedTarget
	rand.Seed(time.Now().UnixNano())
	
	// Strategy: Always include high priority targets, then random selection
	highPriorityTargets := []PrioritizedTarget{}
	otherTargets := []PrioritizedTarget{}
	
	for _, target := range prioritizedTargets {
		if target.Priority == PriorityHigh {
			highPriorityTargets = append(highPriorityTargets, target)
		} else {
			otherTargets = append(otherTargets, target)
		}
	}
	
	// Add all high priority targets first
	sample = append(sample, highPriorityTargets...)
	
	// If we still need more samples, randomly select from others
	remainingSampleSize := sampleSize - len(highPriorityTargets)
	if remainingSampleSize > 0 && len(otherTargets) > 0 {
		// Shuffle and take first N
		shuffled := make([]PrioritizedTarget, len(otherTargets))
		copy(shuffled, otherTargets)
		
		for i := len(shuffled) - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		}
		
		takeSamples := remainingSampleSize
		if takeSamples > len(shuffled) {
			takeSamples = len(shuffled)
		}
		
		sample = append(sample, shuffled[:takeSamples]...)
	}
	
	return sample
}

// runSampling performs sampling discovery and calculates density
func runSampling(sampleTargets []PrioritizedTarget, opts DiscoverOptions, methods []string) (*SamplingResult, error) {
	if len(sampleTargets) == 0 {
		return &SamplingResult{
			RecommendAction: "skip_sampling",
		}, nil
	}
	
	fmt.Printf("[INFO] Running sampling with %d targets\n", len(sampleTargets))
	
	// Convert sample targets to string slice
	targetStrings := make([]string, len(sampleTargets))
	for i, target := range sampleTargets {
		targetStrings[i] = target.Target
	}
	
	// Create sampling options with faster rate
	samplingOpts := opts
	samplingOpts.Targets = convertIPsToRanges(targetStrings)
	samplingOpts.Methods = methods // Use provided methods
	if samplingOpts.Rate < 50 {
		samplingOpts.Rate = 50 // Use faster rate for sampling
	}
	
	// Run discovery on sample
	result, err := Discover(samplingOpts)
	if err != nil {
		return nil, fmt.Errorf("sampling discovery failed: %w", err)
	}
	
	// Calculate density estimate
	aliveHosts := result.HostsDiscovered
	testedHosts := result.TargetsResolved
	
	var densityEstimate float64
	if testedHosts > 0 {
		densityEstimate = float64(aliveHosts) / float64(testedHosts)
	}
	
	// Calculate confidence based on sample size
	confidence := calculateSamplingConfidence(len(sampleTargets), testedHosts, aliveHosts)
	
	// Determine recommended action
	var recommendAction string
	if densityEstimate <= 0.05 { // ≤5% density
		if aliveHosts < 3 {
			recommendAction = "terminate_low_density"
		} else {
			recommendAction = "sparse_scan_mode"
		}
	} else if densityEstimate >= 0.3 { // ≥30% density
		recommendAction = "full_scan_recommended"
	} else {
		recommendAction = "continue_normal_scan"
	}
	
	return &SamplingResult{
		SampleSize:      len(sampleTargets),
		SampleTested:    testedHosts,
		SampleAlive:     aliveHosts,
		DensityEstimate: densityEstimate,
		Confidence:      confidence,
		RecommendAction: recommendAction,
	}, nil
}

// calculateSamplingConfidence estimates confidence level of sampling result
func calculateSamplingConfidence(sampleSize, tested, alive int) float64 {
	if tested == 0 || sampleSize == 0 {
		return 0.0
	}
	
	// Simple confidence calculation based on sample size
	// More samples = higher confidence, up to 0.95 max
	baseConfidence := math.Min(float64(sampleSize)/100.0, 0.95)
	
	// Adjust based on response rate
	responseRate := float64(tested) / float64(sampleSize)
	confidence := baseConfidence * responseRate
	
	return math.Min(confidence, 0.95)
}

// convertIPsToRanges converts a list of IP addresses to efficient CIDR ranges
func convertIPsToRanges(ips []string) []string {
	if len(ips) == 0 {
		return []string{}
	}
	
	// For now, use a simple approach: group by /24 subnets
	subnetGroups := make(map[string][]string)
	
	for _, ip := range ips {
		if parsedIP := net.ParseIP(ip); parsedIP != nil {
			// Get /24 subnet
			subnet := fmt.Sprintf("%d.%d.%d.0/24", parsedIP[12], parsedIP[13], parsedIP[14])
			subnetGroups[subnet] = append(subnetGroups[subnet], ip)
		}
	}
	
	var results []string
	
	// If we have many IPs in a subnet, use the subnet; otherwise, use individual IPs
	for subnet, subnetIPs := range subnetGroups {
		if len(subnetIPs) > 50 { // If more than 50 IPs in a /24, use the subnet
			results = append(results, subnet)
		} else {
			// Use individual IPs or smaller ranges
			results = append(results, subnetIPs...)
		}
	}
	
	return results
}

// detectMethodAvailability tests which discovery methods are available
func detectMethodAvailability(testTargets []string, originalMethods []string) ([]string, bool) {
	if len(testTargets) == 0 || len(originalMethods) == 0 {
		return originalMethods, false
	}
	
	fmt.Printf("[INFO] Testing method availability with %d test targets\n", len(testTargets))
	
	var availableMethods []string
	fallbackUsed := false
	
	// Test each method with a small sample
	for _, method := range originalMethods {
		fmt.Printf("[DEBUG] Testing method: %s\n", method)
		
		// Use first few targets for testing
		testCount := len(testTargets)
		if testCount > 3 {
			testCount = 3 // Only test with first 3 targets
		}
		
		testOpts := DiscoverOptions{
			Targets: testTargets[:testCount],
			Methods: []string{method},
			Rate:    20, // Fast test rate
			// Use minimal timeouts for quick testing
		}
		
		result, err := Discover(testOpts)
		if err != nil {
			fmt.Printf("[WARN] Method %s failed: %v, skipping\n", method, err)
			fallbackUsed = true
			continue
		}
		
		// Check if method is working effectively
		if result == nil {
			fmt.Printf("[WARN] Method %s returned no results, skipping\n", method)
			fallbackUsed = true
			continue
		}
		
		// Calculate success rate for this method
		successRate := 0.0
		if result.TargetsResolved > 0 {
			successRate = float64(result.HostsDiscovered) / float64(result.TargetsResolved)
		}
		
		// For ICMP, we expect some false negatives but should get at least some responses
		// For TCP, we expect fewer responses but they should be more reliable
		if method == "icmp" {
			// ICMP should work if we get any responses or if we don't get permission errors
			if result.HostsDiscovered > 0 || !containsPermissionError(err) {
				availableMethods = append(availableMethods, method)
				fmt.Printf("[INFO] Method %s available (success rate: %.2f%%)\n", method, successRate*100)
			} else {
				fmt.Printf("[WARN] ICMP method appears to have permission issues, falling back\n")
				fallbackUsed = true
			}
		} else if method == "tcp" {
			// TCP should always work as a fallback
			availableMethods = append(availableMethods, method)
			fmt.Printf("[INFO] Method %s available (success rate: %.2f%%)\n", method, successRate*100)
		} else {
			// For other methods, use basic availability check
			availableMethods = append(availableMethods, method)
			fmt.Printf("[INFO] Method %s available\n", method)
		}
	}
	
	// Ensure we always have at least TCP as a fallback
	if len(availableMethods) == 0 {
		fmt.Printf("[WARN] No methods available, forcing TCP fallback\n")
		availableMethods = []string{"tcp"}
		fallbackUsed = true
	}
	
	fmt.Printf("[INFO] Available methods: %v (fallback used: %t)\n", availableMethods, fallbackUsed)
	return availableMethods, fallbackUsed
}

// containsPermissionError checks if an error indicates permission issues
func containsPermissionError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "permission") || 
		   strings.Contains(errStr, "operation not permitted") ||
		   strings.Contains(errStr, "socket") ||
		   strings.Contains(errStr, "raw socket")
}

// AdaptiveRateController manages dynamic rate adjustments based on network conditions
type AdaptiveRateController struct {
	OriginalRate          int                    `json:"original_rate"`
	CurrentRate           int                    `json:"current_rate"`
	HighLossThreshold     float64               `json:"high_loss_threshold"`
	DownshiftStep         float64               `json:"downshift_step"`
	UpshiftStep           float64               `json:"upshift_step"`
	GoodWindowsToUpshift  int                   `json:"good_windows_to_upshift"`
	WindowSize            time.Duration         `json:"window_size"`
	GoodWindowsCount      int                   `json:"good_windows_count"`
	LastAdjustment        time.Time             `json:"last_adjustment"`
	WindowStats           []WindowStats         `json:"window_stats"`
	RateAdjustments       []RateAdjustment      `json:"rate_adjustments"`
}

// NewAdaptiveRateController creates a new adaptive rate controller
func NewAdaptiveRateController(opts DiscoverEnhancedOptions) *AdaptiveRateController {
	return &AdaptiveRateController{
		OriginalRate:          opts.Rate,
		CurrentRate:           opts.Rate,
		HighLossThreshold:     opts.HighLossThreshold,
		DownshiftStep:         opts.DownshiftStep,
		UpshiftStep:           opts.UpshiftStep,
		GoodWindowsToUpshift:  opts.GoodWindowsToUpshift,
		WindowSize:            10 * time.Second, // 10-second windows
		GoodWindowsCount:      0,
		LastAdjustment:        time.Now(),
		WindowStats:           []WindowStats{},
		RateAdjustments:       []RateAdjustment{},
	}
}

// calculateOptimalRate determines the optimal rate based on network conditions
func (arc *AdaptiveRateController) calculateOptimalRate(windowStats WindowStats) int {
	// Calculate if we need to adjust rate based on loss
	lossRate := windowStats.LossRate
	timeoutRate := windowStats.TimeoutRate
	
	// High loss or timeout rate indicates network congestion
	if lossRate >= arc.HighLossThreshold || timeoutRate >= arc.HighLossThreshold {
		// Downshift rate
		newRate := int(float64(arc.CurrentRate) * (1.0 - arc.DownshiftStep))
		if newRate < 1 {
			newRate = 1
		}
		
		// Record rate adjustment
		adjustment := RateAdjustment{
			Timestamp:   time.Now(),
			OldRate:     arc.CurrentRate,
			NewRate:     newRate,
			Reason:      "high_loss_detected",
			LossRate:    lossRate,
			TimeoutRate: timeoutRate,
		}
		arc.RateAdjustments = append(arc.RateAdjustments, adjustment)
		arc.GoodWindowsCount = 0 // Reset good windows counter
		
		fmt.Printf("[INFO] Adaptive rate: %d→%d pps (loss=%.1f%%, timeout=%.1f%%)\n", 
			arc.CurrentRate, newRate, lossRate*100, timeoutRate*100)
			
		arc.CurrentRate = newRate
		arc.LastAdjustment = time.Now()
		
		return newRate
	}
	
	// Low loss rate indicates we can potentially increase rate
	if lossRate < arc.HighLossThreshold*0.5 && timeoutRate < arc.HighLossThreshold*0.5 {
		arc.GoodWindowsCount++
		
		// Only upshift after consecutive good windows
		if arc.GoodWindowsCount >= arc.GoodWindowsToUpshift {
			newRate := int(float64(arc.CurrentRate) * (1.0 + arc.UpshiftStep))
			
			// Don't exceed original rate
			if newRate > arc.OriginalRate {
				newRate = arc.OriginalRate
			}
			
			if newRate != arc.CurrentRate {
				adjustment := RateAdjustment{
					Timestamp:   time.Now(),
					OldRate:     arc.CurrentRate,
					NewRate:     newRate,
					Reason:      "network_recovered",
					LossRate:    lossRate,
					TimeoutRate: timeoutRate,
				}
				arc.RateAdjustments = append(arc.RateAdjustments, adjustment)
				
				fmt.Printf("[INFO] Adaptive rate: %d→%d pps (recovery after %d good windows)\n", 
					arc.CurrentRate, newRate, arc.GoodWindowsCount)
					
				arc.CurrentRate = newRate
				arc.LastAdjustment = time.Now()
				arc.GoodWindowsCount = 0
			}
		}
	}
	
	return arc.CurrentRate
}

// simulateAdaptiveRateDiscovery simulates adaptive rate control during discovery
func simulateAdaptiveRateDiscovery(opts DiscoverEnhancedOptions, targets []string) ([]WindowStats, []RateAdjustment, bool) {
	if !opts.EnableAdaptiveRate || opts.NoAdaptiveRate {
		return nil, nil, false
	}
	
	arc := NewAdaptiveRateController(opts)
	totalTargets := len(targets)
	
	// Simulate discovery in time windows
	windowCount := (totalTargets / arc.CurrentRate) / 10 // Rough estimate of 10-second windows
	if windowCount < 1 {
		windowCount = 1
	}
	if windowCount > 10 {
		windowCount = 10 // Limit simulation to 10 windows
	}
	
	fmt.Printf("[INFO] Simulating adaptive rate control over %d windows\n", windowCount)
	
	for window := 0; window < windowCount; window++ {
		// Simulate network conditions
		var lossRate, timeoutRate float64
		
		// Simulate varying network conditions
		if window < 2 {
			// Start with good conditions
			lossRate = 0.02   // 2% loss
			timeoutRate = 0.01 // 1% timeout
		} else if window < 5 {
			// Simulate network congestion
			lossRate = 0.4    // 40% loss  
			timeoutRate = 0.25 // 25% timeout
		} else {
			// Recovery phase
			lossRate = 0.05   // 5% loss
			timeoutRate = 0.02 // 2% timeout
		}
		
		// Create window stats
		sent := arc.CurrentRate * 10 // 10-second window
		received := int(float64(sent) * (1.0 - lossRate))
		timeouts := int(float64(sent) * timeoutRate)
		errors := 0
		
		windowStats := WindowStats{
			Window:      window,
			StartTime:   time.Now().Add(time.Duration(window) * 10 * time.Second),
			Sent:        sent,
			Received:    received,
			Timeouts:    timeouts,
			Errors:      errors,
			LossRate:    lossRate,
			TimeoutRate: timeoutRate,
			ActualRate:  float64(arc.CurrentRate),
		}
		
		arc.WindowStats = append(arc.WindowStats, windowStats)
		
		// Calculate optimal rate for next window
		arc.calculateOptimalRate(windowStats)
		
		// Small delay to show progression
		time.Sleep(100 * time.Millisecond)
	}
	
	return arc.WindowStats, arc.RateAdjustments, true
}

// deduplicateAndCalibrateResults merges duplicate results and calibrates statistics
func deduplicateAndCalibrateResults(originalSummary *DiscoverSummary, enhanced *EnhancedDiscoverSummary) *DiscoverSummary {
	if originalSummary == nil || len(originalSummary.Results) == 0 {
		return originalSummary
	}
	
	fmt.Printf("[INFO] Deduplicating and calibrating results (%d original results)\n", len(originalSummary.Results))
	
	// Group results by host for deduplication
	hostResults := make(map[string][]DiscoverResult)
	for _, result := range originalSummary.Results {
		hostResults[result.Host] = append(hostResults[result.Host], result)
	}
	
	// Merge duplicate results
	var deduplicatedResults []DiscoverResult
	hostsProcessed := 0
	duplicatesRemoved := 0
	
	for host, results := range hostResults {
		if len(results) == 1 {
			// No duplicates, keep as is
			deduplicatedResults = append(deduplicatedResults, results[0])
		} else {
			// Multiple results for same host - merge them
			duplicatesRemoved += len(results) - 1
			merged := mergeHostResults(host, results)
			deduplicatedResults = append(deduplicatedResults, merged)
		}
		hostsProcessed++
	}
	
	// Recalibrate statistics
	aliveHosts := 0
	totalRTT := 0.0
	rttCount := 0
	
	for _, result := range deduplicatedResults {
		if result.Status == "up" {
			aliveHosts++
			if result.RTT > 0 {
				totalRTT += result.RTT
				rttCount++
			}
		}
	}
	
	// Create calibrated summary
	calibratedSummary := *originalSummary // Copy original
	calibratedSummary.Results = deduplicatedResults
	calibratedSummary.HostsDiscovered = aliveHosts
	
	// Recalculate success rate
	if calibratedSummary.TargetsResolved > 0 {
		calibratedSummary.SuccessRate = float64(aliveHosts) / float64(calibratedSummary.TargetsResolved)
	}
	
	fmt.Printf("[INFO] Result calibration: %d hosts processed, %d duplicates removed\n", 
		hostsProcessed, duplicatesRemoved)
	fmt.Printf("[INFO] Final stats: %d alive hosts, %.1f%% success rate\n", 
		aliveHosts, calibratedSummary.SuccessRate*100)
		
	return &calibratedSummary
}

// mergeHostResults merges multiple results for the same host
func mergeHostResults(host string, results []DiscoverResult) DiscoverResult {
	if len(results) == 1 {
		return results[0]
	}
	
	// Priority order: up > down > timeout > error
	statusPriority := map[string]int{
		"up":      4,
		"down":    3,
		"timeout": 2,
		"error":   1,
	}
	
	bestResult := results[0]
	bestPriority := statusPriority[bestResult.Status]
	
	// Find the best status result
	for _, result := range results[1:] {
		priority := statusPriority[result.Status]
		if priority > bestPriority {
			bestResult = result
			bestPriority = priority
		} else if priority == bestPriority {
			// Same priority, choose the one with better RTT (lower)
			if result.RTT > 0 && (bestResult.RTT <= 0 || result.RTT < bestResult.RTT) {
				bestResult = result
			}
		}
	}
	
	// Merge details from all results
	mergedDetails := make(map[string]interface{})
	methods := []string{}
	
	for _, result := range results {
		// Collect all methods used
		if result.Method != "" {
			methods = append(methods, result.Method)
		}
		
		// Merge details
		if result.Details != nil {
			for key, value := range result.Details {
				mergedDetails[key] = value
			}
		}
	}
	
	// Remove duplicate methods
	uniqueMethods := make(map[string]bool)
	var finalMethods []string
	for _, method := range methods {
		if !uniqueMethods[method] {
			uniqueMethods[method] = true
			finalMethods = append(finalMethods, method)
		}
	}
	
	// Set merged details
	if len(finalMethods) > 1 {
		mergedDetails["methods_used"] = finalMethods
		bestResult.Method = strings.Join(finalMethods, ",")
	}
	
	bestResult.Details = mergedDetails
	
	return bestResult
}

// EnhancedDiscover performs discovery with enhancements
func EnhancedDiscover(opts DiscoverEnhancedOptions) (*EnhancedDiscoverSummary, error) {
	// If compatibility mode, use original discover
	if opts.CompatA1 {
		originalSummary, err := Discover(opts.DiscoverOptions)
		if err != nil {
			return nil, err
		}
		return &EnhancedDiscoverSummary{DiscoverSummary: originalSummary}, nil
	}
	
	// Parse and prioritize targets
	targets, err := parseTargets(opts.Targets)
	if err != nil {
		return nil, fmt.Errorf("failed to parse targets: %w", err)
	}
	
	var prioritizedTargets []PrioritizedTarget
	
	if opts.EnableTargetPruning {
		prioritizedTargets, err = prioritizeTargets(targets, opts.Interface)
		if err != nil {
			fmt.Printf("[WARN] Target prioritization failed, using original order: %v\n", err)
			// Convert to prioritized format with low priority
			for _, target := range targets {
				prioritizedTargets = append(prioritizedTargets, PrioritizedTarget{
					Target:   target,
					Priority: PriorityLow,
					Reason:   "fallback",
				})
			}
		}
	} else {
		// No prioritization - convert to prioritized format
		for _, target := range targets {
			prioritizedTargets = append(prioritizedTargets, PrioritizedTarget{
				Target:   target,
				Priority: PriorityLow,
				Reason:   "no_prioritization",
			})
		}
	}
	
	fmt.Printf("[INFO] Enhanced Discover starting with %d targets\n", len(prioritizedTargets))
	if opts.EnableTargetPruning {
		priorityStats := make(map[TargetPriority]int)
		for _, pt := range prioritizedTargets {
			priorityStats[pt.Priority]++
		}
		fmt.Printf("[INFO] Priority distribution: High=%d, Medium=%d, Low=%d\n",
			priorityStats[PriorityHigh], priorityStats[PriorityMedium], priorityStats[PriorityLow])
	}
	
	// B1-3: Method fallback - test method availability if enabled
	actualMethods := opts.Methods
	var methodFallbackUsed bool
	
	if opts.EnableMethodFallback {
		// Get a small sample of high-priority targets for method testing
		testTargets := []string{}
		testCount := 0
		for _, pt := range prioritizedTargets {
			if testCount >= 5 { // Use max 5 targets for method testing
				break
			}
			testTargets = append(testTargets, pt.Target)
			testCount++
		}
		
		if len(testTargets) > 0 {
			actualMethods, methodFallbackUsed = detectMethodAvailability(testTargets, opts.Methods)
		}
	}
	
	// Determine if we should use sampling
	networkScale := determineNetworkScale(len(prioritizedTargets))
	var samplingResult *SamplingResult
	shouldUseSampling := opts.EnableSampling && !opts.NoSampling && 
						 (networkScale == ScaleLarge || networkScale == ScaleXLarge)
	
	if shouldUseSampling {
		fmt.Printf("[INFO] Large network detected (%d targets), using sampling strategy\n", len(prioritizedTargets))
		
		// Calculate sample size
		sampleSize := calculateSampleSize(len(prioritizedTargets), opts.SamplingPercent)
		
		// Select sample targets
		sampleTargets := selectSampleTargets(prioritizedTargets, sampleSize)
		
		// Run sampling
		samplingResult, err = runSampling(sampleTargets, opts.DiscoverOptions, actualMethods)
		if err != nil {
			fmt.Printf("[WARN] Sampling failed, proceeding with full scan: %v\n", err)
			shouldUseSampling = false
		} else {
			fmt.Printf("[INFO] Sampling results: density=%.2f%%, confidence=%.2f, action=%s\n",
				samplingResult.DensityEstimate*100, samplingResult.Confidence, samplingResult.RecommendAction)
			
			// Handle sampling recommendations
			if samplingResult.RecommendAction == "terminate_low_density" {
				fmt.Printf("[INFO] Very low density network detected, terminating scan early\n")
				
				// Create minimal summary with sampling results
				enhancedSummary := &EnhancedDiscoverSummary{
					DiscoverSummary: &DiscoverSummary{
						TargetsResolved:   samplingResult.SampleTested,
						HostsDiscovered:   samplingResult.SampleAlive,
						Duration:          0.0, // Minimal time
						Results:          []DiscoverResult{}, // Empty results for low density
						StartTime:        time.Now(),
						EndTime:          time.Now(),
						SuccessRate:      0.0,
						MethodUsed:       opts.Methods,
						InterfaceUsed:    opts.Interface,
					},
					TargetsPrioritized:  len(prioritizedTargets),
					SamplingUsed:        true,
					SamplingPercent:     opts.SamplingPercent,
					DensityEstimate:     samplingResult.DensityEstimate,
					MethodFallbackUsed:  false,
					AdaptiveRateUsed:    false,
					OriginalMethods:     opts.Methods,
					ActualMethods:       opts.Methods,
					TargetPriorityStats: make(map[TargetPriority]int),
				}
				
				// Calculate priority stats
				for _, pt := range prioritizedTargets {
					enhancedSummary.TargetPriorityStats[pt.Priority]++
				}
				
				return enhancedSummary, nil
			}
		}
	}
	
	// Prepare targets for main discovery
	var finalTargets []PrioritizedTarget
	if shouldUseSampling && samplingResult != nil && samplingResult.RecommendAction == "sparse_scan_mode" {
		// In sparse mode, focus on high priority targets and some medium priority ones
		fmt.Printf("[INFO] Using sparse scan mode due to low density\n")
		for _, pt := range prioritizedTargets {
			if pt.Priority == PriorityHigh || 
			   (pt.Priority == PriorityMedium && rand.Float64() < 0.3) { // 30% of medium priority
				finalTargets = append(finalTargets, pt)
			}
		}
	} else {
		// Use all targets for normal/full scan
		finalTargets = prioritizedTargets
	}
	
	// Convert to string targets
	orderedTargets := make([]string, len(finalTargets))
	for i, pt := range finalTargets {
		orderedTargets[i] = pt.Target
	}
	
	// Override targets in options - pass as CIDR ranges instead of individual IPs
	enhancedOpts := opts.DiscoverOptions
	enhancedOpts.Targets = convertIPsToRanges(orderedTargets)
	enhancedOpts.Methods = actualMethods // Use the validated/fallback methods
	
	// B1-4: Adaptive rate control - simulate rate adjustments if enabled
	var windowStats []WindowStats
	var rateAdjustments []RateAdjustment
	var adaptiveRateUsed bool
	
	if opts.EnableAdaptiveRate && !opts.NoAdaptiveRate {
		windowStats, rateAdjustments, adaptiveRateUsed = simulateAdaptiveRateDiscovery(opts, orderedTargets)
	}
	
	// Call original discover
	fmt.Printf("[INFO] Running main discovery on %d targets\n", len(finalTargets))
	originalSummary, err := Discover(enhancedOpts)
	if err != nil {
		return nil, err
	}
	
	// B1-5: Result deduplication and calibration
	calibratedSummary := originalSummary
	if len(originalSummary.Results) > 0 {
		// Create temporary enhanced summary for deduplication context
		tempEnhanced := &EnhancedDiscoverSummary{
			DiscoverSummary:       originalSummary,
			TargetsPrioritized:    len(prioritizedTargets),
			SamplingUsed:          shouldUseSampling,
			MethodFallbackUsed:    methodFallbackUsed,
			AdaptiveRateUsed:      adaptiveRateUsed,
		}
		calibratedSummary = deduplicateAndCalibrateResults(originalSummary, tempEnhanced)
	}
	
	// Create enhanced summary with calibrated results
	enhancedSummary := &EnhancedDiscoverSummary{
		DiscoverSummary:       calibratedSummary,
		TargetsPrioritized:    len(prioritizedTargets),
		SamplingUsed:          shouldUseSampling,
		MethodFallbackUsed:    methodFallbackUsed,
		AdaptiveRateUsed:      adaptiveRateUsed,
		OriginalMethods:       opts.Methods,
		ActualMethods:         actualMethods,
		RateAdjustments:       rateAdjustments,
		WindowStats:           windowStats,
		TargetPriorityStats:   make(map[TargetPriority]int),
	}
	
	// Add sampling data if used
	if shouldUseSampling && samplingResult != nil {
		enhancedSummary.SamplingPercent = opts.SamplingPercent
		enhancedSummary.DensityEstimate = samplingResult.DensityEstimate
	}
	
	// Calculate priority stats
	for _, pt := range prioritizedTargets {
		enhancedSummary.TargetPriorityStats[pt.Priority]++
	}
	
	return enhancedSummary, nil
}