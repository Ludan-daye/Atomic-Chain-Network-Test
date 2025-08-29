package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ProtocolFingerprinter performs advanced protocol identification
type ProtocolFingerprinter struct {
	timeout         time.Duration
	maxProbeAttempts int
	userAgent       string
}

// ProtocolFingerprint represents detailed protocol information
type ProtocolFingerprint struct {
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Protocol    string            `json:"protocol"`    // tcp, udp, tls
	Service     string            `json:"service"`     // http, https, ssh, ftp, etc.
	Application string            `json:"application"` // nginx, apache, openssh, etc.
	Version     string            `json:"version"`
	TLS         *TLSInfo          `json:"tls,omitempty"`
	HTTP        *HTTPInfo         `json:"http,omitempty"`
	SSH         *SSHInfo          `json:"ssh,omitempty"`
	MySQL       *MySQLInfo        `json:"mysql,omitempty"`
	Confidence  int               `json:"confidence"`
	Timestamp   time.Time         `json:"timestamp"`
	Duration    string            `json:"duration"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// TLSInfo contains TLS-specific information
type TLSInfo struct {
	Version     string   `json:"version"`      // TLS 1.2, 1.3, etc.
	CipherSuite string   `json:"cipher_suite"`
	Certificate *CertInfo `json:"certificate,omitempty"`
}

// CertInfo contains certificate information
type CertInfo struct {
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	CommonName  string    `json:"common_name"`
	SANs        []string  `json:"sans"` // Subject Alternative Names
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	Fingerprint string    `json:"fingerprint"`
}

// HTTPInfo contains HTTP-specific information
type HTTPInfo struct {
	Status      string            `json:"status"`
	Server      string            `json:"server"`
	Headers     map[string]string `json:"headers"`
	Title       string            `json:"title,omitempty"`
	Technologies []string         `json:"technologies,omitempty"`
	RedirectURL string            `json:"redirect_url,omitempty"`
}

// SSHInfo contains SSH-specific information
type SSHInfo struct {
	Version        string   `json:"version"`         // SSH-2.0
	Implementation string   `json:"implementation"`  // OpenSSH_8.0
	Algorithms     []string `json:"algorithms,omitempty"`
	HostKey        string   `json:"host_key,omitempty"`
}

// MySQLInfo contains MySQL-specific information
type MySQLInfo struct {
	Version       string `json:"version"`
	ServerVersion string `json:"server_version"`
	Protocol      int    `json:"protocol"`
	AuthMethod    string `json:"auth_method"`
}

// FingerprintConfig configures protocol fingerprinting
type FingerprintConfig struct {
	Timeout         time.Duration
	MaxProbeAttempts int
	UserAgent       string
	EnableTLS       bool
	EnableHTTP      bool
	EnableSSH       bool
	EnableMySQL     bool
}

// NewProtocolFingerprinter creates a new protocol fingerprinter
func NewProtocolFingerprinter(config FingerprintConfig) *ProtocolFingerprinter {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxProbeAttempts == 0 {
		config.MaxProbeAttempts = 3
	}
	if config.UserAgent == "" {
		config.UserAgent = "NetCrate/1.0"
	}
	
	return &ProtocolFingerprinter{
		timeout:         config.Timeout,
		maxProbeAttempts: config.MaxProbeAttempts,
		userAgent:       config.UserAgent,
	}
}

// FingerprintProtocol performs comprehensive protocol fingerprinting
func (pf *ProtocolFingerprinter) FingerprintProtocol(host string, port int) *ProtocolFingerprint {
	startTime := time.Now()
	
	fingerprint := &ProtocolFingerprint{
		Host:      host,
		Port:      port,
		Timestamp: startTime,
		Metadata:  make(map[string]string),
	}
	
	// Try different protocol detection methods
	pf.detectProtocol(fingerprint)
	
	// Set final duration
	fingerprint.Duration = time.Since(startTime).String()
	
	return fingerprint
}

// detectProtocol attempts to detect the protocol using various methods
func (pf *ProtocolFingerprinter) detectProtocol(fp *ProtocolFingerprint) {
	// First, try TLS detection
	if pf.probeTLS(fp) {
		return
	}
	
	// Then try HTTP detection
	if pf.probeHTTP(fp) {
		return
	}
	
	// Try SSH detection
	if pf.probeSSH(fp) {
		return
	}
	
	// Try MySQL detection
	if pf.probeMySQL(fp) {
		return
	}
	
	// Try generic TCP banner
	pf.probeGenericTCP(fp)
}

// probeTLS attempts TLS connection and extracts certificate information
func (pf *ProtocolFingerprinter) probeTLS(fp *ProtocolFingerprint) bool {
	address := fmt.Sprintf("%s:%d", fp.Host, fp.Port)
	
	// Try TLS connection
	config := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         fp.Host,
	}
	
	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: pf.timeout,
	}, "tcp", address, config)
	
	if err != nil {
		return false
	}
	defer conn.Close()
	
	// Successfully connected via TLS
	fp.Protocol = "tls"
	fp.Service = "https"
	fp.Confidence = 90
	
	// Extract TLS information
	state := conn.ConnectionState()
	fp.TLS = &TLSInfo{
		Version:     pf.getTLSVersion(state.Version),
		CipherSuite: tls.CipherSuiteName(state.CipherSuite),
	}
	
	// Extract certificate information
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		fp.TLS.Certificate = &CertInfo{
			Subject:     cert.Subject.String(),
			Issuer:      cert.Issuer.String(),
			CommonName:  cert.Subject.CommonName,
			SANs:        cert.DNSNames,
			NotBefore:   cert.NotBefore,
			NotAfter:    cert.NotAfter,
			Fingerprint: fmt.Sprintf("%x", cert.Raw[:10]), // Simplified fingerprint
		}
		fp.Confidence += 10
	}
	
	// Try to detect underlying HTTP service
	if pf.isHTTPSPort(fp.Port) {
		pf.probeHTTPS(fp, conn)
	}
	
	return true
}

// probeHTTPS performs HTTP over TLS detection
func (pf *ProtocolFingerprinter) probeHTTPS(fp *ProtocolFingerprint, tlsConn *tls.Conn) {
	// Send HTTP request over TLS connection
	request := fmt.Sprintf("HEAD / HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\n\r\n", 
		fp.Host, pf.userAgent)
	
	tlsConn.SetDeadline(time.Now().Add(pf.timeout))
	tlsConn.Write([]byte(request))
	
	// Read response
	buffer := make([]byte, 2048)
	n, err := tlsConn.Read(buffer)
	if err == nil && n > 0 {
		response := string(buffer[:n])
		pf.parseHTTPResponse(fp, response)
	}
}

// probeHTTP attempts HTTP connection and extracts HTTP information
func (pf *ProtocolFingerprinter) probeHTTP(fp *ProtocolFingerprint) bool {
	if !pf.isHTTPPort(fp.Port) {
		return false
	}
	
	url := fmt.Sprintf("http://%s:%d/", fp.Host, fp.Port)
	
	client := &http.Client{
		Timeout: pf.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, just capture them
			return http.ErrUseLastResponse
		},
	}
	
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", pf.userAgent)
	
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Successfully connected via HTTP
	fp.Protocol = "tcp"
	fp.Service = "http"
	fp.Confidence = 85
	
	// Extract HTTP information
	fp.HTTP = &HTTPInfo{
		Status:  resp.Status,
		Server:  resp.Header.Get("Server"),
		Headers: make(map[string]string),
	}
	
	// Copy important headers
	importantHeaders := []string{"Server", "X-Powered-By", "Content-Type", "Location"}
	for _, header := range importantHeaders {
		if value := resp.Header.Get(header); value != "" {
			fp.HTTP.Headers[header] = value
		}
	}
	
	// Check for redirect
	if location := resp.Header.Get("Location"); location != "" {
		fp.HTTP.RedirectURL = location
	}
	
	// Detect web technologies
	pf.detectWebTechnologies(fp)
	
	return true
}

// probeSSH attempts SSH connection and extracts SSH information  
func (pf *ProtocolFingerprinter) probeSSH(fp *ProtocolFingerprint) bool {
	if fp.Port != 22 && !pf.isSSHPort(fp.Port) {
		return false
	}
	
	address := fmt.Sprintf("%s:%d", fp.Host, fp.Port)
	conn, err := net.DialTimeout("tcp", address, pf.timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	
	conn.SetReadDeadline(time.Now().Add(pf.timeout))
	
	// Read SSH banner
	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	if err != nil || n == 0 {
		return false
	}
	
	banner := strings.TrimSpace(string(buffer[:n]))
	
	// Check if it's SSH
	if !strings.HasPrefix(banner, "SSH-") {
		return false
	}
	
	fp.Protocol = "tcp"
	fp.Service = "ssh"
	fp.Confidence = 95
	
	// Parse SSH banner
	fp.SSH = &SSHInfo{}
	parts := strings.Split(banner, "-")
	if len(parts) >= 2 {
		fp.SSH.Version = parts[1]
	}
	if len(parts) >= 3 {
		fp.SSH.Implementation = parts[2]
		fp.Version = parts[2]
	}
	
	// Detect SSH implementation
	if strings.Contains(banner, "OpenSSH") {
		fp.Application = "openssh"
		fp.Confidence += 5
	}
	
	return true
}

// probeMySQL attempts MySQL connection and extracts MySQL information
func (pf *ProtocolFingerprinter) probeMySQL(fp *ProtocolFingerprint) bool {
	if fp.Port != 3306 {
		return false
	}
	
	address := fmt.Sprintf("%s:%d", fp.Host, fp.Port)
	conn, err := net.DialTimeout("tcp", address, pf.timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	
	conn.SetReadDeadline(time.Now().Add(pf.timeout))
	
	// Read MySQL handshake packet
	buffer := make([]byte, 512)
	n, err := conn.Read(buffer)
	if err != nil || n < 10 {
		return false
	}
	
	// Basic MySQL packet validation
	if buffer[4] != 0x0a { // Protocol version should be 10
		return false
	}
	
	fp.Protocol = "tcp"
	fp.Service = "mysql"
	fp.Application = "mysql"
	fp.Confidence = 80
	
	// Extract version string (null-terminated after protocol version)
	versionStart := 5
	versionEnd := versionStart
	for versionEnd < n && buffer[versionEnd] != 0 {
		versionEnd++
	}
	
	if versionEnd > versionStart {
		version := string(buffer[versionStart:versionEnd])
		fp.Version = version
		fp.MySQL = &MySQLInfo{
			Version:       version,
			ServerVersion: version,
			Protocol:      10,
		}
		fp.Confidence += 10
	}
	
	return true
}

// probeGenericTCP performs generic TCP banner grabbing
func (pf *ProtocolFingerprinter) probeGenericTCP(fp *ProtocolFingerprint) {
	address := fmt.Sprintf("%s:%d", fp.Host, fp.Port)
	conn, err := net.DialTimeout("tcp", address, pf.timeout)
	if err != nil {
		fp.Error = err.Error()
		return
	}
	defer conn.Close()
	
	conn.SetReadDeadline(time.Now().Add(pf.timeout))
	
	// Read any banner
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil && n == 0 {
		fp.Error = "no response received"
		return
	}
	
	if n > 0 {
		banner := strings.TrimSpace(string(buffer[:n]))
		fp.Protocol = "tcp"
		fp.Service = pf.detectServiceFromBanner(banner, fp.Port)
		fp.Confidence = 40
		fp.Metadata["banner"] = banner
	}
}

// Helper methods

func (pf *ProtocolFingerprinter) getTLSVersion(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

func (pf *ProtocolFingerprinter) isHTTPSPort(port int) bool {
	httpsports := []int{443, 8443, 8080, 8000, 9443}
	for _, p := range httpsports {
		if port == p {
			return true
		}
	}
	return false
}

func (pf *ProtocolFingerprinter) isHTTPPort(port int) bool {
	httpPorts := []int{80, 8080, 8000, 8008, 3000, 5000}
	for _, p := range httpPorts {
		if port == p {
			return true
		}
	}
	return false
}

func (pf *ProtocolFingerprinter) isSSHPort(port int) bool {
	sshPorts := []int{22, 2222}
	for _, p := range sshPorts {
		if port == p {
			return true
		}
	}
	return false
}

func (pf *ProtocolFingerprinter) parseHTTPResponse(fp *ProtocolFingerprint, response string) {
	lines := strings.Split(response, "\n")
	if len(lines) == 0 {
		return
	}
	
	// Parse status line
	if strings.HasPrefix(lines[0], "HTTP/") {
		fp.HTTP = &HTTPInfo{
			Status:  strings.TrimSpace(lines[0]),
			Headers: make(map[string]string),
		}
		
		// Parse headers
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				fp.HTTP.Headers[key] = value
				
				if key == "Server" {
					fp.HTTP.Server = value
					fp.Application = pf.detectServerFromHeader(value)
				}
			}
		}
	}
}

func (pf *ProtocolFingerprinter) detectWebTechnologies(fp *ProtocolFingerprint) {
	if fp.HTTP == nil {
		return
	}
	
	var technologies []string
	
	// Detect from Server header
	if server := fp.HTTP.Server; server != "" {
		server = strings.ToLower(server)
		if strings.Contains(server, "apache") {
			technologies = append(technologies, "Apache")
		}
		if strings.Contains(server, "nginx") {
			technologies = append(technologies, "Nginx")
		}
		if strings.Contains(server, "iis") {
			technologies = append(technologies, "IIS")
		}
	}
	
	// Detect from X-Powered-By header
	if poweredBy := fp.HTTP.Headers["X-Powered-By"]; poweredBy != "" {
		poweredBy = strings.ToLower(poweredBy)
		if strings.Contains(poweredBy, "php") {
			technologies = append(technologies, "PHP")
		}
		if strings.Contains(poweredBy, "asp.net") {
			technologies = append(technologies, "ASP.NET")
		}
	}
	
	fp.HTTP.Technologies = technologies
}

func (pf *ProtocolFingerprinter) detectServerFromHeader(server string) string {
	server = strings.ToLower(server)
	if strings.Contains(server, "apache") {
		return "apache"
	}
	if strings.Contains(server, "nginx") {
		return "nginx"
	}
	if strings.Contains(server, "iis") {
		return "iis"
	}
	return "unknown"
}

func (pf *ProtocolFingerprinter) detectServiceFromBanner(banner string, port int) string {
	banner = strings.ToLower(banner)
	
	// Try port-based detection first
	switch port {
	case 21:
		return "ftp"
	case 23:
		return "telnet"
	case 25:
		return "smtp"
	case 53:
		return "dns"
	case 110:
		return "pop3"
	case 143:
		return "imap"
	case 993:
		return "imaps"
	case 995:
		return "pop3s"
	}
	
	// Try banner-based detection
	if strings.Contains(banner, "ftp") {
		return "ftp"
	}
	if strings.Contains(banner, "smtp") {
		return "smtp"
	}
	if strings.Contains(banner, "pop") {
		return "pop3"
	}
	if strings.Contains(banner, "imap") {
		return "imap"
	}
	
	return "unknown"
}

// FingerprintMultiple performs fingerprinting on multiple targets concurrently
func (pf *ProtocolFingerprinter) FingerprintMultiple(targets []Target, concurrency int) []*ProtocolFingerprint {
	if concurrency <= 0 {
		concurrency = 10
	}
	
	// Create channels
	targetChan := make(chan Target, len(targets))
	resultChan := make(chan *ProtocolFingerprint, len(targets))
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		go func() {
			for target := range targetChan {
				result := pf.FingerprintProtocol(target.Host, target.Port)
				resultChan <- result
			}
		}()
	}
	
	// Send targets
	go func() {
		defer close(targetChan)
		for _, target := range targets {
			targetChan <- target
		}
	}()
	
	// Collect results
	var results []*ProtocolFingerprint
	for i := 0; i < len(targets); i++ {
		result := <-resultChan
		results = append(results, result)
	}
	
	return results
}