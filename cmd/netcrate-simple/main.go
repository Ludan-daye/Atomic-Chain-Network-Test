package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/netcrate/netcrate/internal/netenv"
	"github.com/netcrate/netcrate/internal/ops"
	"github.com/netcrate/netcrate/internal/quick"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("NetCrate MVP - Simple Test Version")
		fmt.Println("Usage:")
		fmt.Println("  netcrate-simple netenv    - Test network environment detection")
		fmt.Println("  netcrate-simple discover  - Test host discovery")
		fmt.Println("  netcrate-simple scan      - Test port scanning")
		fmt.Println("  netcrate-simple packet    - Test packet sending")
		fmt.Println("  netcrate-simple quick     - Test Quick mode (full pipeline)")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "netenv":
		testNetenv()
	case "discover":
		// Check for enhanced flags
		enhanced := false
		targetPruning := false
		compatA1 := false
		target := "auto"
		
		for i := 2; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--enhanced":
				enhanced = true
			case "--target-pruning":
				targetPruning = true
			case "--compat-a1":
				compatA1 = true
			default:
				if !strings.HasPrefix(os.Args[i], "--") {
					target = os.Args[i]
				}
			}
		}
		
		testDiscoverEnhanced(target, enhanced, targetPruning, compatA1)
	case "scan":
		testScan()
	case "packet":
		testPacket()
	case "quick":
		testQuick()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func testNetenv() {
	fmt.Println("=== Testing Network Environment Detection ===")
	
	result, err := netenv.DetectNetworkEnvironment()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}

func testDiscover() {
	fmt.Println("=== Testing Host Discovery ===")
	
	opts := ops.DiscoverOptions{
		Targets: []string{"auto"},
		Rate:    10, // Slower rate for testing
	}
	
	result, err := ops.Discover(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Discovered %d hosts out of %d targets\n", result.HostsDiscovered, result.TargetsResolved)
	for _, host := range result.Results {
		if host.Status == "up" {
			fmt.Printf("  %s - %s (%.2fms via %s)\n", host.Host, host.Status, host.RTT, host.Method)
		}
	}
}

func testDiscoverEnhanced(target string, enhanced, targetPruning, compatA1 bool) {
	if enhanced || targetPruning {
		fmt.Println("=== Testing Enhanced Host Discovery (B1) ===")
	} else if compatA1 {
		fmt.Println("=== Testing Host Discovery (A1 Compatibility) ===")
	} else {
		fmt.Println("=== Testing Host Discovery ===")
	}
	
	opts := ops.DiscoverOptions{
		Targets: []string{target},
		Rate:    10, // Slower rate for testing
		Methods: []string{"icmp", "tcp"},
	}
	
	// Use enhanced discovery if requested
	if (enhanced || targetPruning) && !compatA1 {
		enhancedOpts := ops.DiscoverEnhancedOptions{
			DiscoverOptions:        opts,
			EnableTargetPruning:    targetPruning || enhanced,
			EnableSampling:         enhanced,
			EnableMethodFallback:   enhanced,
			EnableAdaptiveRate:     enhanced,
			SamplingPercent:        0.05,
			HighLossThreshold:      0.3,
			DownshiftStep:          0.2,
			UpshiftStep:            0.1,
			GoodWindowsToUpshift:   3,
			NoAdaptiveRate:         false,
			NoSampling:            false,
			CompatA1:              compatA1,
		}
		
		if targetPruning || enhanced {
			fmt.Printf("[DEBUG] Enhanced discovery with target pruning enabled\n")
		}
		
		result, err := ops.EnhancedDiscover(enhancedOpts)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		
		// Print enhanced summary
		if result.TargetsPrioritized > 0 {
			fmt.Printf("Target prioritization: %d targets processed\n", result.TargetsPrioritized)
			if len(result.TargetPriorityStats) > 0 {
				high := result.TargetPriorityStats[ops.PriorityHigh]
				medium := result.TargetPriorityStats[ops.PriorityMedium] 
				low := result.TargetPriorityStats[ops.PriorityLow]
				fmt.Printf("Priority: High=%d, Medium=%d, Low=%d\n", high, medium, low)
			}
		}
		
		// Print sampling information
		if result.SamplingUsed {
			fmt.Printf("Sampling: %.1f%% sample rate, estimated density=%.2f%%\n", 
				result.SamplingPercent*100, result.DensityEstimate*100)
		}
		
		// Print method fallback information
		if result.MethodFallbackUsed {
			fmt.Printf("Method fallback: %v → %v\n", result.OriginalMethods, result.ActualMethods)
		}
		
		// Print adaptive rate information
		if result.AdaptiveRateUsed {
			fmt.Printf("Adaptive rate: %d adjustments made\n", len(result.RateAdjustments))
			for _, adj := range result.RateAdjustments {
				fmt.Printf("  %s: %d→%d pps (%s)\n", 
					adj.Timestamp.Format("15:04:05"), adj.OldRate, adj.NewRate, adj.Reason)
			}
		}
		
		fmt.Printf("Discovered %d hosts out of %d targets\n", result.HostsDiscovered, result.TargetsResolved)
		for _, host := range result.Results {
			if host.Status == "up" {
				fmt.Printf("  %s - %s (%.2fms via %s)\n", host.Host, host.Status, host.RTT, host.Method)
			}
		}
	} else {
		// Use original discovery
		result, err := ops.Discover(opts)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		
		fmt.Printf("Discovered %d hosts out of %d targets\n", result.HostsDiscovered, result.TargetsResolved)
		for _, host := range result.Results {
			if host.Status == "up" {
				fmt.Printf("  %s - %s (%.2fms via %s)\n", host.Host, host.Status, host.RTT, host.Method)
			}
		}
	}
}

func testScan() {
	fmt.Println("=== Testing Port Scanning ===")
	
	ports, err := ops.ParsePortSpec("22,80,443")
	if err != nil {
		fmt.Printf("Error parsing ports: %v\n", err)
		return
	}
	
	opts := ops.ScanOptions{
		Targets: []string{"127.0.0.1"},
		Ports:   ports,
		Rate:    10,
	}
	
	result, err := ops.ScanPorts(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Scanned %d total combinations on %d targets\n", result.TotalCombinations, result.TargetsCount)
	for _, portResult := range result.Results {
		if portResult.Status == "open" {
			service := "unknown"
			if portResult.Service != nil {
				service = portResult.Service.Name
			}
			fmt.Printf("  %s:%d - %s (%s)\n", portResult.Host, portResult.Port, portResult.Status, service)
		}
	}
}

func testPacket() {
	fmt.Println("=== Testing Packet Sending ===")
	
	opts := ops.PacketOptions{
		Targets:  []string{"127.0.0.1:80"},
		Template: "connect",
		Count:    1,
	}
	
	result, err := ops.SendPackets(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Sent %d total packets to %d targets\n", result.TotalPackets, result.TargetsCount)
	for _, packetResult := range result.Results {
		fmt.Printf("  %s - %s (%.2fms)\n", packetResult.Target, packetResult.Status, packetResult.RTT)
	}
}

func testQuick() {
	fmt.Println("=== Testing Quick Mode ===")
	
	// Check command line arguments for test options
	dryRun := true
	interactive := false
	
	// Parse simple arguments
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--real":
			dryRun = false
			fmt.Println("⚠️ Running REAL network scan (this may take time)")
		case "--interactive":
			interactive = true
			fmt.Println("🎛️ Running in INTERACTIVE mode")
		}
	}
	
	if dryRun && !interactive {
		fmt.Println("🧪 Running DRY RUN (use '--real' for actual scan, '--interactive' for configuration)")
	}
	
	result, err := quick.RunQuickMode(dryRun, !interactive, interactive) // skipConfirm=!interactive
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	if result != nil {
		quick.PrintQuickSummary(result)
	}
}