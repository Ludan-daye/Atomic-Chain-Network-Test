package ops

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PacketOptions contains configuration for packet sending
type PacketOptions struct {
	Targets            []string               `json:"targets"`
	Template           string                 `json:"template"`
	TemplateParams     map[string]interface{} `json:"template_params"`
	Count              int                    `json:"count"`
	Interval           time.Duration          `json:"interval"`
	Timeout            time.Duration          `json:"timeout"`
	FollowRedirects    bool                   `json:"follow_redirects"`
	MaxResponseSize    int                    `json:"max_response_size"`
}

// PacketResult represents the result of packet sending
type PacketResult struct {
	Target    string                 `json:"target"`
	Sequence  int                    `json:"sequence"`
	Status    string                 `json:"status"` // "success", "timeout", "error"
	RTT       float64                `json:"rtt"`    // milliseconds
	Request   RequestInfo            `json:"request"`
	Response  *ResponseInfo          `json:"response,omitempty"`
	Error     *ErrorInfo             `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// RequestInfo contains request details
type RequestInfo struct {
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers,omitempty"`
	BodySize int               `json:"body_size,omitempty"`
}

// ResponseInfo contains response details
type ResponseInfo struct {
	StatusCode   int               `json:"status_code,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyPreview  string            `json:"body_preview,omitempty"`
	BodySize     int               `json:"body_size"`
	TLSVersion   string            `json:"tls_version,omitempty"`
	CertInfo     *CertInfo         `json:"cert_info,omitempty"`
}

// CertInfo contains certificate information
type CertInfo struct {
	Subject string    `json:"subject"`
	Issuer  string    `json:"issuer"`
	Expires time.Time `json:"expires"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// PacketSummary provides summary of packet sending results
type PacketSummary struct {
	RunID               string                    `json:"run_id"`
	TemplateUsed        string                    `json:"template_used"`
	TargetsCount        int                       `json:"targets_count"`
	TotalPackets        int                       `json:"total_packets"`
	SuccessfulResponses int                       `json:"successful_responses"`
	Results             []PacketResult            `json:"results"`
	Stats               PacketStats               `json:"stats"`
}

// PacketStats provides packet sending statistics
type PacketStats struct {
	AvgRTT         float64            `json:"avg_rtt"`
	MinRTT         float64            `json:"min_rtt"`
	MaxRTT         float64            `json:"max_rtt"`
	SuccessRate    float64            `json:"success_rate"`
	ByStatusCode   map[string]int     `json:"by_status_code"`
	ByTemplate     map[string]int     `json:"by_template"`
}

// PacketTemplate defines a packet template
type PacketTemplate struct {
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	RequiredParams      []string               `json:"required_params"`
	OptionalParams      []string               `json:"optional_params"`
	DefaultParams       map[string]interface{} `json:"default_params"`
	RequiresRawSocket   bool                   `json:"requires_raw_socket"`
}

// Available packet templates
var PacketTemplates = map[string]PacketTemplate{
	"syn": {
		Name:              "TCP SYN",
		Description:       "TCP SYN probe packet",
		RequiredParams:    []string{},
		OptionalParams:    []string{"tcp_flags", "window_size"},
		RequiresRawSocket: true,
		DefaultParams:     map[string]interface{}{},
	},
	"connect": {
		Name:           "TCP Connect",
		Description:    "TCP connection test",
		RequiredParams: []string{},
		OptionalParams: []string{"timeout"},
		DefaultParams: map[string]interface{}{
			"timeout": "5s",
		},
	},
	"http": {
		Name:           "HTTP Request",
		Description:    "HTTP/HTTPS request",
		RequiredParams: []string{},
		OptionalParams: []string{"method", "path", "headers", "body", "user_agent"},
		DefaultParams: map[string]interface{}{
			"method":     "GET",
			"path":       "/",
			"user_agent": "NetCrate/1.0",
		},
	},
	"https": {
		Name:           "HTTPS Request",
		Description:    "HTTPS request with TLS info",
		RequiredParams: []string{},
		OptionalParams: []string{"method", "path", "headers", "sni", "verify_cert"},
		DefaultParams: map[string]interface{}{
			"method":      "GET",
			"path":        "/",
			"user_agent":  "NetCrate/1.0",
			"verify_cert": false,
		},
	},
	"tls": {
		Name:           "TLS Handshake",
		Description:    "TLS handshake probe",
		RequiredParams: []string{},
		OptionalParams: []string{"sni", "version", "ciphers"},
		DefaultParams: map[string]interface{}{
			"version": "1.3",
		},
	},
	"icmp": {
		Name:              "ICMP Ping",
		Description:       "ICMP echo request",
		RequiredParams:    []string{},
		OptionalParams:    []string{"type", "code", "payload"},
		RequiresRawSocket: true,
		DefaultParams: map[string]interface{}{
			"type": "echo",
		},
	},
	"udp": {
		Name:           "UDP Probe",
		Description:    "UDP packet probe",
		RequiredParams: []string{},
		OptionalParams: []string{"payload"},
		DefaultParams: map[string]interface{}{
			"payload": "NetCrate",
		},
	},
}

// SendPackets sends packets using the specified template
func SendPackets(opts PacketOptions) (*PacketSummary, error) {
	startTime := time.Now()
	runID := fmt.Sprintf("packet_%d", startTime.Unix())

	// Validate inputs
	if len(opts.Targets) == 0 {
		return nil, fmt.Errorf("no targets specified")
	}
	if opts.Template == "" {
		return nil, fmt.Errorf("no template specified")
	}

	template, exists := PacketTemplates[opts.Template]
	if !exists {
		return nil, fmt.Errorf("unknown template: %s", opts.Template)
	}

	// Set defaults
	if opts.Count == 0 {
		opts.Count = 1
	}
	if opts.Interval == 0 {
		opts.Interval = 100 * time.Millisecond
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.MaxResponseSize == 0 {
		opts.MaxResponseSize = 1024 * 1024 // 1MB
	}
	if opts.TemplateParams == nil {
		opts.TemplateParams = make(map[string]interface{})
	}

	// Merge default parameters
	for key, value := range template.DefaultParams {
		if _, exists := opts.TemplateParams[key]; !exists {
			opts.TemplateParams[key] = value
		}
	}

	// Validate required parameters
	for _, param := range template.RequiredParams {
		if _, exists := opts.TemplateParams[param]; !exists {
			return nil, fmt.Errorf("missing required parameter: %s", param)
		}
	}

	// Send packets
	var allResults []PacketResult
	var stats PacketStats
	stats.ByStatusCode = make(map[string]int)
	stats.ByTemplate = make(map[string]int)
	stats.MinRTT = float64(^uint(0) >> 1) // Max float64

	for _, target := range opts.Targets {
		for i := 0; i < opts.Count; i++ {
			if i > 0 {
				time.Sleep(opts.Interval)
			}

			result := sendSinglePacket(target, i+1, opts.Template, opts)
			allResults = append(allResults, result)

			// Update statistics
			if result.Status == "success" {
				stats.ByTemplate[opts.Template]++
				if result.Response != nil {
					statusCode := strconv.Itoa(result.Response.StatusCode)
					stats.ByStatusCode[statusCode]++
				}
			}

			// Update RTT stats
			if result.RTT > 0 {
				if result.RTT < stats.MinRTT {
					stats.MinRTT = result.RTT
				}
				if result.RTT > stats.MaxRTT {
					stats.MaxRTT = result.RTT
				}
			}
		}
	}

	// Calculate final statistics
	var totalRTT float64
	var successCount int
	for _, result := range allResults {
		totalRTT += result.RTT
		if result.Status == "success" {
			successCount++
		}
	}

	if len(allResults) > 0 {
		stats.AvgRTT = totalRTT / float64(len(allResults))
		stats.SuccessRate = float64(successCount) / float64(len(allResults))
	}

	summary := &PacketSummary{
		RunID:               runID,
		TemplateUsed:        opts.Template,
		TargetsCount:        len(opts.Targets),
		TotalPackets:        len(allResults),
		SuccessfulResponses: successCount,
		Results:             allResults,
		Stats:               stats,
	}

	return summary, nil
}

func sendSinglePacket(target string, sequence int, templateName string, opts PacketOptions) PacketResult {
	start := time.Now()
	result := PacketResult{
		Target:    target,
		Sequence:  sequence,
		Status:    "error",
		Timestamp: start,
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	switch templateName {
	case "syn":
		result = sendSynPacket(ctx, target, sequence, opts)
	case "connect":
		result = sendConnectPacket(ctx, target, sequence, opts)
	case "http":
		result = sendHTTPPacket(ctx, target, sequence, opts, false)
	case "https":
		result = sendHTTPPacket(ctx, target, sequence, opts, true)
	case "tls":
		result = sendTLSPacket(ctx, target, sequence, opts)
	case "icmp":
		result = sendICMPPacket(ctx, target, sequence, opts)
	case "udp":
		result = sendUDPPacket(ctx, target, sequence, opts)
	default:
		result.Error = &ErrorInfo{
			Type:    "unknown_template",
			Message: fmt.Sprintf("unknown template: %s", templateName),
		}
	}

	result.RTT = float64(time.Since(start)) / float64(time.Millisecond)
	return result
}

func sendSynPacket(ctx context.Context, target string, sequence int, opts PacketOptions) PacketResult {
	// SYN packets require raw socket privileges
	// For now, fall back to connect scan
	return sendConnectPacket(ctx, target, sequence, opts)
}

func sendConnectPacket(ctx context.Context, target string, sequence int, opts PacketOptions) PacketResult {
	result := PacketResult{
		Target:   target,
		Sequence: sequence,
		Status:   "error",
		Request: RequestInfo{
			Method: "CONNECT",
		},
	}

	conn, err := net.DialTimeout("tcp", target, opts.Timeout)
	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "connection_failed",
			Message: err.Error(),
		}
		return result
	}
	defer conn.Close()

	result.Status = "success"
	result.Response = &ResponseInfo{
		StatusCode: 200, // Connection successful
	}

	return result
}

func sendHTTPPacket(ctx context.Context, target string, sequence int, opts PacketOptions, useHTTPS bool) PacketResult {
	result := PacketResult{
		Target:   target,
		Sequence: sequence,
		Status:   "error",
	}

	// Parse target to extract host and port
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		// Assume it's just a host, add default port
		host = target
		if useHTTPS {
			port = "443"
		} else {
			port = "80"
		}
		target = net.JoinHostPort(host, port)
	}

	// Build URL
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}

	path := getStringParam(opts.TemplateParams, "path", "/")
	url := fmt.Sprintf("%s://%s%s", scheme, target, path)

	// Create HTTP request
	method := getStringParam(opts.TemplateParams, "method", "GET")
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "request_creation_failed",
			Message: err.Error(),
		}
		return result
	}

	// Set headers
	userAgent := getStringParam(opts.TemplateParams, "user_agent", "NetCrate/1.0")
	req.Header.Set("User-Agent", userAgent)

	if headersParam, exists := opts.TemplateParams["headers"]; exists {
		if headersStr, ok := headersParam.(string); ok {
			// Parse headers like "Authorization: Bearer token, Content-Type: application/json"
			headerPairs := strings.Split(headersStr, ",")
			for _, pair := range headerPairs {
				parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
				if len(parts) == 2 {
					req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}
		}
	}

	result.Request = RequestInfo{
		Method:  method,
		Headers: make(map[string]string),
	}
	for key, values := range req.Header {
		if len(values) > 0 {
			result.Request.Headers[key] = values[0]
		}
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !opts.FollowRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	if useHTTPS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: getBoolParam(opts.TemplateParams, "verify_cert", false) == false,
				ServerName:         getStringParam(opts.TemplateParams, "sni", host),
			},
		}
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "request_failed",
			Message: err.Error(),
		}
		return result
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(opts.MaxResponseSize)))
	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "response_read_failed",
			Message: err.Error(),
		}
		return result
	}

	result.Status = "success"
	result.Response = &ResponseInfo{
		StatusCode:  resp.StatusCode,
		Headers:     make(map[string]string),
		BodySize:    len(body),
		BodyPreview: truncateString(string(body), 1024),
	}

	for key, values := range resp.Header {
		if len(values) > 0 {
			result.Response.Headers[key] = values[0]
		}
	}

	// Extract TLS information if HTTPS
	if useHTTPS && resp.TLS != nil {
		result.Response.TLSVersion = getTLSVersion(resp.TLS.Version)
		if len(resp.TLS.PeerCertificates) > 0 {
			cert := resp.TLS.PeerCertificates[0]
			result.Response.CertInfo = &CertInfo{
				Subject: cert.Subject.String(),
				Issuer:  cert.Issuer.String(),
				Expires: cert.NotAfter,
			}
		}
	}

	return result
}

func sendTLSPacket(ctx context.Context, target string, sequence int, opts PacketOptions) PacketResult {
	result := PacketResult{
		Target:   target,
		Sequence: sequence,
		Status:   "error",
		Request: RequestInfo{
			Method: "TLS_HANDSHAKE",
		},
	}

	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
		target = net.JoinHostPort(host, "443")
	}

	config := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         getStringParam(opts.TemplateParams, "sni", host),
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: opts.Timeout}, "tcp", target, config)
	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "tls_handshake_failed",
			Message: err.Error(),
		}
		return result
	}
	defer conn.Close()

	result.Status = "success"
	result.Response = &ResponseInfo{
		TLSVersion: getTLSVersion(conn.ConnectionState().Version),
	}

	if len(conn.ConnectionState().PeerCertificates) > 0 {
		cert := conn.ConnectionState().PeerCertificates[0]
		result.Response.CertInfo = &CertInfo{
			Subject: cert.Subject.String(),
			Issuer:  cert.Issuer.String(),
			Expires: cert.NotAfter,
		}
	}

	return result
}

func sendICMPPacket(ctx context.Context, target string, sequence int, opts PacketOptions) PacketResult {
	result := PacketResult{
		Target:   target,
		Sequence: sequence,
		Status:   "error",
		Request: RequestInfo{
			Method: "ICMP",
		},
	}

	// Use system ping command
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1000", target)
	output, err := cmd.Output()

	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "ping_failed",
			Message: err.Error(),
		}
		return result
	}

	result.Status = "success"
	result.Response = &ResponseInfo{
		BodyPreview: strings.TrimSpace(string(output)),
	}

	return result
}

func sendUDPPacket(ctx context.Context, target string, sequence int, opts PacketOptions) PacketResult {
	result := PacketResult{
		Target:   target,
		Sequence: sequence,
		Status:   "error",
		Request: RequestInfo{
			Method: "UDP",
		},
	}

	payload := getStringParam(opts.TemplateParams, "payload", "NetCrate")
	result.Request.BodySize = len(payload)

	conn, err := net.DialTimeout("udp", target, opts.Timeout)
	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "udp_connection_failed",
			Message: err.Error(),
		}
		return result
	}
	defer conn.Close()

	// Send payload
	_, err = conn.Write([]byte(payload))
	if err != nil {
		result.Error = &ErrorInfo{
			Type:    "udp_send_failed",
			Message: err.Error(),
		}
		return result
	}

	// Try to read response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)

	result.Status = "success"
	result.Response = &ResponseInfo{}

	if err == nil && n > 0 {
		result.Response.BodySize = n
		result.Response.BodyPreview = truncateString(string(buffer[:n]), 512)
	}

	return result
}

// Helper functions

func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, exists := params[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, exists := params[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
		if str, ok := val.(string); ok {
			return strings.ToLower(str) == "true"
		}
	}
	return defaultValue
}

func getTLSVersion(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS13:
		return "1.3"
	default:
		return "unknown"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}