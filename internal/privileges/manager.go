package privileges

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// PrivilegeLevel represents the current privilege level
type PrivilegeLevel int

const (
	PrivilegeLevelFull      PrivilegeLevel = iota // Raw sockets, full capabilities
	PrivilegeLevelDegraded                        // Limited capabilities, fallback methods
	PrivilegeLevelRestricted                      // Minimal capabilities
)

func (p PrivilegeLevel) String() string {
	switch p {
	case PrivilegeLevelFull:
		return "full"
	case PrivilegeLevelDegraded:
		return "degraded"
	case PrivilegeLevelRestricted:
		return "restricted"
	default:
		return "unknown"
	}
}

// PrivilegeManager manages privilege detection and fallback mechanisms
type PrivilegeManager struct {
	level              PrivilegeLevel
	capabilities       map[string]bool
	fallbackReasons    []string
	detectionTime      time.Time
	isRoot             bool
	hasRawSocket       bool
	hasICMPSocket      bool
	canPing            bool
	canCreateRawSocket bool
}

// Capability constants
const (
	CapabilityRawSocket    = "raw_socket"
	CapabilityICMP         = "icmp"
	CapabilitySYN          = "syn_scan"
	CapabilitySystemPing   = "system_ping"
	CapabilityTCPConnect   = "tcp_connect"
	CapabilityUDP          = "udp"
)

// NewPrivilegeManager creates and initializes a privilege manager
func NewPrivilegeManager() *PrivilegeManager {
	pm := &PrivilegeManager{
		capabilities:    make(map[string]bool),
		fallbackReasons: make([]string, 0),
		detectionTime:   time.Now(),
	}
	
	pm.detectPrivileges()
	pm.determineLevel()
	
	return pm
}

// detectPrivileges performs comprehensive privilege detection
func (pm *PrivilegeManager) detectPrivileges() {
	// Check if running as root/admin
	pm.isRoot = pm.checkRootPrivileges()
	
	// Test raw socket creation
	pm.hasRawSocket = pm.testRawSocketCapability()
	pm.capabilities[CapabilityRawSocket] = pm.hasRawSocket
	
	// Test ICMP socket creation
	pm.hasICMPSocket = pm.testICMPCapability()
	pm.capabilities[CapabilityICMP] = pm.hasICMPSocket
	
	// Test system ping availability
	pm.canPing = pm.testSystemPing()
	pm.capabilities[CapabilitySystemPing] = pm.canPing
	
	// TCP connect is always available
	pm.capabilities[CapabilityTCPConnect] = true
	
	// UDP is usually available
	pm.capabilities[CapabilityUDP] = pm.testUDPCapability()
	
	// SYN scan requires raw sockets
	pm.capabilities[CapabilitySYN] = pm.hasRawSocket
}

// checkRootPrivileges checks if running with root/administrator privileges
func (pm *PrivilegeManager) checkRootPrivileges() bool {
	switch runtime.GOOS {
	case "windows":
		// On Windows, check if running as administrator
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	case "darwin", "linux":
		return os.Geteuid() == 0
	default:
		return false
	}
}

// testRawSocketCapability tests if raw sockets can be created
func (pm *PrivilegeManager) testRawSocketCapability() bool {
	// Try to create a raw socket
	switch runtime.GOOS {
	case "linux":
		// Try SOCK_RAW with IPPROTO_ICMP
		fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
		if err != nil {
			pm.fallbackReasons = append(pm.fallbackReasons, fmt.Sprintf("raw socket creation failed: %v", err))
			return false
		}
		syscall.Close(fd)
		return true
		
	case "darwin":
		// On macOS, try creating raw socket
		fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
		if err != nil {
			pm.fallbackReasons = append(pm.fallbackReasons, fmt.Sprintf("raw socket creation failed: %v", err))
			return false
		}
		syscall.Close(fd)
		return true
		
	case "windows":
		// Windows raw socket support is limited and complex
		pm.fallbackReasons = append(pm.fallbackReasons, "Windows raw socket support limited")
		return false
		
	default:
		pm.fallbackReasons = append(pm.fallbackReasons, fmt.Sprintf("raw socket support unknown for OS: %s", runtime.GOOS))
		return false
	}
}

// testICMPCapability tests ICMP socket capability
func (pm *PrivilegeManager) testICMPCapability() bool {
	// Try creating an ICMP connection
	conn, err := net.Dial("ip4:icmp", "127.0.0.1")
	if err != nil {
		pm.fallbackReasons = append(pm.fallbackReasons, fmt.Sprintf("ICMP socket failed: %v", err))
		return false
	}
	conn.Close()
	return true
}

// testSystemPing tests if system ping command is available
func (pm *PrivilegeManager) testSystemPing() bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("ping", "-n", "1", "127.0.0.1")
	case "darwin", "linux":
		cmd = exec.Command("ping", "-c", "1", "127.0.0.1")
	default:
		pm.fallbackReasons = append(pm.fallbackReasons, "system ping not available on this OS")
		return false
	}
	
	err := cmd.Run()
	if err != nil {
		pm.fallbackReasons = append(pm.fallbackReasons, fmt.Sprintf("system ping failed: %v", err))
		return false
	}
	return true
}

// testUDPCapability tests UDP socket capability
func (pm *PrivilegeManager) testUDPCapability() bool {
	conn, err := net.Dial("udp", "127.0.0.1:53")
	if err != nil {
		pm.fallbackReasons = append(pm.fallbackReasons, fmt.Sprintf("UDP socket failed: %v", err))
		return false
	}
	conn.Close()
	return true
}

// determineLevel determines the overall privilege level based on capabilities
func (pm *PrivilegeManager) determineLevel() {
	if pm.hasRawSocket && pm.hasICMPSocket {
		pm.level = PrivilegeLevelFull
	} else if pm.canPing || pm.capabilities[CapabilityTCPConnect] {
		pm.level = PrivilegeLevelDegraded
		if !pm.hasRawSocket {
			pm.fallbackReasons = append(pm.fallbackReasons, "SYN scan unavailable - using TCP connect")
		}
		if !pm.hasICMPSocket && pm.canPing {
			pm.fallbackReasons = append(pm.fallbackReasons, "ICMP unavailable - using system ping")
		}
	} else {
		pm.level = PrivilegeLevelRestricted
	}
}

// GetLevel returns the current privilege level
func (pm *PrivilegeManager) GetLevel() PrivilegeLevel {
	return pm.level
}

// HasCapability checks if a specific capability is available
func (pm *PrivilegeManager) HasCapability(capability string) bool {
	return pm.capabilities[capability]
}

// GetFallbackReasons returns the reasons for privilege fallbacks
func (pm *PrivilegeManager) GetFallbackReasons() []string {
	return pm.fallbackReasons
}

// GetAvailableCapabilities returns all available capabilities
func (pm *PrivilegeManager) GetAvailableCapabilities() []string {
	var available []string
	for capability, isAvailable := range pm.capabilities {
		if isAvailable {
			available = append(available, capability)
		}
	}
	return available
}

// GetUnavailableCapabilities returns all unavailable capabilities
func (pm *PrivilegeManager) GetUnavailableCapabilities() []string {
	var unavailable []string
	for capability, isAvailable := range pm.capabilities {
		if !isAvailable {
			unavailable = append(unavailable, capability)
		}
	}
	return unavailable
}

// SuggestPrivilegeElevation provides suggestions for improving privileges
func (pm *PrivilegeManager) SuggestPrivilegeElevation() []string {
	var suggestions []string
	
	if pm.level != PrivilegeLevelFull {
		switch runtime.GOOS {
		case "linux", "darwin":
			suggestions = append(suggestions, "Run with sudo for full capabilities: sudo netcrate ...")
			if !pm.hasRawSocket {
				suggestions = append(suggestions, "Raw socket access requires root privileges")
			}
		case "windows":
			suggestions = append(suggestions, "Run as Administrator for enhanced capabilities")
		}
		
		if pm.level == PrivilegeLevelDegraded {
			suggestions = append(suggestions, "Current mode will use fallback methods (slower but functional)")
		}
	}
	
	return suggestions
}

// PrintPrivilegeStatus prints a comprehensive privilege status report
func (pm *PrivilegeManager) PrintPrivilegeStatus() {
	fmt.Printf("🔒 Privilege Status Report\n")
	fmt.Printf("==========================\n")
	fmt.Printf("Level: %s\n", pm.level.String())
	fmt.Printf("Root/Admin: %v\n", pm.isRoot)
	fmt.Printf("Detection time: %s\n", pm.detectionTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("OS: %s\n", runtime.GOOS)
	
	fmt.Printf("\nCapabilities:\n")
	fmt.Printf("-------------\n")
	for capability, available := range pm.capabilities {
		status := "❌"
		if available {
			status = "✅"
		}
		fmt.Printf("%s %s\n", status, capability)
	}
	
	if len(pm.fallbackReasons) > 0 {
		fmt.Printf("\nFallback Reasons:\n")
		fmt.Printf("-----------------\n")
		for _, reason := range pm.fallbackReasons {
			fmt.Printf("⚠️  %s\n", reason)
		}
	}
	
	suggestions := pm.SuggestPrivilegeElevation()
	if len(suggestions) > 0 {
		fmt.Printf("\nSuggestions:\n")
		fmt.Printf("------------\n")
		for _, suggestion := range suggestions {
			fmt.Printf("💡 %s\n", suggestion)
		}
	}
	
	fmt.Printf("\n")
}

// GetDiscoveryMethodRecommendation recommends discovery methods based on privileges
func (pm *PrivilegeManager) GetDiscoveryMethodRecommendation() map[string]string {
	recommendations := make(map[string]string)
	
	if pm.HasCapability(CapabilityICMP) {
		recommendations["icmp"] = "available - native ICMP"
	} else if pm.HasCapability(CapabilitySystemPing) {
		recommendations["icmp"] = "fallback - system ping command"
	} else {
		recommendations["icmp"] = "unavailable"
	}
	
	if pm.HasCapability(CapabilityTCPConnect) {
		recommendations["tcp"] = "available - TCP connect"
	} else {
		recommendations["tcp"] = "unavailable"
	}
	
	if pm.HasCapability(CapabilityRawSocket) {
		recommendations["arp"] = "available - raw socket ARP"
	} else {
		recommendations["arp"] = "unavailable - requires raw socket"
	}
	
	return recommendations
}

// GetScanMethodRecommendation recommends scan methods based on privileges
func (pm *PrivilegeManager) GetScanMethodRecommendation() map[string]string {
	recommendations := make(map[string]string)
	
	if pm.HasCapability(CapabilitySYN) {
		recommendations["syn"] = "available - SYN scan"
	} else {
		recommendations["syn"] = "unavailable - requires raw socket"
	}
	
	if pm.HasCapability(CapabilityTCPConnect) {
		recommendations["connect"] = "available - TCP connect scan"
	} else {
		recommendations["connect"] = "unavailable"
	}
	
	if pm.HasCapability(CapabilityUDP) {
		recommendations["udp"] = "available - UDP scan"
	} else {
		recommendations["udp"] = "unavailable"
	}
	
	return recommendations
}

// GetPrivilegeSummary returns a summary suitable for inclusion in scan results
func (pm *PrivilegeManager) GetPrivilegeSummary() map[string]interface{} {
	return map[string]interface{}{
		"privilege_mode":     pm.level.String(),
		"is_root":           pm.isRoot,
		"detection_time":    pm.detectionTime,
		"os":               runtime.GOOS,
		"capabilities":     pm.capabilities,
		"fallback_reasons": pm.fallbackReasons,
		"available_methods": pm.GetAvailableCapabilities(),
	}
}

// IsPrivileged returns true if running with elevated privileges
func (pm *PrivilegeManager) IsPrivileged() bool {
	return pm.level == PrivilegeLevelFull
}

// IsDegraded returns true if running in degraded mode
func (pm *PrivilegeManager) IsDegraded() bool {
	return pm.level == PrivilegeLevelDegraded
}

// RequiresElevation returns true if privilege elevation would improve capabilities
func (pm *PrivilegeManager) RequiresElevation() bool {
	return pm.level != PrivilegeLevelFull
}

// GetOptimalMethod returns the best available method for a given operation
func (pm *PrivilegeManager) GetOptimalMethod(operation string) string {
	switch operation {
	case "discovery":
		if pm.HasCapability(CapabilityICMP) {
			return "icmp"
		} else if pm.HasCapability(CapabilitySystemPing) {
			return "ping"
		} else {
			return "tcp"
		}
	case "scan":
		if pm.HasCapability(CapabilitySYN) {
			return "syn"
		} else {
			return "connect"
		}
	default:
		return "auto"
	}
}