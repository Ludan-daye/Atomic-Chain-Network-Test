package quick

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/netcrate/netcrate/internal/netenv"
	"github.com/netcrate/netcrate/internal/ops"
)

// QuickConfig holds configuration for quick mode
type QuickConfig struct {
	Interface    *netenv.NetworkInterface
	TargetCIDR   string
	PortSet      string // "top100", "top1000", "web", "database", "custom"
	Profile      string // "safe", "fast", "custom"
	DiscoverOpts ops.DiscoverOptions
	ScanOpts     ops.ScanOptions
	OutputDir    string
	DryRun       bool
	SkipConfirm  bool
	Interactive  bool   // Enable interactive mode
}

// QuickResult holds the complete results of quick mode execution
type QuickResult struct {
	RunID         string                `json:"run_id"`
	Interface     *netenv.NetworkInterface `json:"interface"`
	TargetCIDR    string                `json:"target_cidr"`
	StartTime     time.Time             `json:"start_time"`
	EndTime       time.Time             `json:"end_time"`
	Duration      float64               `json:"duration"`
	DiscoverResult *ops.DiscoverSummary `json:"discover_result"`
	ScanResult     *ops.ScanSummary     `json:"scan_result"`
	Summary        QuickSummary          `json:"summary"`
}

// QuickSummary provides a high-level overview
type QuickSummary struct {
	HostsDiscovered int               `json:"hosts_discovered"`
	OpenPorts       int               `json:"open_ports"`
	TopServices     map[string]int    `json:"top_services"`
	LiveHosts       []string          `json:"live_hosts"`
	CriticalPorts   []CriticalPort    `json:"critical_ports"`
}

// CriticalPort represents a notable open port
type CriticalPort struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Service string `json:"service"`
	Risk    string `json:"risk"` // "low", "medium", "high"
}

// RunQuickMode executes the complete quick mode workflow
func RunQuickMode(dryRun bool, skipConfirm bool, interactive bool) (*QuickResult, error) {
	startTime := time.Now()
	runID := fmt.Sprintf("quick_%d", startTime.Unix())

	fmt.Println("🚀 NetCrate Quick Mode")
	fmt.Println("======================")

	// Step 1: Auto-detect network interface
	fmt.Println("\n[1/4] 🔍 自动检测网络接口...")
	
	config, err := autoDetectInterface()
	if err != nil {
		return nil, fmt.Errorf("interface detection failed: %w", err)
	}
	
	config.DryRun = dryRun
	config.SkipConfirm = skipConfirm
	config.Interactive = interactive

	// Step 2: Calculate target network
	fmt.Println("\n[2/4] 🎯 计算目标网段...")
	
	err = calculateTargetNetwork(config)
	if err != nil {
		return nil, fmt.Errorf("target calculation failed: %w", err)
	}

	// Step 2.5: Interactive configuration selection
	if interactive && !skipConfirm {
		fmt.Println("\n[2.5/4] ⚙️ 扫描配置")
		err = interactiveConfiguration(config)
		if err != nil {
			return nil, fmt.Errorf("configuration selection failed: %w", err)
		}
	}

	// Step 3: Show configuration and get confirmation
	if !skipConfirm {
		fmt.Println("\n[3/4] ⚙️ 配置确认")
		fmt.Println("==================")
		printConfiguration(config)
		
		if !getUserConfirmation() {
			fmt.Println("\n❌ 用户取消操作")
			return nil, fmt.Errorf("user cancelled")
		}
	}

	// Step 4: Execute scan pipeline
	fmt.Println("\n[4/4] 🔍 执行扫描流水线...")
	
	if dryRun {
		fmt.Println("🧪 [DRY RUN] 跳过实际执行")
		return &QuickResult{
			RunID:      runID,
			Interface:  config.Interface,
			TargetCIDR: config.TargetCIDR,
			StartTime:  startTime,
			EndTime:    time.Now(),
		}, nil
	}

	result, err := executeScanPipeline(config)
	if err != nil {
		return nil, fmt.Errorf("scan pipeline failed: %w", err)
	}

	result.RunID = runID
	result.Interface = config.Interface
	result.TargetCIDR = config.TargetCIDR
	result.StartTime = startTime
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime).Seconds()

	// Save results
	err = saveResults(result)
	if err != nil {
		fmt.Printf("⚠️ 结果保存失败: %v\n", err)
	}

	return result, nil
}

// autoDetectInterface automatically selects the best network interface
func autoDetectInterface() (*QuickConfig, error) {
	// Get network environment
	netEnv, err := netenv.DetectNetworkEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to detect network environment: %w", err)
	}

	if len(netEnv.Interfaces) == 0 {
		return nil, fmt.Errorf("no network interfaces found")
	}

	// Find the best private network interface
	var selectedInterface *netenv.NetworkInterface
	
	// Priority: private networks first, then any active interface
	for _, iface := range netEnv.Interfaces {
		if iface.Status != "up" {
			continue
		}
		
		for _, addr := range iface.Addresses {
			ip := net.ParseIP(addr.IP)
			if ip != nil && isPrivateIP(ip) {
				selectedInterface = &iface
				break
			}
		}
		
		if selectedInterface != nil {
			break
		}
	}

	// If no private interface, use the recommended one
	if selectedInterface == nil {
		for _, iface := range netEnv.Interfaces {
			if iface.Name == netEnv.Recommended && iface.Status == "up" {
				selectedInterface = &iface
				break
			}
		}
	}

	// If still no interface, use the first active one
	if selectedInterface == nil {
		for _, iface := range netEnv.Interfaces {
			if iface.Status == "up" {
				selectedInterface = &iface
				break
			}
		}
	}

	if selectedInterface == nil {
		return nil, fmt.Errorf("未检测到可用的网络接口")
	}

	fmt.Printf("✅ 自动选择接口: %s (%s)\n", 
		selectedInterface.Name, selectedInterface.DisplayName)
	
	if len(selectedInterface.Addresses) > 0 {
		fmt.Printf("   IP地址: %s\n", selectedInterface.Addresses[0].IP)
	}

	return &QuickConfig{
		Interface: selectedInterface,
	}, nil
}

// calculateTargetNetwork derives the target CIDR from interface information
func calculateTargetNetwork(config *QuickConfig) error {
	if len(config.Interface.Addresses) == 0 {
		return fmt.Errorf("selected interface has no IP addresses")
	}

	addr := config.Interface.Addresses[0]
	
	// Parse the network CIDR
	if !strings.Contains(addr.Network, "/") {
		return fmt.Errorf("invalid network format: %s", addr.Network)
	}

	// Extract network address
	_, ipnet, err := net.ParseCIDR(addr.Network)
	if err != nil {
		return fmt.Errorf("failed to parse network CIDR: %w", err)
	}

	targetCIDR := ipnet.String()
	
	// Safety check: ensure it's a private network
	if !isPrivateNetwork(ipnet) {
		return fmt.Errorf("⚠️ 检测到公网地址 %s\n"+
			"为了安全，Quick模式只能扫描私网地址\n"+
			"如需扫描公网，请使用: netcrate ops discover --dangerous", 
			targetCIDR)
	}

	config.TargetCIDR = targetCIDR
	
	fmt.Printf("✅ 目标网段: %s\n", targetCIDR)
	
	// Set default configuration
	config.PortSet = "top100"  // Default port set
	config.Profile = "safe"    // Default profile
	
	err = applyConfiguration(config)
	if err != nil {
		return fmt.Errorf("failed to apply configuration: %w", err)
	}

	return nil
}

// printConfiguration displays the configuration for user confirmation
func printConfiguration(config *QuickConfig) {
	fmt.Printf("📡 接口: %s (%s)\n", config.Interface.Name, config.Interface.DisplayName)
	if len(config.Interface.Addresses) > 0 {
		fmt.Printf("📍 本机IP: %s\n", config.Interface.Addresses[0].IP)
	}
	fmt.Printf("🎯 目标网段: %s\n", config.TargetCIDR)
	fmt.Printf("🔍 主机发现: ICMP + TCP (22,80,443)\n")
	
	// Display port set information
	portCount := len(config.ScanOpts.Ports)
	portSetDesc := getPortSetDescription(config.PortSet, portCount)
	fmt.Printf("📊 端口扫描: %s\n", portSetDesc)
	
	// Display speed profile information  
	profileDesc := getProfileDescription(config.Profile, config.DiscoverOpts.Rate, config.DiscoverOpts.Concurrency)
	fmt.Printf("⚡ 速率档位: %s\n", profileDesc)
}

// getPortSetDescription returns a human-readable description of the port set
func getPortSetDescription(portSet string, portCount int) string {
	switch portSet {
	case "top100":
		return fmt.Sprintf("top100 (%d 个最常用端口)", portCount)
	case "top1000":
		return fmt.Sprintf("top1000 (%d 个最常用端口)", portCount)
	case "web":
		return fmt.Sprintf("web (%d 个Web服务端口)", portCount)
	case "database":
		return fmt.Sprintf("database (%d 个数据库端口)", portCount)
	case "common":
		return fmt.Sprintf("common (%d 个通用服务端口)", portCount)
	default:
		return fmt.Sprintf("%s (%d 个端口)", portSet, portCount)
	}
}

// getProfileDescription returns a human-readable description of the speed profile
func getProfileDescription(profile string, rate, concurrency int) string {
	switch {
	case profile == "safe":
		return fmt.Sprintf("safe - 安全模式 (%d pps, %d 并发)", rate, concurrency)
	case profile == "fast":
		return fmt.Sprintf("fast - 快速模式 (%d pps, %d 并发)", rate, concurrency)
	case strings.HasPrefix(profile, "custom-"):
		return fmt.Sprintf("custom - 自定义 (%d pps, %d 并发)", rate, concurrency)
	default:
		return fmt.Sprintf("%s (%d pps, %d 并发)", profile, rate, concurrency)
	}
}

// getUserConfirmation prompts user for confirmation
func getUserConfirmation() bool {
	fmt.Printf("\n按 Enter 继续，输入 'q' 退出: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if input == "q" || input == "quit" || input == "exit" {
			return false
		}
	}
	
	return true
}

// executeScanPipeline runs the discovery and scanning operations
func executeScanPipeline(config *QuickConfig) (*QuickResult, error) {
	result := &QuickResult{}

	// Phase 1: Host Discovery
	fmt.Println("\n🔍 阶段 1: 主机发现")
	fmt.Println("==================")
	
	discoverResult, err := ops.Discover(config.DiscoverOpts)
	if err != nil {
		return nil, fmt.Errorf("host discovery failed: %w", err)
	}
	
	result.DiscoverResult = discoverResult
	
	fmt.Printf("✅ 发现 %d 个活跃主机 (耗时 %.1fs)\n", 
		discoverResult.HostsDiscovered, discoverResult.Duration)

	// Extract live hosts for port scanning
	var liveHosts []string
	for _, hostResult := range discoverResult.Results {
		if hostResult.Status == "up" {
			liveHosts = append(liveHosts, hostResult.Host)
		}
	}

	if len(liveHosts) == 0 {
		fmt.Println("⚠️ 未发现活跃主机，跳过端口扫描")
		result.Summary = QuickSummary{
			HostsDiscovered: 0,
			LiveHosts:       liveHosts,
		}
		return result, nil
	}

	// Phase 2: Port Scanning
	fmt.Println("\n🔍 阶段 2: 端口扫描")
	fmt.Println("==================")
	
	config.ScanOpts.Targets = liveHosts
	
	scanResult, err := ops.ScanPorts(config.ScanOpts)
	if err != nil {
		return nil, fmt.Errorf("port scanning failed: %w", err)
	}
	
	result.ScanResult = scanResult
	
	fmt.Printf("✅ 扫描完成：发现 %d 个开放端口 (耗时 %.1fs)\n", 
		scanResult.OpenPorts, scanResult.Duration)

	// Generate summary
	result.Summary = generateSummary(discoverResult, scanResult)
	
	return result, nil
}

// generateSummary creates a high-level summary of results
func generateSummary(discoverResult *ops.DiscoverSummary, scanResult *ops.ScanSummary) QuickSummary {
	summary := QuickSummary{
		HostsDiscovered: discoverResult.HostsDiscovered,
		OpenPorts:       scanResult.OpenPorts,
		TopServices:     make(map[string]int),
		LiveHosts:       make([]string, 0),
		CriticalPorts:   make([]CriticalPort, 0),
	}

	// Extract live hosts
	for _, hostResult := range discoverResult.Results {
		if hostResult.Status == "up" {
			summary.LiveHosts = append(summary.LiveHosts, hostResult.Host)
		}
	}

	// Analyze port scan results
	for _, portResult := range scanResult.Results {
		if portResult.Status == "open" {
			service := "unknown"
			if portResult.Service != nil {
				service = portResult.Service.Name
			}
			
			// Count services
			summary.TopServices[service]++
			
			// Identify critical ports
			risk := assessPortRisk(portResult.Port, service)
			if risk != "low" {
				summary.CriticalPorts = append(summary.CriticalPorts, CriticalPort{
					Host:    portResult.Host,
					Port:    portResult.Port,
					Service: service,
					Risk:    risk,
				})
			}
		}
	}

	return summary
}

// assessPortRisk evaluates the security risk level of an open port
func assessPortRisk(port int, service string) string {
	// High risk ports
	highRiskPorts := map[int]bool{
		21: true,  // FTP
		22: true,  // SSH (if exposed publicly)
		23: true,  // Telnet
		135: true, // RPC
		139: true, // NetBIOS
		445: true, // SMB
		3389: true, // RDP
	}

	// Medium risk ports
	mediumRiskPorts := map[int]bool{
		80:   true, // HTTP
		443:  true, // HTTPS
		3306: true, // MySQL
		5432: true, // PostgreSQL
		27017: true, // MongoDB
	}

	if highRiskPorts[port] {
		return "high"
	}
	if mediumRiskPorts[port] {
		return "medium"
	}
	return "low"
}

// saveResults saves the results to ~/.netcrate/runs/
func saveResults(result *QuickResult) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	runsDir := filepath.Join(homeDir, ".netcrate", "runs")
	err = os.MkdirAll(runsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create runs directory: %w", err)
	}

	runDir := filepath.Join(runsDir, result.RunID)
	err = os.MkdirAll(runDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}

	// Save main result as JSON
	resultFile := filepath.Join(runDir, "result.json")
	file, err := os.Create(resultFile)
	if err != nil {
		return fmt.Errorf("failed to create result file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(result)
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Printf("✅ 结果已保存到: %s\n", runDir)
	return nil
}

// Helper functions

func isPrivateIP(ip net.IP) bool {
	// RFC 1918 private networks
	private := []net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
	}

	for _, cidr := range private {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func isPrivateNetwork(ipnet *net.IPNet) bool {
	return isPrivateIP(ipnet.IP)
}

// interactiveConfiguration prompts user for scanning configuration
func interactiveConfiguration(config *QuickConfig) error {
	fmt.Println("================")
	
	// Port set selection
	err := selectPortSet(config)
	if err != nil {
		return err
	}
	
	// Speed profile selection  
	err = selectSpeedProfile(config)
	if err != nil {
		return err
	}
	
	// Apply the selected configuration
	return applyConfiguration(config)
}

// selectPortSet prompts user to select a port set
func selectPortSet(config *QuickConfig) error {
	fmt.Println("\n📊 选择端口集:")
	fmt.Println("  1. top100    - 最常用100个端口 (默认)")
	fmt.Println("  2. top1000   - 最常用1000个端口")
	fmt.Println("  3. web       - Web服务端口")
	fmt.Println("  4. database  - 数据库端口")
	fmt.Println("  5. common    - 通用服务端口")
	
	fmt.Printf("请选择 (1-5) [默认: 1]: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		
		switch choice {
		case "", "1":
			config.PortSet = "top100"
		case "2":
			config.PortSet = "top1000"
		case "3":
			config.PortSet = "web"
		case "4":
			config.PortSet = "database"
		case "5":
			config.PortSet = "common"
		default:
			fmt.Printf("无效选择，使用默认值 (top100)\n")
			config.PortSet = "top100"
		}
	}
	
	fmt.Printf("✅ 端口集: %s\n", config.PortSet)
	return nil
}

// selectSpeedProfile prompts user to select a speed profile
func selectSpeedProfile(config *QuickConfig) error {
	fmt.Println("\n⚡ 选择速率档位:")
	fmt.Println("  1. safe   - 安全模式 (100pps, 200并发) [默认]")
	fmt.Println("  2. fast   - 快速模式 (400pps, 800并发)")
	fmt.Println("  3. custom - 自定义参数")
	
	fmt.Printf("请选择 (1-3) [默认: 1]: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		
		switch choice {
		case "", "1":
			config.Profile = "safe"
		case "2":
			config.Profile = "fast"
		case "3":
			config.Profile = "custom"
			return selectCustomProfile(config)
		default:
			fmt.Printf("无效选择，使用默认值 (safe)\n")
			config.Profile = "safe"
		}
	}
	
	fmt.Printf("✅ 速率档位: %s\n", config.Profile)
	return nil
}

// selectCustomProfile prompts for custom rate settings
func selectCustomProfile(config *QuickConfig) error {
	fmt.Println("\n🔧 自定义速率参数:")
	
	// Get custom rate
	fmt.Printf("扫描速率 (pps) [默认: 100]: ")
	scanner := bufio.NewScanner(os.Stdin)
	rate := 100
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			if r, err := fmt.Sscanf(input, "%d", &rate); err != nil || r != 1 {
				fmt.Printf("无效输入，使用默认值 100\n")
				rate = 100
			}
		}
	}
	
	// Get custom concurrency
	fmt.Printf("并发数 [默认: 200]: ")
	concurrency := 200
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			if r, err := fmt.Sscanf(input, "%d", &concurrency); err != nil || r != 1 {
				fmt.Printf("无效输入，使用默认值 200\n")
				concurrency = 200
			}
		}
	}
	
	// Store custom values in a special profile format
	config.Profile = fmt.Sprintf("custom-%d-%d", rate, concurrency)
	
	fmt.Printf("✅ 自定义档位: %dpps, %d并发\n", rate, concurrency)
	return nil
}

// applyConfiguration applies the selected configuration to discover and scan options
func applyConfiguration(config *QuickConfig) error {
	// Parse port set
	portSet := config.PortSet
	if portSet == "" {
		portSet = "top100"
	}
	
	ports, err := ops.ParsePortSpec(portSet)
	if err != nil {
		return fmt.Errorf("invalid port set %s: %w", portSet, err)
	}
	
	// Parse speed profile
	rate, concurrency := parseSpeedProfile(config.Profile)
	
	// Configure discovery options
	config.DiscoverOpts = ops.DiscoverOptions{
		Targets:     []string{config.TargetCIDR},
		Methods:     []string{"icmp", "tcp"},
		Rate:        rate,
		Concurrency: concurrency,
		TCPPorts:    []int{22, 80, 443},
	}

	// Configure scan options
	config.ScanOpts = ops.ScanOptions{
		Targets:          []string{}, // Will be filled with discovered hosts
		Ports:            ports,
		ServiceDetection: true,
		Rate:             rate,
		Concurrency:      concurrency,
	}
	
	return nil
}

// parseSpeedProfile parses a speed profile and returns rate and concurrency values
func parseSpeedProfile(profile string) (int, int) {
	switch {
	case profile == "safe":
		return 100, 200
	case profile == "fast":
		return 400, 800
	case strings.HasPrefix(profile, "custom-"):
		// Parse custom-rate-concurrency format
		parts := strings.Split(profile, "-")
		if len(parts) == 3 {
			var rate, concurrency int
			if n, err := fmt.Sscanf(parts[1]+"-"+parts[2], "%d-%d", &rate, &concurrency); err == nil && n == 2 {
				return rate, concurrency
			}
		}
		fallthrough // If parsing fails, use safe defaults
	default:
		return 100, 200
	}
}

// PrintQuickSummary displays a formatted summary of results
func PrintQuickSummary(result *QuickResult) {
	fmt.Println("\n🎉 扫描完成！")
	fmt.Println("==============")
	
	fmt.Printf("运行ID: %s\n", result.RunID)
	fmt.Printf("目标网段: %s\n", result.TargetCIDR)
	fmt.Printf("总耗时: %.1f 秒\n", result.Duration)
	
	fmt.Println("\n📊 扫描结果")
	fmt.Println("============")
	fmt.Printf("活跃主机: %d\n", result.Summary.HostsDiscovered)
	fmt.Printf("开放端口: %d\n", result.Summary.OpenPorts)
	
	if len(result.Summary.LiveHosts) > 0 {
		fmt.Println("\n🟢 活跃主机列表:")
		for _, host := range result.Summary.LiveHosts {
			fmt.Printf("  • %s\n", host)
		}
	}
	
	if len(result.Summary.TopServices) > 0 {
		fmt.Println("\n🔧 发现的服务:")
		for service, count := range result.Summary.TopServices {
			fmt.Printf("  • %s: %d 个实例\n", service, count)
		}
	}
	
	if len(result.Summary.CriticalPorts) > 0 {
		fmt.Println("\n⚠️ 关键端口 (需要注意):")
		for _, cp := range result.Summary.CriticalPorts {
			fmt.Printf("  • %s:%d (%s) - %s 风险\n", cp.Host, cp.Port, cp.Service, cp.Risk)
		}
	}
	
	fmt.Printf("\n💾 详细结果: netcrate output show --run %s\n", result.RunID)
}