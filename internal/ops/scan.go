package ops

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/netcrate/netcrate/internal/privileges"
)

// ScanOptions contains configuration for port scanning
type ScanOptions struct {
	Targets           []string      `json:"targets"`
	Ports             []int         `json:"ports"`
	ScanType          string        `json:"scan_type"` // "syn", "connect", "udp", "auto"
	ServiceDetection  bool          `json:"service_detection"`
	Rate              int           `json:"rate"`
	Timeout           time.Duration `json:"timeout"`
	Concurrency       int           `json:"concurrency"`
	RetryCount        int           `json:"retry_count"`
}

// ScanResult represents the result of a port scan
type ScanResult struct {
	Host      string                 `json:"host"`
	Port      int                    `json:"port"`
	Status    string                 `json:"status"`   // "open", "closed", "filtered", "error"
	Protocol  string                 `json:"protocol"` // "tcp", "udp"
	RTT       float64                `json:"rtt"`      // milliseconds
	Service   *ServiceInfo           `json:"service,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ServiceInfo contains detected service information
type ServiceInfo struct {
	Name       string  `json:"name"`
	Version    string  `json:"version,omitempty"`
	Banner     string  `json:"banner,omitempty"`
	Confidence float64 `json:"confidence"` // 0.0-1.0
}

// ScanSummary provides summary statistics and results
type ScanSummary struct {
	RunID            string            `json:"run_id"`
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	Duration         float64           `json:"duration"`
	TargetsCount     int               `json:"targets_count"`
	PortsPerTarget   int               `json:"ports_per_target"`
	TotalCombinations int              `json:"total_combinations"`
	OpenPorts        int               `json:"open_ports"`
	ClosedPorts      int               `json:"closed_ports"`
	FilteredPorts    int               `json:"filtered_ports"`
	ScanTypeUsed     string            `json:"scan_type_used"`
	Results          []ScanResult      `json:"results"`
	Stats            ScanStats         `json:"stats"`
	PrivilegeMode    string            `json:"privilege_mode"`
	FallbackReasons  []string          `json:"fallback_reasons,omitempty"`
	PrivilegeSummary map[string]interface{} `json:"privilege_summary,omitempty"`
}

// ScanStats provides detailed scanning statistics
type ScanStats struct {
	HostsScanned   int     `json:"hosts_scanned"`
	PortsScanned   int     `json:"ports_scanned"`
	SuccessRate    float64 `json:"success_rate"`
	AvgRTT         float64 `json:"avg_rtt"`
	ScanRate       float64 `json:"scan_rate"` // actual pps
	ByStatus       map[string]int `json:"by_status"`
	ByService      map[string]int `json:"by_service"`
}

// Predefined port sets
var PortSets = map[string][]int{
	"top100": {
		7, 9, 13, 21, 22, 23, 25, 26, 37, 53, 79, 80, 81, 88, 106, 110, 111, 113,
		119, 135, 139, 143, 144, 179, 199, 389, 427, 443, 444, 445, 465, 513, 514,
		515, 543, 544, 548, 554, 587, 631, 646, 873, 990, 993, 995, 1025, 1026,
		1027, 1028, 1029, 1110, 1433, 1720, 1723, 1755, 1900, 2000, 2001, 2049,
		2121, 2717, 3000, 3128, 3306, 3389, 3986, 4899, 5000, 5009, 5051, 5060,
		5101, 5190, 5357, 5432, 5631, 5666, 5800, 5900, 6000, 6001, 6646, 7070,
		8000, 8008, 8009, 8080, 8081, 8443, 8888, 9100, 9999, 10000, 32768, 49152,
		49153, 49154, 49155, 49156, 49157,
	},
	"web": {80, 443, 8080, 8000, 8443, 8888, 9000, 3000},
	"database": {3306, 5432, 1433, 27017, 6379, 1521, 50000},
	"common": {21, 22, 23, 25, 53, 80, 110, 143, 443, 993, 995},
}

func init() {
	// Build top1000 by extending top100
	top100 := PortSets["top100"]
	additional := []int{
		1, 2, 3, 4, 5, 6, 11, 12, 15, 17, 18, 19, 20, 24, 27, 28, 29, 31, 32, 33,
		35, 36, 38, 39, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 54, 55,
		56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73,
		74, 75, 76, 77, 78, 82, 83, 84, 85, 86, 87, 89, 90, 91, 92, 93, 94, 95,
		96, 97, 98, 99, 100, 101, 102, 105, 107, 108, 109, 112, 114, 115, 116,
		117, 118, 120, 121, 122, 123, 126, 128, 129, 130, 131, 132, 133, 134,
		136, 137, 138, 140, 141, 142, 145, 146, 161, 163, 164, 174, 177, 178,
		180, 195, 311, 340, 350, 366, 371, 383, 384, 387, 411, 412, 413, 414,
		415, 416, 417, 418, 421, 458, 464, 481, 500, 512, 524, 540, 546, 547,
		555, 563, 593, 616, 617, 625, 636, 648, 666, 683, 705, 711, 714, 720,
		722, 726, 749, 765, 777, 783, 787, 800, 801, 808, 843, 864, 871, 901,
		902, 903, 911, 912, 981, 987, 992, 1000, 1001, 1002, 1007, 1009, 1010,
		1011, 1021, 1022, 1023, 1024, 1030, 1031, 1032, 1033, 1034, 1035, 1036,
		1037, 1038, 1039, 1040, 1041, 1042, 1043, 1044, 1045, 1046, 1047, 1048,
		1049, 1050, 1051, 1052, 1053, 1054, 1055, 1056, 1057, 1058, 1059, 1060,
		1061, 1062, 1063, 1064, 1065, 1066, 1067, 1068, 1069, 1070, 1071, 1072,
		1073, 1074, 1075, 1076, 1077, 1078, 1079, 1080, 1081, 1082, 1083, 1084,
		1085, 1086, 1087, 1088, 1089, 1090, 1091, 1092, 1093, 1094, 1095, 1096,
		1097, 1098, 1099, 1100, 1102, 1104, 1105, 1106, 1107, 1108, 1111, 1112,
		1113, 1114, 1117, 1119, 1121, 1122, 1123, 1124, 1126, 1130, 1131, 1132,
		1137, 1138, 1141, 1145, 1147, 1148, 1149, 1151, 1152, 1154, 1163, 1164,
		1165, 1166, 1169, 1174, 1175, 1183, 1185, 1186, 1187, 1192, 1198, 1199,
		1201, 1213, 1216, 1217, 1218, 1233, 1234, 1236, 1244, 1247, 1248, 1259,
		1271, 1272, 1277, 1287, 1296, 1300, 1301, 1309, 1310, 1311, 1322, 1328,
		1334, 1352, 1417, 1434, 1443, 1455, 1461, 1494, 1500, 1501, 1503, 1521,
		1524, 1533, 1556, 1580, 1583, 1594, 1600, 1641, 1658, 1666, 1687, 1688,
		1700, 1717, 1718, 1719, 1721, 1722, 1725, 1726, 1727, 1728, 1729, 1730,
		1731, 1732, 1733, 1734, 1735, 1741, 1748, 1754, 1756, 1761, 1782, 1783,
		1801, 1805, 1812, 1839, 1840, 1862, 1863, 1864, 1875, 1900, 1914, 1935,
		1947, 1971, 1972, 1974, 1984, 1998, 1999, 2003, 2004, 2005, 2006, 2007,
		2008, 2009, 2010, 2013, 2020, 2021, 2022, 2030, 2033, 2034, 2035, 2038,
		2040, 2041, 2042, 2043, 2045, 2046, 2047, 2048, 2050, 2051, 2052, 2053,
		2063, 2064, 2065, 2066, 2067, 2068, 2099, 2100, 2103, 2105, 2106, 2107,
		2111, 2119, 2135, 2144, 2160, 2161, 2170, 2179, 2190, 2191, 2196, 2200,
		2222, 2251, 2260, 2288, 2301, 2323, 2366, 2381, 2382, 2383, 2393, 2394,
		2399, 2401, 2492, 2500, 2522, 2525, 2557, 2601, 2602, 2604, 2605, 2607,
		2608, 2638, 2701, 2702, 2710, 2718, 2725, 2800, 2809, 2811, 2869, 2875,
		2909, 2910, 2920, 2967, 2968, 2998, 3001, 3003, 3005, 3006, 3007, 3011,
		3013, 3017, 3030, 3031, 3052, 3071, 3077, 3128, 3168, 3211, 3221, 3260,
		3261, 3268, 3269, 3283, 3300, 3301, 3322, 3323, 3324, 3325, 3333, 3351,
		3367, 3369, 3370, 3371, 3372, 3389, 3390, 3404, 3476, 3493, 3517, 3527,
		3546, 3551, 3580, 3659, 3689, 3690, 3703, 3737, 3766, 3784, 3800, 3801,
		3809, 3814, 3826, 3827, 3828, 3851, 3869, 3871, 3878, 3880, 3889, 3905,
		3914, 3918, 3920, 3945, 3971, 3986, 3995, 3998, 4000, 4001, 4002, 4003,
		4004, 4005, 4006, 4045, 4111, 4125, 4126, 4129, 4224, 4242, 4279, 4321,
		4343, 4443, 4444, 4445, 4446, 4449, 4550, 4567, 4662, 4848, 4899, 4900,
		4998, 5000, 5001, 5002, 5003, 5004, 5009, 5030, 5033, 5050, 5051, 5054,
		5060, 5061, 5080, 5087, 5100, 5101, 5102, 5120, 5190, 5200, 5214, 5221,
		5222, 5225, 5226, 5269, 5280, 5298, 5357, 5405, 5414, 5431, 5432, 5440,
		5500, 5510, 5544, 5550, 5555, 5560, 5566, 5631, 5633, 5666, 5678, 5679,
		5718, 5730, 5800, 5801, 5802, 5810, 5811, 5815, 5822, 5825, 5850, 5859,
		5862, 5877, 5900, 5901, 5902, 5903, 5904, 5906, 5907, 5910, 5911, 5915,
		5922, 5925, 5950, 5952, 5959, 5960, 5961, 5962, 5963, 5987, 5988, 5989,
		5998, 5999, 6000, 6001, 6002, 6003, 6004, 6005, 6006, 6007, 6009, 6025,
		6059, 6100, 6101, 6106, 6112, 6123, 6129, 6156, 6346, 6389, 6502, 6510,
		6543, 6547, 6565, 6566, 6567, 6580, 6646, 6666, 6667, 6668, 6669, 6689,
		6692, 6699, 6779, 6788, 6789, 6792, 6839, 6881, 6901, 6969, 7000, 7001,
		7002, 7004, 7007, 7019, 7025, 7070, 7100, 7103, 7106, 7200, 7201, 7402,
		7435, 7443, 7496, 7512, 7625, 7627, 7676, 7741, 7777, 7778, 7800, 7911,
		7920, 7921, 7937, 7938, 7999, 8000, 8001, 8002, 8007, 8008, 8009, 8010,
		8011, 8021, 8022, 8031, 8042, 8045, 8080, 8081, 8082, 8083, 8084, 8085,
		8086, 8087, 8088, 8089, 8090, 8093, 8099, 8100, 8180, 8181, 8192, 8193,
		8194, 8200, 8222, 8254, 8290, 8291, 8292, 8300, 8333, 8383, 8400, 8402,
		8443, 8500, 8600, 8649, 8651, 8652, 8654, 8701, 8800, 8873, 8888, 8899,
		8994, 9000, 9001, 9002, 9003, 9009, 9010, 9011, 9040, 9050, 9071, 9080,
		9081, 9090, 9091, 9099, 9100, 9101, 9102, 9103, 9110, 9111, 9200, 9207,
		9220, 9290, 9415, 9418, 9485, 9500, 9502, 9503, 9535, 9575, 9593, 9594,
		9595, 9618, 9666, 9876, 9877, 9878, 9898, 9900, 9917, 9929, 9943, 9944,
		9968, 9998, 9999, 10000, 10001, 10002, 10003, 10004, 10009, 10010, 10012,
		10024, 10025, 10082, 10180, 10215, 10243, 10566, 10616, 10617, 10621,
		10626, 10628, 10629, 10778, 11110, 11111, 11967, 12000, 12174, 12265,
		12345, 13456, 13722, 13782, 13783, 14000, 14238, 14441, 14442, 15000,
		15002, 15003, 15004, 15660, 15742, 16000, 16001, 16012, 16016, 16018,
		16080, 16113, 16992, 16993, 17877, 17988, 18040, 18101, 18988, 19101,
		19283, 19315, 19350, 19780, 19801, 19842, 20000, 20005, 20031, 20221,
		20222, 20828, 21571, 22939, 23502, 24444, 24800, 25734, 25735, 26214,
		27000, 27352, 27353, 27355, 27356, 27715, 28201, 30000, 30718, 30951,
		31038, 31337, 32768, 32769, 32770, 32771, 32772, 32773, 32774, 32775,
		32776, 32777, 32778, 32779, 32780, 32781, 32782, 32783, 32784, 32785,
		33354, 33899, 34571, 34572, 34573, 35500, 38292, 40193, 40911, 41511,
		42510, 44176, 44442, 44443, 44501, 45100, 48080, 49152, 49153, 49154,
		49155, 49156, 49157, 49158, 49159, 49160, 49161, 49163, 49165, 49167,
		49175, 49176, 49400, 49999, 50000, 50001, 50002, 50003, 50006, 50300,
		50389, 50500, 50636, 50800, 51103, 51493, 52673, 52822, 52848, 52869,
		54045, 54328, 55055, 55056, 55555, 55600, 56737, 56738, 57294, 57797,
		58080, 60020, 60443, 61532, 61900, 62078, 63331, 64623, 64680, 65000,
		65129, 65389,
	}
	PortSets["top1000"] = append(top100, additional...)
}

// ScanPorts performs port scanning on the specified targets
func ScanPorts(opts ScanOptions) (*ScanSummary, error) {
	startTime := time.Now()
	runID := fmt.Sprintf("scan_%d", startTime.Unix())

	// Initialize privilege manager for capability detection
	pm := privileges.NewPrivilegeManager()

	// Validate inputs
	if len(opts.Targets) == 0 {
		return nil, fmt.Errorf("no targets specified")
	}
	if len(opts.Ports) == 0 {
		return nil, fmt.Errorf("no ports specified")
	}

	// Set defaults
	if opts.Rate == 0 {
		opts.Rate = 100
	}
	if opts.Timeout == 0 {
		opts.Timeout = 800 * time.Millisecond
	}
	if opts.Concurrency == 0 {
		opts.Concurrency = 200
	}
	if opts.ScanType == "" {
		opts.ScanType = "auto"
	}
	if opts.RetryCount == 0 {
		opts.RetryCount = 1
	}

	// Determine actual scan type based on privileges
	actualScanType := determineScanType(opts.ScanType, pm)

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Calculate total combinations
	totalCombinations := len(opts.Targets) * len(opts.Ports)

	// Rate limiter
	rateLimiter := time.NewTicker(time.Second / time.Duration(opts.Rate))
	defer rateLimiter.Stop()

	// Results channel
	results := make(chan ScanResult, opts.Concurrency)
	
	// Semaphore for concurrency control
	sem := make(chan struct{}, opts.Concurrency)

	var wg sync.WaitGroup
	var stats ScanStats
	stats.ByStatus = make(map[string]int)
	stats.ByService = make(map[string]int)

	// Start scanning workers
	for _, target := range opts.Targets {
		for _, port := range opts.Ports {
			wg.Add(1)
			
			go func(target string, port int) {
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

				result := scanSinglePort(ctx, target, port, actualScanType, opts)
				
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}(target, port)
		}
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []ScanResult
	var totalRTT float64
	uniqueHosts := make(map[string]bool)

	for result := range results {
		allResults = append(allResults, result)
		totalRTT += result.RTT
		uniqueHosts[result.Host] = true

		// Update stats
		stats.ByStatus[result.Status]++
		if result.Service != nil {
			stats.ByService[result.Service.Name]++
		} else {
			stats.ByService["unknown"]++
		}
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Calculate statistics
	stats.HostsScanned = len(uniqueHosts)
	stats.PortsScanned = len(allResults)
	if len(allResults) > 0 {
		stats.SuccessRate = float64(stats.ByStatus["open"]) / float64(len(allResults))
		stats.AvgRTT = totalRTT / float64(len(allResults))
	}
	stats.ScanRate = float64(len(allResults)) / duration.Seconds()

	summary := &ScanSummary{
		RunID:             runID,
		StartTime:         startTime,
		EndTime:           endTime,
		Duration:          duration.Seconds(),
		TargetsCount:      len(opts.Targets),
		PortsPerTarget:    len(opts.Ports),
		TotalCombinations: totalCombinations,
		OpenPorts:         stats.ByStatus["open"],
		ClosedPorts:       stats.ByStatus["closed"],
		FilteredPorts:     stats.ByStatus["filtered"],
		ScanTypeUsed:      actualScanType,
		Results:           allResults,
		Stats:             stats,
		PrivilegeMode:     pm.GetLevel().String(),
		FallbackReasons:   pm.GetFallbackReasons(),
		PrivilegeSummary:  pm.GetPrivilegeSummary(),
	}

	return summary, nil
}

// ParsePortSpec parses port specifications like "top100", "80,443", "8000-9000"
func ParsePortSpec(spec string) ([]int, error) {
	if spec == "" {
		return nil, fmt.Errorf("empty port specification")
	}

	// Check for predefined port sets
	if ports, exists := PortSets[spec]; exists {
		return ports, nil
	}

	var result []int
	parts := strings.Split(spec, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		
		if strings.Contains(part, "-") {
			// Port range
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", part)
			}
			
			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port: %s", rangeParts[0])
			}
			
			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port: %s", rangeParts[1])
			}
			
			if start > end || start < 1 || end > 65535 {
				return nil, fmt.Errorf("invalid port range: %d-%d", start, end)
			}
			
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			// Single port
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", part)
			}
			
			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("port out of range: %d", port)
			}
			
			result = append(result, port)
		}
	}

	return result, nil
}

func determineScanType(requested string, pm *privileges.PrivilegeManager) string {
	switch requested {
	case "syn":
		if pm.HasCapability(privileges.CapabilitySYN) {
			return "syn"
		}
		// Fallback to connect scan
		return "connect"
	case "connect", "udp":
		return requested
	case "auto":
		// Use best available method based on privileges
		if pm.HasCapability(privileges.CapabilitySYN) {
			return "syn"
		} else {
			return "connect"
		}
	default:
		return "connect"
	}
}

func scanSinglePort(ctx context.Context, target string, port int, scanType string, opts ScanOptions) ScanResult {
	result := ScanResult{
		Host:      target,
		Port:      port,
		Status:    "closed",
		Protocol:  "tcp", // Default to TCP
		Timestamp: time.Now(),
	}

	switch scanType {
	case "connect":
		result = tcpConnectScan(ctx, target, port, opts.Timeout, opts.ServiceDetection)
	case "syn":
		result = tcpSynScan(ctx, target, port, opts.Timeout)
	case "udp":
		result = udpScan(ctx, target, port, opts.Timeout)
	default:
		result = tcpConnectScan(ctx, target, port, opts.Timeout, opts.ServiceDetection)
	}

	// Retry on error if configured
	if result.Status == "error" && opts.RetryCount > 0 {
		for i := 0; i < opts.RetryCount; i++ {
			time.Sleep(100 * time.Millisecond) // Brief delay before retry
			retryResult := tcpConnectScan(ctx, target, port, opts.Timeout, opts.ServiceDetection)
			if retryResult.Status != "error" {
				result = retryResult
				break
			}
		}
	}

	return result
}

func tcpConnectScan(ctx context.Context, target string, port int, timeout time.Duration, serviceDetection bool) ScanResult {
	start := time.Now()
	result := ScanResult{
		Host:      target,
		Port:      port,
		Status:    "closed",
		Protocol:  "tcp",
		Timestamp: start,
	}

	address := fmt.Sprintf("%s:%d", target, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	result.RTT = float64(time.Since(start)) / float64(time.Millisecond)

	if err != nil {
		if isConnectionRefused(err) {
			result.Status = "closed"
		} else if isTimeout(err) {
			result.Status = "filtered"
		} else {
			result.Status = "error"
		}
		return result
	}

	result.Status = "open"
	defer conn.Close()

	// Service detection if requested
	if serviceDetection {
		service := detectService(conn, port, 2*time.Second)
		if service != nil {
			result.Service = service
		}
	}

	return result
}

func tcpSynScan(ctx context.Context, target string, port int, timeout time.Duration) ScanResult {
	// SYN scanning requires raw socket privileges
	// For now, fall back to connect scan
	// TODO: Implement actual SYN scanning with raw sockets
	result := tcpConnectScan(ctx, target, port, timeout, false)
	// Mark that we fell back to connect scan
	if result.Status == "open" {
		if result.Service == nil {
			result.Service = &ServiceInfo{}
		}
		result.Service.Banner = "[fallback: connect scan used instead of SYN]"
	}
	return result
}

func udpScan(ctx context.Context, target string, port int, timeout time.Duration) ScanResult {
	start := time.Now()
	result := ScanResult{
		Host:      target,
		Port:      port,
		Status:    "open|filtered", // UDP is tricky to determine
		Protocol:  "udp",
		Timestamp: start,
	}

	address := fmt.Sprintf("%s:%d", target, port)
	conn, err := net.DialTimeout("udp", address, timeout)
	result.RTT = float64(time.Since(start)) / float64(time.Millisecond)

	if err != nil {
		result.Status = "error"
		return result
	}

	defer conn.Close()

	// Try sending a UDP packet and see if we get a response
	_, err = conn.Write([]byte("NetCrate"))
	if err == nil {
		// Set a short read timeout
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		buffer := make([]byte, 1024)
		_, err = conn.Read(buffer)
		if err == nil {
			result.Status = "open"
		}
	}

	return result
}

func detectService(conn net.Conn, port int, timeout time.Duration) *ServiceInfo {
	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(timeout))

	// Try to read banner
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	
	var banner string
	if err == nil && n > 0 {
		banner = strings.TrimSpace(string(buffer[:n]))
	}

	// Service detection based on port and banner
	service := &ServiceInfo{
		Name:       guessServiceByPort(port),
		Confidence: 0.5, // Default confidence
	}

	if banner != "" {
		service.Banner = banner
		service.Confidence = 0.8

		// Improve service detection based on banner
		if detectedService := guessServiceByBanner(banner); detectedService != "" {
			service.Name = detectedService
			service.Confidence = 0.9
		}

		// Extract version if possible
		if version := extractVersion(banner); version != "" {
			service.Version = version
			service.Confidence = 0.95
		}
	}

	return service
}

func guessServiceByPort(port int) string {
	commonServices := map[int]string{
		21:   "ftp",
		22:   "ssh",
		23:   "telnet",
		25:   "smtp",
		53:   "dns",
		80:   "http",
		110:  "pop3",
		143:  "imap",
		443:  "https",
		993:  "imaps",
		995:  "pop3s",
		3306: "mysql",
		5432: "postgresql",
		6379: "redis",
		27017: "mongodb",
	}

	if service, exists := commonServices[port]; exists {
		return service
	}
	return "unknown"
}

func guessServiceByBanner(banner string) string {
	banner = strings.ToLower(banner)
	
	if strings.Contains(banner, "ssh") {
		return "ssh"
	}
	if strings.Contains(banner, "http") || strings.Contains(banner, "html") {
		return "http"
	}
	if strings.Contains(banner, "ftp") {
		return "ftp"
	}
	if strings.Contains(banner, "smtp") || strings.Contains(banner, "mail") {
		return "smtp"
	}
	if strings.Contains(banner, "mysql") {
		return "mysql"
	}
	if strings.Contains(banner, "postgresql") || strings.Contains(banner, "postgres") {
		return "postgresql"
	}
	
	return ""
}

func extractVersion(banner string) string {
	// Simple version extraction patterns
	// This could be much more sophisticated
	if strings.Contains(banner, "OpenSSH") {
		if idx := strings.Index(banner, "OpenSSH_"); idx != -1 {
			version := banner[idx+8:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			return version
		}
	}
	
	return ""
}

func isConnectionRefused(err error) bool {
	return strings.Contains(err.Error(), "connection refused")
}

func isTimeout(err error) bool {
	return strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded")
}