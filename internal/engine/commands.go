package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/netcrate/netcrate/internal/compliance"
	"github.com/netcrate/netcrate/internal/config"
	"github.com/netcrate/netcrate/internal/netenv"
	"github.com/netcrate/netcrate/internal/ops"
	"github.com/netcrate/netcrate/internal/output"
	"github.com/netcrate/netcrate/internal/quick"
	"github.com/netcrate/netcrate/internal/templates"
	"github.com/spf13/cobra"
)

// applyRateProfile applies the current rate profile to operation options if not explicitly set
func applyRateProfile(rate *int, concurrency *int, timeout *time.Duration) {
	cm, err := config.NewConfigManager()
	if err != nil {
		// If config fails, use defaults - don't block execution
		return
	}
	
	profile := cm.GetCurrentRateProfile()
	
	// Only apply if values are at defaults (0 or very low values)
	if *rate == 0 || *rate == 100 { // 100 is common default
		*rate = profile.Rate
	}
	if *concurrency == 0 || *concurrency == 200 { // 200 is common default
		*concurrency = profile.Concurrency
	}
	if *timeout == 0 || *timeout == time.Second { // 1s is common default
		*timeout = profile.Timeout
	}
}

// NewQuickCommand creates the quick wizard command
func NewQuickCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quick",
		Short: "Zero-config network scanning",
		Long: `Quick mode automatically detects your network interface and performs 
a comprehensive network scan with minimal configuration.

Examples:
  netcrate quick              # Auto-detect and scan local network
  netcrate quick --dry-run    # Show what would be done
  netcrate quick --yes        # Skip confirmation prompts`,
		Run: runQuick,
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "Show what would be done without executing")
	cmd.Flags().Bool("yes", false, "Skip all confirmations")
	cmd.Flags().Bool("interactive", false, "Enable interactive configuration selection")
	cmd.Flags().String("iface", "", "Force specific network interface")
	cmd.Flags().Bool("dangerous", false, "Allow scanning of non-private networks")

	return cmd
}

// runQuick executes the quick mode workflow
func runQuick(cmd *cobra.Command, args []string) {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	skipConfirm, _ := cmd.Flags().GetBool("yes")
	interactive, _ := cmd.Flags().GetBool("interactive")
	dangerousFlag, _ := cmd.Flags().GetBool("dangerous")
	
	// Run compliance check before execution
	checker, err := compliance.NewComplianceChecker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Compliance checker initialization failed: %v\n", err)
		os.Exit(1)
	}
	
	// For quick mode, we need to analyze targets from the detected network
	// This is a simplified approach - in real implementation we'd get targets from quick mode analysis
	targets := []string{"auto-detect"}
	sessionID := fmt.Sprintf("quick-%d", time.Now().Unix())
	
	complianceResult, err := checker.CheckCompliance(sessionID, "quick", "netcrate quick", targets, dangerousFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Compliance violation: %v\n", err)
		os.Exit(1)
	}
	
	if complianceResult.Status == "blocked" {
		fmt.Fprintf(os.Stderr, "❌ Scan blocked by compliance rules: %s\n", complianceResult.BlockReason)
		os.Exit(1)
	}
	
	result, err := quick.RunQuickMode(dryRun, skipConfirm, interactive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Quick模式执行失败: %v\n", err)
		os.Exit(1)
	}
	
	if result != nil {
		quick.PrintQuickSummary(result)
	}
}

// NewOpsCommand creates the ops (atomic operations) command
func NewOpsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ops",
		Short: "Atomic network operations",
		Long:  `Ops mode provides individual network operations like discover, scan, and packet sending.`,
	}

	// Add subcommands
	cmd.AddCommand(newNetenvCommand())
	cmd.AddCommand(newDiscoverCommand())
	cmd.AddCommand(newScanCommand())
	cmd.AddCommand(newPacketCommand())

	return cmd
}

// NewTemplateCommand creates the template management command
func NewTemplateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Manage and run templates",
		Long:  `Template mode allows you to create, manage, and execute reusable network testing templates.`,
	}

	// Add subcommands
	cmd.AddCommand(newTemplateListCommand())
	cmd.AddCommand(newTemplateRunCommand())
	cmd.AddCommand(newTemplateViewCommand())
	cmd.AddCommand(newTemplateIndexCommand())

	return cmd
}

// NewConfigCommand creates the configuration management command
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration settings",
		Long:  `Configuration management for NetCrate settings, rate limits, and compliance options.`,
	}

	// Add subcommands
	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigEditCommand())
	cmd.AddCommand(newConfigResetCommand())

	return cmd
}

// NewOutputCommand creates the output management command
func NewOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "output",
		Short: "Manage scan results and outputs",
		Long:  `Output management for viewing, exporting, and managing scan results.`,
	}

	// Add subcommands
	cmd.AddCommand(newOutputShowCommand())
	cmd.AddCommand(newOutputListCommand())
	cmd.AddCommand(newOutputExportCommand())

	return cmd
}

// Placeholder implementations for subcommands
func newNetenvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "netenv",
		Short: "Network environment detection",
		Long:  `Detect network interfaces, addresses, and system capabilities.`,
		Run: func(cmd *cobra.Command, args []string) {
			runNetenvDetect(cmd)
		},
	}

	// Add flags
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("ping-test", false, "Test gateway connectivity")
	cmd.Flags().String("interface", "auto", "Filter by interface name")
	
	return cmd
}

func newDiscoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover [targets|auto]",
		Short: "Discover active hosts",
		Long:  `Discover active hosts using ICMP, TCP, or ARP methods.`,
		Run: func(cmd *cobra.Command, args []string) {
			runDiscover(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().StringSlice("methods", []string{"icmp", "tcp"}, "Discovery methods (icmp,tcp,arp)")
	cmd.Flags().String("interface", "auto", "Network interface to use")
	cmd.Flags().Int("rate", 100, "Packets per second")
	cmd.Flags().Duration("timeout", 1000*time.Millisecond, "Timeout per target")
	cmd.Flags().Int("concurrency", 200, "Maximum concurrent operations")
	cmd.Flags().IntSlice("tcp-ports", []int{80, 443, 22}, "TCP ports for discovery")
	cmd.Flags().Bool("resolve", false, "Resolve hostnames")
	
	// Enhanced discovery flags
	cmd.Flags().Bool("enhanced", false, "Enable enhanced discovery features (B1)")
	cmd.Flags().Bool("target-pruning", false, "Enable target prioritization (ARP cache, gateway)")
	cmd.Flags().Bool("no-adaptive-rate", false, "Disable adaptive rate control")
	cmd.Flags().Bool("no-sampling", false, "Disable sampling for large ranges")
	cmd.Flags().Bool("compat-a1", false, "Use A1 compatibility mode (disable all enhancements)")
	cmd.Flags().Bool("dangerous", false, "Allow scanning of public networks")

	return cmd
}

func newScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Port scanning operations",
	}
	
	cmd.AddCommand(newScanPortsCommand())
	
	return cmd
}

func newScanPortsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ports",
		Short: "Scan ports on targets",
		Long:  `Scan ports on specified targets using TCP connect, SYN, or UDP methods.`,
		Run: func(cmd *cobra.Command, args []string) {
			runScanPorts(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().StringSlice("targets", []string{}, "Target hosts")
	cmd.Flags().String("ports", "top100", "Ports to scan (top100,top1000,web,database,custom)")
	cmd.Flags().String("scan-type", "auto", "Scan type (connect,syn,udp,auto)")
	cmd.Flags().Bool("service-detection", true, "Enable service detection")
	cmd.Flags().Int("rate", 100, "Packets per second")
	cmd.Flags().Duration("timeout", 800*time.Millisecond, "Timeout per port")
	cmd.Flags().Int("concurrency", 200, "Maximum concurrent connections")
	cmd.Flags().Int("retries", 1, "Retry count for failed connections")
	cmd.Flags().Bool("dangerous", false, "Allow scanning of public networks")

	return cmd
}

func newPacketCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packet",
		Short: "Packet operations",
		Long:  `Packet operations for sending custom packets with various templates.`,
	}

	// Add subcommands
	cmd.AddCommand(newPacketSendCommand())
	cmd.AddCommand(newPacketTemplatesCommand())

	return cmd
}

func newPacketSendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send packets using templates",
		Long:  `Send custom packets to targets using predefined templates.`,
		Run: func(cmd *cobra.Command, args []string) {
			runPacketSend(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().StringSlice("targets", []string{}, "Target endpoints (IP:Port)")
	cmd.Flags().String("template", "connect", "Packet template to use")
	cmd.Flags().StringToString("param", map[string]string{}, "Template parameters (key=value)")
	cmd.Flags().Int("count", 1, "Number of packets per target")
	cmd.Flags().Duration("interval", 100*time.Millisecond, "Interval between packets")
	cmd.Flags().Duration("timeout", 5*time.Second, "Timeout per packet")
	cmd.Flags().Bool("follow-redirects", false, "Follow HTTP redirects")
	cmd.Flags().Int("max-response-size", 1024*1024, "Maximum response size")

	return cmd
}

func newPacketTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "List available packet templates",
		Run: func(cmd *cobra.Command, args []string) {
			runPacketTemplates(cmd, args)
		},
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

func newTemplateListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List available templates",
		Run: func(cmd *cobra.Command, args []string) {
			runTemplateList(cmd, args)
		},
	}
	
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

func newTemplateRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a template",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runTemplateRun(cmd, args)
		},
	}
	
	cmd.Flags().StringSlice("param", []string{}, "Template parameters (key=value)")
	cmd.Flags().Bool("yes", false, "Skip parameter confirmation")
	cmd.Flags().Bool("continue-on-error", false, "Continue execution on step failures")
	cmd.Flags().String("log-level", "info", "Log level (info, debug)")
	cmd.Flags().Bool("dangerous", false, "Allow scanning of public networks")
	
	return cmd
}

func newTemplateViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <name>",
		Short: "View template details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runTemplateView(cmd, args)
		},
	}
}

func newTemplateIndexCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "index",
		Short: "Show template search paths and index debug info",
		Run: func(cmd *cobra.Command, args []string) {
			runTemplateIndex(cmd, args)
		},
	}
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Config show command - Coming soon!")
		},
	}
}

func newConfigEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration interactively",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Config edit command - Coming soon!")
		},
	}
}

func newConfigResetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Config reset command - Coming soon!")
		},
	}
}

func newOutputShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show scan results",
		Long: `Show detailed results from previous scans.
		
Examples:
  netcrate output show --last        # Show latest run
  netcrate output show --run quick_123456  # Show specific run`,
		Run: runOutputShow,
	}

	cmd.Flags().Bool("last", false, "Show the most recent run")
	cmd.Flags().String("run", "", "Show specific run by ID")
	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

func newOutputListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all saved results",
		Long:  `List all saved scan results with summary information.`,
		Run:   runOutputList,
	}
}

func newOutputExportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export results to file",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Output export command - Coming soon!")
		},
	}
}

// Implementation functions

func runNetenvDetect(cmd *cobra.Command) {
	// Get flags
	jsonOutput, _ := cmd.Flags().GetBool("json")
	pingTest, _ := cmd.Flags().GetBool("ping-test")
	interfaceFilter, _ := cmd.Flags().GetString("interface")

	// Detect network environment
	result, err := netenv.DetectNetworkEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting network environment: %v\n", err)
		os.Exit(1)
	}

	// Filter interfaces if specified
	if interfaceFilter != "auto" && interfaceFilter != "" {
		var filtered []netenv.NetworkInterface
		for _, iface := range result.Interfaces {
			if strings.Contains(iface.Name, interfaceFilter) {
				filtered = append(filtered, iface)
			}
		}
		result.Interfaces = filtered
	}

	// Test gateway connectivity if requested
	if pingTest {
		for i := range result.Interfaces {
			if result.Interfaces[i].Gateway != nil {
				err := netenv.PingGateway(result.Interfaces[i].Gateway)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to ping gateway %s: %v\n", 
						result.Interfaces[i].Gateway.IP, err)
				}
			}
		}
	}

	// Output results
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		printNetenvTable(result)
	}
}

func printNetenvTable(result *netenv.DetectResult) {
	fmt.Println("🌐 Network Environment Detection")
	fmt.Println()

	// System Information
	fmt.Printf("📋 System Information:\n")
	fmt.Printf("  Platform: %s\n", result.SystemInfo.Platform)
	fmt.Printf("  Hostname: %s\n", result.SystemInfo.Hostname)
	if len(result.SystemInfo.DNSServers) > 0 {
		fmt.Printf("  DNS Servers: %s\n", strings.Join(result.SystemInfo.DNSServers, ", "))
	}
	if result.SystemInfo.DefaultRoute != "" {
		fmt.Printf("  Default Route: %s\n", result.SystemInfo.DefaultRoute)
	}
	fmt.Println()

	// Capabilities
	fmt.Printf("🔧 Capabilities:\n")
	fmt.Printf("  Raw Socket: %v\n", result.Capabilities.RawSocket)
	fmt.Printf("  Promiscuous Mode: %v\n", result.Capabilities.PromiscuousMode)
	fmt.Printf("  Packet Capture: %v\n", result.Capabilities.PacketCapture)
	fmt.Println()

	// Network Interfaces
	fmt.Printf("🔌 Network Interfaces (%d found):\n", len(result.Interfaces))
	if result.Recommended != "" {
		fmt.Printf("   Recommended: %s\n", result.Recommended)
	}
	fmt.Println()

	for _, iface := range result.Interfaces {
		isRecommended := iface.Name == result.Recommended
		prefix := "  "
		if isRecommended {
			prefix = "▸ "
		}

		fmt.Printf("%s%s (%s)\n", prefix, iface.Name, iface.DisplayName)
		fmt.Printf("    Type: %s | Status: %s | MTU: %d\n", 
			iface.Type, iface.Status, iface.MTU)
		
		if iface.MacAddress != "" {
			fmt.Printf("    MAC: %s\n", iface.MacAddress)
		}

		// Print addresses
		for _, addr := range iface.Addresses {
			fmt.Printf("    IP: %s/%s", addr.IP, addr.Network)
			if addr.Scope != "global" {
				fmt.Printf(" (scope: %s)", addr.Scope)
			}
			fmt.Println()
		}

		// Print gateway information
		if iface.Gateway != nil {
			fmt.Printf("    Gateway: %s", iface.Gateway.IP)
			if iface.Gateway.RTT > 0 {
				fmt.Printf(" (RTT: %.1fms)", iface.Gateway.RTT)
			}
			fmt.Println()
		}

		fmt.Println()
	}

	if len(result.Interfaces) == 0 {
		fmt.Println("  No active interfaces found")
	}
}

func runDiscover(cmd *cobra.Command, args []string) {
	// Get flags
	jsonOutput, _ := cmd.Flags().GetBool("json")
	methods, _ := cmd.Flags().GetStringSlice("methods")
	iface, _ := cmd.Flags().GetString("interface")
	rate, _ := cmd.Flags().GetInt("rate")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	tcpPorts, _ := cmd.Flags().GetIntSlice("tcp-ports")
	resolve, _ := cmd.Flags().GetBool("resolve")
	
	// Apply rate profile if values not explicitly set
	applyRateProfile(&rate, &concurrency, &timeout)
	
	// Enhanced discovery flags
	enhanced, _ := cmd.Flags().GetBool("enhanced")
	targetPruning, _ := cmd.Flags().GetBool("target-pruning")
	noAdaptiveRate, _ := cmd.Flags().GetBool("no-adaptive-rate")
	noSampling, _ := cmd.Flags().GetBool("no-sampling")
	compatA1, _ := cmd.Flags().GetBool("compat-a1")

	// Get targets from arguments
	var targets []string
	if len(args) == 0 {
		targets = []string{"auto"}
	} else {
		targets = args
	}

	// Create discover options
	opts := ops.DiscoverOptions{
		Targets:          targets,
		Methods:          methods,
		Interface:        iface,
		Rate:            rate,
		Timeout:         timeout,
		Concurrency:     concurrency,
		TCPPorts:        tcpPorts,
		ResolveHostnames: resolve,
	}

	// Check if we should use enhanced discovery
	useEnhanced := enhanced || targetPruning || (!noAdaptiveRate && !compatA1) || (!noSampling && !compatA1)
	
	if useEnhanced && !compatA1 {
		// Use enhanced discovery
		enhancedOpts := ops.DiscoverEnhancedOptions{
			DiscoverOptions:        opts,
			EnableTargetPruning:    targetPruning || enhanced,
			EnableSampling:         !noSampling && enhanced,
			EnableMethodFallback:   enhanced,
			EnableAdaptiveRate:     !noAdaptiveRate && enhanced,
			SamplingPercent:        0.05, // 5% for large networks
			HighLossThreshold:      0.3,  // 30%
			DownshiftStep:          0.2,  // 20% reduction
			UpshiftStep:            0.1,  // 10% increase
			GoodWindowsToUpshift:   3,
			NoAdaptiveRate:         noAdaptiveRate,
			NoSampling:            noSampling,
			CompatA1:              compatA1,
		}
		
		// Run enhanced discovery
		fmt.Fprintf(os.Stderr, "🚀 Starting enhanced host discovery (B1)...\n")
		if enhancedOpts.EnableTargetPruning {
			fmt.Fprintf(os.Stderr, "✨ Target prioritization enabled (ARP cache, gateway)\n")
		}
		fmt.Fprintf(os.Stderr, "Targets: %s\n", strings.Join(targets, ", "))
		fmt.Fprintf(os.Stderr, "Methods: %s\n", strings.Join(methods, ", "))
		fmt.Fprintf(os.Stderr, "Rate: %d pps | Concurrency: %d | Timeout: %v\n", rate, concurrency, timeout)
		fmt.Fprintf(os.Stderr, "\n")

		enhancedResult, err := ops.EnhancedDiscover(enhancedOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during enhanced discovery: %v\n", err)
			os.Exit(1)
		}

		// Output results
		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(enhancedResult); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Print enhanced summary first
			printEnhancedDiscoverSummary(enhancedResult)
			// Then print regular table
			printDiscoverTable(enhancedResult.DiscoverSummary)
		}
	} else {
		// Use original discovery
		fmt.Fprintf(os.Stderr, "🔍 Starting host discovery...\n")
		if compatA1 {
			fmt.Fprintf(os.Stderr, "🔄 A1 compatibility mode enabled\n")
		}
		fmt.Fprintf(os.Stderr, "Targets: %s\n", strings.Join(targets, ", "))
		fmt.Fprintf(os.Stderr, "Methods: %s\n", strings.Join(methods, ", "))
		fmt.Fprintf(os.Stderr, "Rate: %d pps | Concurrency: %d | Timeout: %v\n", rate, concurrency, timeout)
		fmt.Fprintf(os.Stderr, "\n")

		result, err := ops.Discover(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during discovery: %v\n", err)
			os.Exit(1)
		}

		// Output results
		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
		} else {
			printDiscoverTable(result)
		}
	}
}

func printDiscoverTable(result *ops.DiscoverSummary) {
	fmt.Printf("🔍 Host Discovery Results\n")
	fmt.Printf("Run ID: %s\n", result.RunID)
	fmt.Printf("Duration: %.1fs\n", result.Duration)
	fmt.Printf("Targets: %d | Discovered: %d | Success Rate: %.1f%%\n", 
		result.TargetsResolved, result.HostsDiscovered, result.SuccessRate*100)
	fmt.Printf("Methods Used: %s\n", strings.Join(result.MethodUsed, ", "))
	fmt.Println()

	if len(result.Results) == 0 {
		fmt.Println("No hosts discovered.")
		return
	}

	// Filter and sort results - show active hosts first
	var activeHosts []ops.DiscoverResult
	var inactiveHosts []ops.DiscoverResult

	for _, r := range result.Results {
		if r.Status == "up" {
			activeHosts = append(activeHosts, r)
		} else {
			inactiveHosts = append(inactiveHosts, r)
		}
	}

	// Print active hosts
	if len(activeHosts) > 0 {
		fmt.Printf("✅ Active Hosts (%d):\n", len(activeHosts))
		fmt.Printf("%-15s %-8s %-8s %-10s %s\n", "Host", "Status", "RTT", "Method", "Details")
		fmt.Println(strings.Repeat("-", 60))

		for _, host := range activeHosts {
			rttStr := fmt.Sprintf("%.1fms", host.RTT)
			details := ""
			if host.Hostname != "" {
				details = fmt.Sprintf("(%s)", host.Hostname)
			}
			if port, ok := host.Details["tcp_port"]; ok {
				details = fmt.Sprintf("port %v", port)
			}

			fmt.Printf("%-15s %-8s %-8s %-10s %s\n", 
				host.Host, host.Status, rttStr, host.Method, details)
		}
		fmt.Println()
	}

	// Print summary statistics
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("  Total Sent: %d\n", result.Stats.Sent)
	fmt.Printf("  Responses: %d\n", result.Stats.Received)
	fmt.Printf("  Timeouts: %d\n", result.Stats.Timeouts)
	fmt.Printf("  Errors: %d\n", result.Stats.Errors)
	fmt.Println()

	// Print method breakdown
	if len(result.Stats.MethodBreakdown) > 0 {
		fmt.Printf("🔧 Method Breakdown:\n")
		for method, stats := range result.Stats.MethodBreakdown {
			successRate := float64(0)
			if stats.Sent > 0 {
				successRate = float64(stats.Received) / float64(stats.Sent) * 100
			}
			fmt.Printf("  %s: %d/%d (%.1f%%)\n", method, stats.Received, stats.Sent, successRate)
		}
		fmt.Println()
	}

	// Show inactive hosts summary (don't spam with details)
	if len(inactiveHosts) > 0 {
		fmt.Printf("❌ Inactive Hosts: %d\n", len(inactiveHosts))
		fmt.Printf("   Use --json flag to see full details\n")
	}
}

func runPacketSend(cmd *cobra.Command, args []string) {
	// Get flags
	jsonOutput, _ := cmd.Flags().GetBool("json")
	targets, _ := cmd.Flags().GetStringSlice("targets")
	template, _ := cmd.Flags().GetString("template")
	params, _ := cmd.Flags().GetStringToString("param")
	count, _ := cmd.Flags().GetInt("count")
	interval, _ := cmd.Flags().GetDuration("interval")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	followRedirects, _ := cmd.Flags().GetBool("follow-redirects")
	maxResponseSize, _ := cmd.Flags().GetInt("max-response-size")

	// Get targets from arguments if not provided via flags
	if len(targets) == 0 && len(args) > 0 {
		targets = args
	}

	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No targets specified\n")
		fmt.Fprintf(os.Stderr, "Use: netcrate ops packet send --targets 192.168.1.1:80 --template http\n")
		os.Exit(1)
	}

	// Convert string params to interface{} map
	templateParams := make(map[string]interface{})
	for k, v := range params {
		templateParams[k] = v
	}

	// Create packet options
	opts := ops.PacketOptions{
		Targets:         targets,
		Template:        template,
		TemplateParams:  templateParams,
		Count:           count,
		Interval:        interval,
		Timeout:         timeout,
		FollowRedirects: followRedirects,
		MaxResponseSize: maxResponseSize,
	}

	// Run packet sending
	fmt.Fprintf(os.Stderr, "📦 Sending packets...\n")
	fmt.Fprintf(os.Stderr, "Template: %s\n", template)
	fmt.Fprintf(os.Stderr, "Targets: %s\n", strings.Join(targets, ", "))
	fmt.Fprintf(os.Stderr, "Count: %d | Interval: %v | Timeout: %v\n", count, interval, timeout)
	fmt.Fprintf(os.Stderr, "\n")

	result, err := ops.SendPackets(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending packets: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		printPacketTable(result)
	}
}

func runPacketTemplates(cmd *cobra.Command, args []string) {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(ops.PacketTemplates); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		printPacketTemplatesTable()
	}
}

func printPacketTable(result *ops.PacketSummary) {
	fmt.Printf("📦 Packet Send Results\n")
	fmt.Printf("Run ID: %s\n", result.RunID)
	fmt.Printf("Template: %s\n", result.TemplateUsed)
	fmt.Printf("Targets: %d | Total Packets: %d | Successful: %d | Success Rate: %.1f%%\n",
		result.TargetsCount, result.TotalPackets, result.SuccessfulResponses,
		result.Stats.SuccessRate*100)
	fmt.Println()

	if len(result.Results) == 0 {
		fmt.Println("No results.")
		return
	}

	// Group results by target
	targetResults := make(map[string][]ops.PacketResult)
	for _, r := range result.Results {
		targetResults[r.Target] = append(targetResults[r.Target], r)
	}

	// Print results for each target
	for target, results := range targetResults {
		fmt.Printf("🎯 Target: %s\n", target)
		fmt.Printf("%-3s %-8s %-8s %-12s %s\n", "Seq", "Status", "RTT", "Method", "Details")
		fmt.Println(strings.Repeat("-", 50))

		for _, result := range results {
			rttStr := fmt.Sprintf("%.1fms", result.RTT)
			method := result.Request.Method
			details := ""

			if result.Status == "success" && result.Response != nil {
				if result.Response.StatusCode > 0 {
					details = fmt.Sprintf("HTTP %d", result.Response.StatusCode)
				}
				if result.Response.TLSVersion != "" {
					details += fmt.Sprintf(" TLS %s", result.Response.TLSVersion)
				}
				if result.Response.BodySize > 0 {
					details += fmt.Sprintf(" (%d bytes)", result.Response.BodySize)
				}
			} else if result.Error != nil {
				details = result.Error.Type
			}

			status := result.Status
			if result.Status == "success" {
				status = "✅"
			} else if result.Status == "error" {
				status = "❌"
			}

			fmt.Printf("%-3d %-8s %-8s %-12s %s\n",
				result.Sequence, status, rttStr, method, details)
		}
		fmt.Println()
	}

	// Print statistics
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("  Average RTT: %.1fms\n", result.Stats.AvgRTT)
	fmt.Printf("  Min RTT: %.1fms\n", result.Stats.MinRTT)
	fmt.Printf("  Max RTT: %.1fms\n", result.Stats.MaxRTT)
	fmt.Printf("  Success Rate: %.1f%%\n", result.Stats.SuccessRate*100)
	fmt.Println()

	// Print status code breakdown for HTTP(S)
	if len(result.Stats.ByStatusCode) > 0 {
		fmt.Printf("🔢 HTTP Status Codes:\n")
		for code, count := range result.Stats.ByStatusCode {
			fmt.Printf("  %s: %d\n", code, count)
		}
		fmt.Println()
	}
}

func printPacketTemplatesTable() {
	fmt.Printf("📦 Available Packet Templates\n")
	fmt.Println()

	for name, template := range ops.PacketTemplates {
		fmt.Printf("🔹 %s\n", name)
		fmt.Printf("   Description: %s\n", template.Description)

		if len(template.RequiredParams) > 0 {
			fmt.Printf("   Required Parameters: %s\n", strings.Join(template.RequiredParams, ", "))
		}

		if len(template.OptionalParams) > 0 {
			fmt.Printf("   Optional Parameters: %s\n", strings.Join(template.OptionalParams, ", "))
		}

		if template.RequiresRawSocket {
			fmt.Printf("   ⚠️  Requires raw socket privileges\n")
		}

		if len(template.DefaultParams) > 0 {
			fmt.Printf("   Defaults: ")
			var defaults []string
			for k, v := range template.DefaultParams {
				defaults = append(defaults, fmt.Sprintf("%s=%v", k, v))
			}
			fmt.Printf("%s\n", strings.Join(defaults, ", "))
		}

		fmt.Println()
	}

	fmt.Printf("Usage examples:\n")
	fmt.Printf("  netcrate ops packet send --targets 192.168.1.1:80 --template http\n")
	fmt.Printf("  netcrate ops packet send --targets example.com:443 --template https --param path=/api\n")
	fmt.Printf("  netcrate ops packet send --targets 192.168.1.1:22 --template connect\n")
	fmt.Printf("  netcrate ops packet templates --json\n")
}

func runScanPorts(cmd *cobra.Command, args []string) {
	// Get flags
	jsonOutput, _ := cmd.Flags().GetBool("json")
	targets, _ := cmd.Flags().GetStringSlice("targets")
	portsSpec, _ := cmd.Flags().GetString("ports")
	scanType, _ := cmd.Flags().GetString("scan-type")
	serviceDetection, _ := cmd.Flags().GetBool("service-detection")
	rate, _ := cmd.Flags().GetInt("rate")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	retries, _ := cmd.Flags().GetInt("retries")
	
	// Apply rate profile if values not explicitly set
	applyRateProfile(&rate, &concurrency, &timeout)

	// Get targets from arguments if not provided via flags
	if len(targets) == 0 && len(args) > 0 {
		targets = args
	}

	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No targets specified\n")
		fmt.Fprintf(os.Stderr, "Use: netcrate ops scan ports --targets 192.168.1.1,192.168.1.2 --ports top100\n")
		os.Exit(1)
	}

	// Parse port specification
	ports, err := ops.ParsePortSpec(portsSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing ports '%s': %v\n", portsSpec, err)
		os.Exit(1)
	}

	// Create scan options
	opts := ops.ScanOptions{
		Targets:          targets,
		Ports:            ports,
		ScanType:         scanType,
		ServiceDetection: serviceDetection,
		Rate:             rate,
		Timeout:          timeout,
		Concurrency:      concurrency,
		RetryCount:       retries,
	}

	// Run port scanning
	fmt.Fprintf(os.Stderr, "🔌 Starting port scan...\n")
	fmt.Fprintf(os.Stderr, "Targets: %s\n", strings.Join(targets, ", "))
	fmt.Fprintf(os.Stderr, "Ports: %s (%d ports)\n", portsSpec, len(ports))
	fmt.Fprintf(os.Stderr, "Type: %s | Rate: %d pps | Concurrency: %d | Timeout: %v\n", 
		scanType, rate, concurrency, timeout)
	fmt.Fprintf(os.Stderr, "\n")

	result, err := ops.ScanPorts(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during port scan: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		printScanTable(result)
	}
}

func printScanTable(result *ops.ScanSummary) {
	fmt.Printf("🔌 Port Scan Results\n")
	fmt.Printf("Run ID: %s\n", result.RunID)
	fmt.Printf("Duration: %.1fs\n", result.Duration)
	fmt.Printf("Targets: %d | Combinations: %d | Open Ports: %d | Success Rate: %.1f%%\n",
		result.TargetsCount, result.TotalCombinations, result.OpenPorts, 
		result.Stats.SuccessRate*100)
	fmt.Printf("Scan Type: %s\n", result.ScanTypeUsed)
	fmt.Println()

	if len(result.Results) == 0 {
		fmt.Println("No results.")
		return
	}

	// Filter and group results - show open ports first
	var openPorts []ops.ScanResult
	var otherPorts []ops.ScanResult

	for _, r := range result.Results {
		if r.Status == "open" {
			openPorts = append(openPorts, r)
		} else {
			otherPorts = append(otherPorts, r)
		}
	}

	// Print open ports
	if len(openPorts) > 0 {
		fmt.Printf("✅ Open Ports (%d):\n", len(openPorts))
		fmt.Printf("%-15s %-6s %-8s %-8s %-12s %s\n", "Host", "Port", "Status", "RTT", "Service", "Details")
		fmt.Println(strings.Repeat("-", 70))

		for _, port := range openPorts {
			rttStr := fmt.Sprintf("%.1fms", port.RTT)
			service := "unknown"
			details := ""

			if port.Service != nil {
				service = port.Service.Name
				if port.Service.Version != "" {
					details = port.Service.Version
				} else if port.Service.Banner != "" {
					details = truncateString(port.Service.Banner, 30)
				}
				if port.Service.Confidence < 0.7 {
					service += "?"
				}
			}

			fmt.Printf("%-15s %-6d %-8s %-8s %-12s %s\n",
				port.Host, port.Port, port.Status, rttStr, service, details)
		}
		fmt.Println()
	}

	// Print summary statistics
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("  Hosts Scanned: %d\n", result.Stats.HostsScanned)
	fmt.Printf("  Ports Scanned: %d\n", result.Stats.PortsScanned)
	fmt.Printf("  Average RTT: %.1fms\n", result.Stats.AvgRTT)
	fmt.Printf("  Scan Rate: %.1f pps\n", result.Stats.ScanRate)
	fmt.Println()

	// Print port status breakdown
	fmt.Printf("🔧 Port Status:\n")
	fmt.Printf("  Open: %d\n", result.Stats.ByStatus["open"])
	fmt.Printf("  Closed: %d\n", result.Stats.ByStatus["closed"])
	fmt.Printf("  Filtered: %d\n", result.Stats.ByStatus["filtered"])
	if result.Stats.ByStatus["error"] > 0 {
		fmt.Printf("  Errors: %d\n", result.Stats.ByStatus["error"])
	}
	fmt.Println()

	// Print service breakdown
	if len(result.Stats.ByService) > 0 {
		fmt.Printf("🔍 Services Detected:\n")
		for service, count := range result.Stats.ByService {
			if service != "unknown" || count > 0 {
				fmt.Printf("  %s: %d\n", service, count)
			}
		}
		fmt.Println()
	}

	// Show summary for non-open ports
	nonOpenCount := len(otherPorts)
	if nonOpenCount > 0 {
		fmt.Printf("❌ Non-Open Ports: %d\n", nonOpenCount)
		fmt.Printf("   Use --json flag to see full details\n")
	}
}

// Helper function for string truncation
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// runOutputShow handles the output show command
func runOutputShow(cmd *cobra.Command, args []string) {
	showLast, _ := cmd.Flags().GetBool("last")
	runID, _ := cmd.Flags().GetString("run")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	var runInfo *output.RunInfo
	var err error

	if showLast {
		runInfo, err = output.GetLastRun()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 获取最近运行失败: %v\n", err)
			os.Exit(1)
		}
	} else if runID != "" {
		runInfo, err = output.GetRunByID(runID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 找不到运行 '%s': %v\n", runID, err)
			os.Exit(1)
		}
	} else {
		// Show latest by default
		runInfo, err = output.GetLastRun()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 没有找到保存的运行结果\n")
			fmt.Printf("运行 'netcrate quick' 来创建你的第一次扫描\n")
			os.Exit(1)
		}
	}

	if jsonOutput {
		result, err := output.LoadQuickResult(runInfo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 加载结果失败: %v\n", err)
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(result)
	} else {
		err = output.PrintRunDetails(runInfo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 显示结果失败: %v\n", err)
			os.Exit(1)
		}
		
		// Show compliance summary
		checker, err := compliance.NewComplianceChecker()
		if err == nil {
			if summary, err := checker.GetComplianceSummary(); err == nil && summary.TotalChecks > 0 {
				fmt.Printf("\n📋 Compliance Summary:\n")
				fmt.Printf("======================\n")
				fmt.Printf("Total checks: %d\n", summary.TotalChecks)
				fmt.Printf("Allowed scans: %d\n", summary.AllowedScans)
				fmt.Printf("Blocked scans: %d\n", summary.BlockedScans)
				fmt.Printf("Public targets: %d\n", summary.PublicTargets)
				fmt.Printf("Private targets: %d\n", summary.PrivateTargets)
				if summary.LastCheck != "" {
					fmt.Printf("Last check: %s\n", summary.LastCheck)
				}
			}
		}
	}
}

// printEnhancedDiscoverSummary prints summary of enhanced discovery features
func printEnhancedDiscoverSummary(result *ops.EnhancedDiscoverSummary) {
	fmt.Fprintf(os.Stderr, "📈 Enhanced Discovery Summary (B1)\n")
	fmt.Fprintf(os.Stderr, "=====================================\n")
	
	// Target prioritization info
	if result.TargetsPrioritized > 0 {
		fmt.Fprintf(os.Stderr, "🎯 Target prioritization: %d targets processed\n", result.TargetsPrioritized)
		if len(result.TargetPriorityStats) > 0 {
			high := result.TargetPriorityStats[ops.PriorityHigh]
			medium := result.TargetPriorityStats[ops.PriorityMedium]
			low := result.TargetPriorityStats[ops.PriorityLow]
			fmt.Fprintf(os.Stderr, "   Priority distribution: High=%d, Medium=%d, Low=%d\n", high, medium, low)
		}
	}
	
	// Sampling info
	if result.SamplingUsed {
		fmt.Fprintf(os.Stderr, "📊 Sampling: %.1f%% of targets, estimated density: %.2f\n", 
			result.SamplingPercent*100, result.DensityEstimate)
	}
	
	// Method fallback info
	if result.MethodFallbackUsed {
		fmt.Fprintf(os.Stderr, "🔄 Method fallback: %s → %s\n", 
			strings.Join(result.OriginalMethods, ","), strings.Join(result.ActualMethods, ","))
	}
	
	// Adaptive rate info
	if result.AdaptiveRateUsed && len(result.RateAdjustments) > 0 {
		fmt.Fprintf(os.Stderr, "⚡ Rate adjustments: %d changes\n", len(result.RateAdjustments))
		for _, adj := range result.RateAdjustments {
			fmt.Fprintf(os.Stderr, "   %s: %dpps → %dpps (%s)\n", 
				adj.Timestamp.Format("15:04:05"), adj.OldRate, adj.NewRate, adj.Reason)
		}
	}
	
	fmt.Fprintf(os.Stderr, "\n")
}

// runOutputList handles the output list command
func runOutputList(cmd *cobra.Command, args []string) {
	runs, err := output.ListRuns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 获取运行列表失败: %v\n", err)
		os.Exit(1)
	}

	output.PrintRunsList(runs)
}

// Template command implementations

// runTemplateList handles the template list command
func runTemplateList(cmd *cobra.Command, args []string) {
	registry := templates.NewRegistry()
	if err := registry.LoadTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading templates: %v\n", err)
		os.Exit(1)
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	templateList := registry.List()

	if jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(templateList); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Table output
	fmt.Printf("📋 Available Templates (%d)\n\n", len(templateList))
	
	if len(templateList) == 0 {
		fmt.Println("No templates found.")
		fmt.Println("\nTo get started:")
		fmt.Printf("  1. Check search paths: netcrate templates index\n")
		fmt.Printf("  2. Create user template directory: mkdir -p ~/.netcrate/templates/\n")
		fmt.Printf("  3. Copy example template: cp templates/builtin/basic_scan.yaml ~/.netcrate/templates/\n")
		return
	}

	for _, template := range templateList {
		fmt.Printf("🔹 %s v%s (%s)\n", template.Name, template.Version, template.Source)
		fmt.Printf("   %s\n", template.Description)
		
		if len(template.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(template.Tags, ", "))
		}
		
		if len(template.Parameters) > 0 {
			fmt.Printf("   Parameters: %d\n", len(template.Parameters))
		}
		
		fmt.Println()
	}
}

// runTemplateView handles the template view command
func runTemplateView(cmd *cobra.Command, args []string) {
	templateName := args[0]
	
	registry := templates.NewRegistry()
	if err := registry.LoadTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading templates: %v\n", err)
		os.Exit(1)
	}

	template, exists := registry.Get(templateName)
	if !exists {
		fmt.Fprintf(os.Stderr, "Template '%s' not found.\n", templateName)
		fmt.Fprintf(os.Stderr, "Use 'netcrate templates ls' to list available templates.\n")
		os.Exit(1)
	}

	// Display template details
	fmt.Printf("📄 Template: %s\n", template.Name)
	fmt.Printf("====================\n\n")
	
	fmt.Printf("Version: %s\n", template.Version)
	fmt.Printf("Author: %s\n", template.Author)
	fmt.Printf("Source: %s\n", template.Source)
	fmt.Printf("Path: %s\n", template.Path)
	fmt.Printf("Description: %s\n", template.Description)
	
	if len(template.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(template.Tags, ", "))
	}
	
	if template.RequireDangerous {
		fmt.Printf("⚠️  Requires --dangerous flag\n")
	}
	
	fmt.Printf("\n📋 Parameters (%d):\n", len(template.Parameters))
	for _, param := range template.Parameters {
		required := ""
		if param.Required {
			required = " (required)"
		}
		
		fmt.Printf("  • %s (%s)%s\n", param.Name, param.Type, required)
		fmt.Printf("    %s\n", param.Description)
		
		if param.Default != nil {
			fmt.Printf("    Default: %v\n", param.Default)
		}
		
		if param.Validation != "" {
			fmt.Printf("    Validation: %s\n", param.Validation)
		}
		
		fmt.Println()
	}

	fmt.Printf("🔄 Steps (%d):\n", len(template.Steps))
	for i, step := range template.Steps {
		fmt.Printf("  %d. %s (%s)\n", i+1, step.Name, step.Operation)
		
		if step.DependsOn != "" {
			fmt.Printf("     Depends on: %s\n", step.DependsOn)
		}
		
		if step.OnError != "" && step.OnError != "fail" {
			fmt.Printf("     On error: %s\n", step.OnError)
		}
		
		fmt.Println()
	}
}

// runTemplateRun handles the template run command
func runTemplateRun(cmd *cobra.Command, args []string) {
	templateName := args[0]
	dangerousFlag, _ := cmd.Flags().GetBool("dangerous")
	
	registry := templates.NewRegistry()
	if err := registry.LoadTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading templates: %v\n", err)
		os.Exit(1)
	}

	template, exists := registry.Get(templateName)
	if !exists {
		fmt.Fprintf(os.Stderr, "Template '%s' not found.\n", templateName)
		fmt.Fprintf(os.Stderr, "Use 'netcrate templates ls' to list available templates.\n")
		os.Exit(1)
	}

	// Parse parameters from command line
	paramFlags, _ := cmd.Flags().GetStringSlice("param")
	parameters := make(map[string]interface{})
	for _, param := range paramFlags {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 2 {
			parameters[parts[0]] = parts[1]
		}
	}
	
	// Set default parameters if not provided
	for _, paramDef := range template.Parameters {
		if _, exists := parameters[paramDef.Name]; !exists && paramDef.Default != nil {
			parameters[paramDef.Name] = paramDef.Default
		}
	}

	// Run compliance check
	checker, err := compliance.NewComplianceChecker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Compliance checker initialization failed: %v\n", err)
		os.Exit(1)
	}

	targets := checker.ParseTargetsFromTemplate(parameters)
	sessionID := fmt.Sprintf("template-%s-%d", templateName, time.Now().Unix())
	command := fmt.Sprintf("netcrate templates run %s", templateName)
	
	complianceResult, err := checker.CheckCompliance(sessionID, templateName, command, targets, dangerousFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Compliance violation: %v\n", err)
		os.Exit(1)
	}
	
	if complianceResult.Status == "blocked" {
		fmt.Fprintf(os.Stderr, "❌ Template execution blocked by compliance rules: %s\n", complianceResult.BlockReason)
		os.Exit(1)
	}

	fmt.Printf("🚀 Running template: %s v%s\n", template.Name, template.Version)
	fmt.Printf("Description: %s\n", template.Description)
	
	// Show compliance info if there are public targets
	if len(complianceResult.PublicTargets) > 0 {
		fmt.Printf("⚠️  Public targets detected: %v\n", complianceResult.PublicTargets)
		fmt.Printf("Risk level: %s\n", complianceResult.RiskLevel)
		for _, warning := range complianceResult.Warnings {
			fmt.Printf("⚠️  %s\n", warning)
		}
		fmt.Printf("\n")
	}
	
	// TODO: Implement parameter collection and validation (C2)
	// TODO: Implement step execution with error handling (C3)
	
	fmt.Printf("⚠️  Template execution not yet implemented.\n")
	fmt.Printf("This will be completed in Step C2 (parameter validation) and C3 (execution).\n")
	fmt.Printf("Compliance check passed ✅\n")
}

// runTemplateIndex handles the template index command
func runTemplateIndex(cmd *cobra.Command, args []string) {
	registry := templates.NewRegistry()
	if err := registry.LoadTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading templates: %v\n", err)
		os.Exit(1)
	}

	registry.PrintIndex()
}