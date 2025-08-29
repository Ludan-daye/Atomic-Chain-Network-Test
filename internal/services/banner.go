package services

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// BannerGrabber performs lightweight banner grabbing for service identification
type BannerGrabber struct {
	timeout     time.Duration
	bufferSize  int
	maxAttempts int
}

// ServiceBanner represents a grabbed service banner
type ServiceBanner struct {
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Protocol    string    `json:"protocol"` // tcp, udp
	Service     string    `json:"service"`  // detected service name
	Banner      string    `json:"banner"`
	Version     string    `json:"version,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Duration    string    `json:"duration"`
	Error       string    `json:"error,omitempty"`
	Confidence  int       `json:"confidence"` // 0-100, confidence in service detection
}

// BannerGrabConfig configures banner grabbing behavior
type BannerGrabConfig struct {
	Timeout     time.Duration
	BufferSize  int
	MaxAttempts int
	Protocols   []string // ["tcp", "udp"]
	Services    []string // specific services to target, empty for all
}

// NewBannerGrabber creates a new banner grabber
func NewBannerGrabber(config BannerGrabConfig) *BannerGrabber {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1024
	}
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	
	return &BannerGrabber{
		timeout:     config.Timeout,
		bufferSize:  config.BufferSize,
		maxAttempts: config.MaxAttempts,
	}
}

// GrabBanner performs banner grabbing on a single host:port
func (bg *BannerGrabber) GrabBanner(host string, port int) *ServiceBanner {
	startTime := time.Now()
	
	banner := &ServiceBanner{
		Host:      host,
		Port:      port,
		Protocol:  "tcp", // Default to TCP
		Timestamp: startTime,
	}
	
	// Try to grab banner
	bannerText, err := bg.grabTCPBanner(host, port)
	duration := time.Since(startTime)
	banner.Duration = duration.String()
	
	if err != nil {
		banner.Error = err.Error()
		return banner
	}
	
	banner.Banner = bannerText
	
	// Analyze banner to detect service and version
	bg.analyzeBanner(banner)
	
	return banner
}

// GrabBanners performs banner grabbing on multiple targets concurrently
func (bg *BannerGrabber) GrabBanners(targets []Target, concurrency int) []*ServiceBanner {
	if concurrency <= 0 {
		concurrency = 10
	}
	
	// Create channels
	targetChan := make(chan Target, len(targets))
	resultChan := make(chan *ServiceBanner, len(targets))
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		go bg.worker(targetChan, resultChan)
	}
	
	// Send targets
	go func() {
		defer close(targetChan)
		for _, target := range targets {
			targetChan <- target
		}
	}()
	
	// Collect results
	var results []*ServiceBanner
	for i := 0; i < len(targets); i++ {
		result := <-resultChan
		results = append(results, result)
	}
	
	return results
}

// Target represents a banner grabbing target
type Target struct {
	Host string
	Port int
}

// worker processes banner grabbing tasks
func (bg *BannerGrabber) worker(targets <-chan Target, results chan<- *ServiceBanner) {
	for target := range targets {
		result := bg.GrabBanner(target.Host, target.Port)
		results <- result
	}
}

// grabTCPBanner performs the actual TCP banner grabbing
func (bg *BannerGrabber) grabTCPBanner(host string, port int) (string, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	
	// Connect with timeout
	conn, err := net.DialTimeout("tcp", address, bg.timeout)
	if err != nil {
		return "", fmt.Errorf("connection failed: %v", err)
	}
	defer conn.Close()
	
	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(bg.timeout))
	
	// Try sending service-specific probes
	banner := bg.probeService(conn, port)
	if banner != "" {
		return strings.TrimSpace(banner), nil
	}
	
	// If no banner received, try reading directly
	buffer := make([]byte, bg.bufferSize)
	n, err := conn.Read(buffer)
	if err != nil && n == 0 {
		return "", fmt.Errorf("no banner received: %v", err)
	}
	
	return strings.TrimSpace(string(buffer[:n])), nil
}

// probeService sends service-specific probes to elicit banners
func (bg *BannerGrabber) probeService(conn net.Conn, port int) string {
	switch port {
	case 21: // FTP
		return bg.readResponse(conn)
	case 22: // SSH
		return bg.readResponse(conn)
	case 23: // Telnet
		return bg.readResponse(conn)
	case 25: // SMTP
		return bg.readResponse(conn)
	case 53: // DNS - TCP
		return bg.probeDNS(conn)
	case 80, 8080, 8000: // HTTP
		return bg.probeHTTP(conn)
	case 110: // POP3
		return bg.readResponse(conn)
	case 143: // IMAP
		return bg.readResponse(conn)
	case 443, 8443: // HTTPS
		return bg.probeHTTPS(conn)
	case 993: // IMAPS
		return bg.readResponse(conn)
	case 995: // POP3S
		return bg.readResponse(conn)
	case 3306: // MySQL
		return bg.readResponse(conn)
	case 5432: // PostgreSQL
		return bg.readResponse(conn)
	case 6379: // Redis
		return bg.probeRedis(conn)
	default:
		// Generic probe - just read
		return bg.readResponse(conn)
	}
}

// readResponse reads a response from connection
func (bg *BannerGrabber) readResponse(conn net.Conn) string {
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

// probeHTTP sends HTTP request and reads response
func (bg *BannerGrabber) probeHTTP(conn net.Conn) string {
	request := "HEAD / HTTP/1.0\r\n\r\n"
	conn.Write([]byte(request))
	
	buffer := make([]byte, bg.bufferSize)
	n, _ := conn.Read(buffer)
	return string(buffer[:n])
}

// probeHTTPS attempts HTTPS connection (simplified)
func (bg *BannerGrabber) probeHTTPS(conn net.Conn) string {
	// For HTTPS, we can't do much without TLS handshake
	// Return connection info
	return fmt.Sprintf("HTTPS service detected on %s", conn.RemoteAddr())
}

// probeDNS sends DNS query
func (bg *BannerGrabber) probeDNS(conn net.Conn) string {
	// Simple DNS query - just check if connection works
	return fmt.Sprintf("DNS service detected on %s", conn.RemoteAddr())
}

// probeRedis sends Redis PING command
func (bg *BannerGrabber) probeRedis(conn net.Conn) string {
	conn.Write([]byte("PING\r\n"))
	buffer := make([]byte, 256)
	n, _ := conn.Read(buffer)
	return string(buffer[:n])
}

// analyzeBanner analyzes banner text to detect service and version
func (bg *BannerGrabber) analyzeBanner(banner *ServiceBanner) {
	bannerText := strings.ToLower(banner.Banner)
	
	// Service detection patterns
	servicePatterns := map[string][]string{
		"ssh": {
			`ssh-`,
			`openssh`,
		},
		"ftp": {
			`ftp`,
			`^220.*ftp`,
			`filezilla`,
			`vsftpd`,
		},
		"http": {
			`http/`,
			`server:`,
			`apache`,
			`nginx`,
			`iis`,
		},
		"smtp": {
			`^220.*smtp`,
			`postfix`,
			`sendmail`,
			`exim`,
		},
		"mysql": {
			`mysql`,
			`mariadb`,
		},
		"postgresql": {
			`postgresql`,
			`postgres`,
		},
		"redis": {
			`redis`,
			`\+pong`,
		},
		"telnet": {
			`telnet`,
			`login:`,
		},
		"pop3": {
			`\+ok.*pop`,
			`pop3`,
		},
		"imap": {
			`\* ok.*imap`,
			`imap4`,
		},
		"dns": {
			`dns`,
			`bind`,
		},
	}
	
	// Version extraction patterns
	versionPatterns := map[string]*regexp.Regexp{
		"ssh":        regexp.MustCompile(`ssh-[\d.]+_([^\s\r\n]+)`),
		"apache":     regexp.MustCompile(`apache[/\s]+([\d.]+)`),
		"nginx":      regexp.MustCompile(`nginx[/\s]+([\d.]+)`),
		"mysql":      regexp.MustCompile(`mysql[^\d]+([\d.]+)`),
		"postgresql": regexp.MustCompile(`postgresql[^\d]+([\d.]+)`),
		"openssh":    regexp.MustCompile(`openssh[_\s]+([\d.]+)`),
	}
	
	maxConfidence := 0
	detectedService := "unknown"
	
	// Check each service pattern
	for service, patterns := range servicePatterns {
		confidence := 0
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			if re.MatchString(bannerText) {
				confidence += 30
				if strings.Contains(bannerText, service) {
					confidence += 20
				}
			}
		}
		
		if confidence > maxConfidence {
			maxConfidence = confidence
			detectedService = service
		}
	}
	
	banner.Service = detectedService
	banner.Confidence = maxConfidence
	
	// Extract version if service detected
	if detectedService != "unknown" {
		if versionRe, exists := versionPatterns[detectedService]; exists {
			if matches := versionRe.FindStringSubmatch(bannerText); len(matches) > 1 {
				banner.Version = matches[1]
				banner.Confidence += 10 // Bonus for version detection
			}
		}
		
		// Try generic version patterns
		if banner.Version == "" {
			genericVersionRe := regexp.MustCompile(`[\d]+\.[\d]+(?:\.[\d]+)?`)
			if match := genericVersionRe.FindString(bannerText); match != "" {
				banner.Version = match
				banner.Confidence += 5
			}
		}
	}
	
	// Cap confidence at 100
	if banner.Confidence > 100 {
		banner.Confidence = 100
	}
}

// FilterBanners filters banners based on criteria
func (bg *BannerGrabber) FilterBanners(banners []*ServiceBanner, criteria BannerFilterCriteria) []*ServiceBanner {
	var filtered []*ServiceBanner
	
	for _, banner := range banners {
		if bg.matchesFilter(banner, criteria) {
			filtered = append(filtered, banner)
		}
	}
	
	return filtered
}

// BannerFilterCriteria defines filtering options
type BannerFilterCriteria struct {
	Services          []string // Filter by service names
	MinConfidence     int      // Minimum confidence level
	HasBanner         bool     // Only include results with banners
	HasVersion        bool     // Only include results with version info
	ExcludeErrors     bool     // Exclude results with errors
	PortRanges        []PortRange
}

// PortRange defines a port range filter
type PortRange struct {
	Start int
	End   int
}

// matchesFilter checks if banner matches filter criteria
func (bg *BannerGrabber) matchesFilter(banner *ServiceBanner, criteria BannerFilterCriteria) bool {
	// Service filter
	if len(criteria.Services) > 0 {
		found := false
		for _, service := range criteria.Services {
			if strings.EqualFold(banner.Service, service) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Confidence filter
	if banner.Confidence < criteria.MinConfidence {
		return false
	}
	
	// Banner presence filter
	if criteria.HasBanner && banner.Banner == "" {
		return false
	}
	
	// Version presence filter
	if criteria.HasVersion && banner.Version == "" {
		return false
	}
	
	// Error filter
	if criteria.ExcludeErrors && banner.Error != "" {
		return false
	}
	
	// Port range filter
	if len(criteria.PortRanges) > 0 {
		found := false
		for _, portRange := range criteria.PortRanges {
			if banner.Port >= portRange.Start && banner.Port <= portRange.End {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

// GetServiceSummary returns summary statistics of banner grabbing results
func (bg *BannerGrabber) GetServiceSummary(banners []*ServiceBanner) map[string]interface{} {
	serviceCounts := make(map[string]int)
	portCounts := make(map[int]int)
	totalBanners := len(banners)
	successfulBanners := 0
	errorCount := 0
	
	for _, banner := range banners {
		serviceCounts[banner.Service]++
		portCounts[banner.Port]++
		
		if banner.Banner != "" {
			successfulBanners++
		}
		if banner.Error != "" {
			errorCount++
		}
	}
	
	successRate := 0.0
	if totalBanners > 0 {
		successRate = float64(successfulBanners) / float64(totalBanners) * 100
	}
	
	return map[string]interface{}{
		"total_banners":     totalBanners,
		"successful_banners": successfulBanners,
		"error_count":       errorCount,
		"success_rate":      successRate,
		"service_counts":    serviceCounts,
		"port_counts":       portCounts,
		"unique_services":   len(serviceCounts),
		"unique_ports":      len(portCounts),
	}
}