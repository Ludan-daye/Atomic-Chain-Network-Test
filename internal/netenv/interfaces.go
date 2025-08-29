// Package netenv provides network environment detection and configuration
package netenv

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	MacAddress   string    `json:"mac_address"`
	MTU          int       `json:"mtu"`
	Status       string    `json:"status"`
	Type         string    `json:"type"`
	Addresses    []Address `json:"addresses"`
	Gateway      *Gateway  `json:"gateway,omitempty"`
}

// Address represents an IP address configuration
type Address struct {
	IP        string `json:"ip"`
	Network   string `json:"network"`
	Netmask   string `json:"netmask"`
	Broadcast string `json:"broadcast,omitempty"`
	Scope     string `json:"scope"`
}

// Gateway represents gateway information
type Gateway struct {
	IP         string  `json:"ip"`
	MacAddress string  `json:"mac_address,omitempty"`
	RTT        float64 `json:"rtt,omitempty"`
}

// DetectResult represents the complete network environment detection result
type DetectResult struct {
	Interfaces    []NetworkInterface `json:"interfaces"`
	Recommended   string            `json:"recommended"`
	SystemInfo    SystemInfo        `json:"system_info"`
	Capabilities  Capabilities      `json:"capabilities"`
}

// SystemInfo represents system network information
type SystemInfo struct {
	Platform     string   `json:"platform"`
	Hostname     string   `json:"hostname"`
	DNSServers   []string `json:"dns_servers"`
	DefaultRoute string   `json:"default_route"`
}

// Capabilities represents network capabilities
type Capabilities struct {
	RawSocket        bool `json:"raw_socket"`
	PromiscuousMode  bool `json:"promiscuous_mode"`
	PacketCapture    bool `json:"packet_capture"`
}

// GetActiveInterfaces returns all active network interfaces
func GetActiveInterfaces() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var result []NetworkInterface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // Skip inactive interfaces
		}

		netIface := NetworkInterface{
			Name:        iface.Name,
			DisplayName: getDisplayName(iface.Name),
			MacAddress:  iface.HardwareAddr.String(),
			MTU:         iface.MTU,
			Status:      "up",
			Type:        getInterfaceType(iface.Name),
		}

		// Get IP addresses for this interface
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipnet.IP.IsLoopback() {
				netIface.Type = "loopback"
				continue
			}

			// Skip IPv6 for now (focus on IPv4)
			if ipnet.IP.To4() == nil {
				continue
			}

			address := Address{
				IP:      ipnet.IP.String(),
				Network: ipnet.String(),
				Netmask: net.IP(ipnet.Mask).String(),
				Scope:   getAddressScope(ipnet.IP),
			}

			// Calculate broadcast address for IPv4
			if ipnet.IP.To4() != nil {
				broadcast := calculateBroadcast(ipnet)
				if broadcast != nil {
					address.Broadcast = broadcast.String()
				}
			}

			netIface.Addresses = append(netIface.Addresses, address)
		}

		// Only include interfaces with at least one IPv4 address
		if len(netIface.Addresses) > 0 {
			// Try to detect gateway
			gateway := detectGateway(netIface.Name)
			if gateway != nil {
				netIface.Gateway = gateway
			}

			result = append(result, netIface)
		}
	}

	return result, nil
}

// DetectNetworkEnvironment performs comprehensive network environment detection
func DetectNetworkEnvironment() (*DetectResult, error) {
	interfaces, err := GetActiveInterfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to detect interfaces: %w", err)
	}

	// Get system information
	hostname, _ := exec.Command("hostname").Output()
	
	systemInfo := SystemInfo{
		Platform:     runtime.GOOS,
		Hostname:     strings.TrimSpace(string(hostname)),
		DNSServers:   detectDNSServers(),
		DefaultRoute: detectDefaultRoute(),
	}

	// Detect capabilities
	capabilities := Capabilities{
		RawSocket:       checkRawSocketCapability(),
		PromiscuousMode: checkPromiscuousCapability(),
		PacketCapture:   checkPacketCaptureCapability(),
	}

	// Find recommended interface
	recommended := findRecommendedInterface(interfaces)

	return &DetectResult{
		Interfaces:   interfaces,
		Recommended:  recommended,
		SystemInfo:   systemInfo,
		Capabilities: capabilities,
	}, nil
}

// InferNetworkRange infers the network range for an interface
func InferNetworkRange(iface NetworkInterface) (*net.IPNet, error) {
	if len(iface.Addresses) == 0 {
		return nil, fmt.Errorf("interface %s has no addresses", iface.Name)
	}

	// Use the first non-loopback address
	for _, addr := range iface.Addresses {
		if addr.Scope != "host" {
			_, ipnet, err := net.ParseCIDR(addr.Network)
			if err != nil {
				continue
			}
			return ipnet, nil
		}
	}

	return nil, fmt.Errorf("no suitable address found for interface %s", iface.Name)
}

// IsPrivateNetwork checks if an IP address is in a private network range
func IsPrivateNetwork(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Convert to IPv4 if possible
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}

	// RFC 1918 private networks
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8", // Loopback
	}

	for _, rangeStr := range privateRanges {
		_, privateNet, err := net.ParseCIDR(rangeStr)
		if err != nil {
			continue
		}
		if privateNet.Contains(ip) {
			return true
		}
	}

	return false
}

// Helper functions

func getDisplayName(name string) string {
	// On macOS, provide more friendly names
	switch runtime.GOOS {
	case "darwin":
		switch {
		case strings.HasPrefix(name, "en"):
			return name + " (Ethernet/WiFi)"
		case strings.HasPrefix(name, "lo"):
			return name + " (Loopback)"
		case strings.HasPrefix(name, "utun"):
			return name + " (VPN)"
		}
	}
	return name
}

func getInterfaceType(name string) string {
	switch {
	case strings.HasPrefix(name, "lo"):
		return "loopback"
	case strings.HasPrefix(name, "en"):
		return "ethernet"
	case strings.HasPrefix(name, "wl"):
		return "wireless"
	case strings.Contains(name, "tun") || strings.Contains(name, "tap"):
		return "vpn"
	default:
		return "unknown"
	}
}

func getAddressScope(ip net.IP) string {
	if ip.IsLoopback() {
		return "host"
	}
	if ip.IsLinkLocalUnicast() {
		return "link"
	}
	return "global"
}

func calculateBroadcast(ipnet *net.IPNet) net.IP {
	if ipnet.IP.To4() == nil {
		return nil // Only for IPv4
	}

	ip := ipnet.IP.To4()
	mask := ipnet.Mask

	broadcast := make(net.IP, len(ip))
	for i := range ip {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return broadcast
}

func detectGateway(interfaceName string) *Gateway {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("route", "-n", "get", "default")
	case "linux":
		cmd = exec.Command("ip", "route", "show", "default")
	default:
		return nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse output to extract gateway IP
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		switch runtime.GOOS {
		case "darwin":
			if strings.HasPrefix(line, "gateway:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					gatewayIP := parts[1]
					if net.ParseIP(gatewayIP) != nil {
						return &Gateway{IP: gatewayIP}
					}
				}
			}
		case "linux":
			if strings.Contains(line, "default via") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					gatewayIP := parts[2]
					if net.ParseIP(gatewayIP) != nil {
						return &Gateway{IP: gatewayIP}
					}
				}
			}
		}
	}

	return nil
}

func detectDNSServers() []string {
	var servers []string
	
	// Try to read from /etc/resolv.conf (Unix-like systems)
	if content, err := exec.Command("cat", "/etc/resolv.conf").Output(); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "nameserver") {
				parts := strings.Fields(line)
				if len(parts) >= 2 && net.ParseIP(parts[1]) != nil {
					servers = append(servers, parts[1])
				}
			}
		}
	}
	
	return servers
}

func detectDefaultRoute() string {
	interfaces, err := GetActiveInterfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Gateway != nil && iface.Type != "loopback" {
			return iface.Name
		}
	}

	return ""
}

func findRecommendedInterface(interfaces []NetworkInterface) string {
	var candidates []NetworkInterface

	// Filter out loopback interfaces
	for _, iface := range interfaces {
		if iface.Type != "loopback" && len(iface.Addresses) > 0 {
			candidates = append(candidates, iface)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Prefer interfaces with gateways and private IP addresses
	for _, iface := range candidates {
		if iface.Gateway != nil {
			for _, addr := range iface.Addresses {
				if ip := net.ParseIP(addr.IP); ip != nil && IsPrivateNetwork(ip) {
					return iface.Name
				}
			}
		}
	}

	// Fallback to first active non-loopback interface
	return candidates[0].Name
}

func checkRawSocketCapability() bool {
	// Try to create a raw socket (this will fail without privileges)
	conn, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func checkPromiscuousCapability() bool {
	// This is a simplified check - actual implementation would be more complex
	return checkRawSocketCapability()
}

func checkPacketCaptureCapability() bool {
	// Check if we can open packet capture (simplified)
	return checkRawSocketCapability()
}

// PingGateway attempts to ping the gateway to measure RTT
func PingGateway(gateway *Gateway) error {
	if gateway == nil || gateway.IP == "" {
		return fmt.Errorf("no gateway specified")
	}

	start := time.Now()
	
	// Use system ping command
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("ping", "-c", "1", "-W", "1000", gateway.IP)
	default:
		return fmt.Errorf("ping not supported on %s", runtime.GOOS)
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	gateway.RTT = float64(time.Since(start)) / float64(time.Millisecond)
	return nil
}