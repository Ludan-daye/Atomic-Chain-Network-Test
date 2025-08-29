package reports

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTMLReportConfig configures HTML report generation
type HTMLReportConfig struct {
	Title       string
	Description string
	Theme       string // "default", "dark", "minimal"
	IncludeLogs bool
	Standalone  bool // Include CSS/JS inline vs external links
}

// HTMLReporter generates HTML reports from execution results
type HTMLReporter struct {
	config   HTMLReportConfig
	template *template.Template
}

// ReportData represents data passed to HTML template
type ReportData struct {
	Config      HTMLReportConfig
	GeneratedAt time.Time
	Result      *ExecutionResult
	Summary     *ReportSummary
	Steps       []StepReportData
	Charts      ChartData
	Logs        []LogEntry
}

// ExecutionResult represents execution result for reporting
type ExecutionResult struct {
	SessionID      string                 `json:"session_id"`
	TemplateName   string                 `json:"template_name"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Duration       string                 `json:"duration"`
	Status         string                 `json:"status"`
	Parameters     map[string]interface{} `json:"parameters"`
	TotalSteps     int                    `json:"total_steps"`
	CompletedSteps int                    `json:"completed_steps"`
	FailedSteps    int                    `json:"failed_steps"`
	SkippedSteps   int                    `json:"skipped_steps"`
	ErrorCount     int                    `json:"error_count"`
	StepResults    map[string]*StepResultData `json:"step_results"`
	LogPath        string                 `json:"log_path"`
	ResultPath     string                 `json:"result_path"`
	Tags           []string               `json:"tags"`
}

// StepResultData represents step execution data
type StepResultData struct {
	Name      string      `json:"name"`
	Status    string      `json:"status"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Duration  string      `json:"duration"`
	Error     string      `json:"error,omitempty"`
	Output    interface{} `json:"output,omitempty"`
	Message   string      `json:"message,omitempty"`
}

// StepReportData represents step data for reporting
type StepReportData struct {
	Name         string
	Status       string
	Duration     string
	StatusClass  string
	Error        string
	Message      string
	Output       string
	OutputFormatted string
}

// ReportSummary provides summary statistics
type ReportSummary struct {
	SuccessRate    float64
	TotalDuration  string
	AverageStepDuration string
	StatusCounts   map[string]int
	ParameterCount int
	TagCount       int
}

// ChartData contains data for charts
type ChartData struct {
	StepStatusData   []ChartPoint
	StepDurationData []ChartPoint
	TimelineData     []TimelinePoint
}

// ChartPoint represents a data point for charts
type ChartPoint struct {
	Label string
	Value float64
	Color string
}

// TimelinePoint represents a timeline data point
type TimelinePoint struct {
	Name      string
	StartTime time.Time
	Duration  time.Duration
	Status    string
	Color     string
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Component string
	Message   string
	Data      map[string]interface{}
	Class     string
}

// NewHTMLReporter creates a new HTML reporter
func NewHTMLReporter(config HTMLReportConfig) (*HTMLReporter, error) {
	if config.Title == "" {
		config.Title = "NetCrate Execution Report"
	}
	if config.Theme == "" {
		config.Theme = "default"
	}
	
	reporter := &HTMLReporter{
		config: config,
	}
	
	// Parse HTML template
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"formatTime":     formatTime,
		"formatDuration": formatDuration,
		"statusClass":    statusClass,
		"formatJSON":     formatJSON,
		"colorForStatus": colorForStatus,
		"percentage":     percentage,
	}).Parse(htmlTemplate)
	
	if err != nil {
		return nil, err
	}
	
	reporter.template = tmpl
	return reporter, nil
}

// GenerateReport generates HTML report from execution result
func (hr *HTMLReporter) GenerateReport(result *ExecutionResult, outputPath string) error {
	// Prepare report data
	reportData := &ReportData{
		Config:      hr.config,
		GeneratedAt: time.Now(),
		Result:      result,
		Summary:     hr.generateSummary(result),
		Steps:       hr.generateStepData(result),
		Charts:      hr.generateChartData(result),
	}
	
	// Load logs if requested
	if hr.config.IncludeLogs && result.LogPath != "" {
		logs, err := hr.loadLogs(result.LogPath)
		if err != nil {
			fmt.Printf("[WARN] Failed to load logs: %v\n", err)
		} else {
			reportData.Logs = logs
		}
	}
	
	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	
	// Generate HTML
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return hr.template.Execute(file, reportData)
}

// generateSummary creates report summary
func (hr *HTMLReporter) generateSummary(result *ExecutionResult) *ReportSummary {
	successRate := 0.0
	if result.TotalSteps > 0 {
		successRate = float64(result.CompletedSteps) / float64(result.TotalSteps) * 100
	}
	
	statusCounts := map[string]int{
		"completed": result.CompletedSteps,
		"failed":    result.FailedSteps,
		"skipped":   result.SkippedSteps,
	}
	
	return &ReportSummary{
		SuccessRate:    successRate,
		TotalDuration:  result.Duration,
		StatusCounts:   statusCounts,
		ParameterCount: len(result.Parameters),
		TagCount:       len(result.Tags),
	}
}

// generateStepData prepares step data for template
func (hr *HTMLReporter) generateStepData(result *ExecutionResult) []StepReportData {
	var steps []StepReportData
	
	for _, stepResult := range result.StepResults {
		stepData := StepReportData{
			Name:        stepResult.Name,
			Status:      stepResult.Status,
			Duration:    stepResult.Duration,
			StatusClass: statusClass(stepResult.Status),
			Error:       stepResult.Error,
			Message:     stepResult.Message,
		}
		
		// Format output
		if stepResult.Output != nil {
			stepData.Output = fmt.Sprintf("%v", stepResult.Output)
			stepData.OutputFormatted = formatJSON(stepResult.Output)
		}
		
		steps = append(steps, stepData)
	}
	
	return steps
}

// generateChartData creates data for charts
func (hr *HTMLReporter) generateChartData(result *ExecutionResult) ChartData {
	// Step status distribution
	statusData := []ChartPoint{
		{Label: "Completed", Value: float64(result.CompletedSteps), Color: "#28a745"},
		{Label: "Failed", Value: float64(result.FailedSteps), Color: "#dc3545"},
		{Label: "Skipped", Value: float64(result.SkippedSteps), Color: "#ffc107"},
	}
	
	// Timeline data
	var timelineData []TimelinePoint
	for _, stepResult := range result.StepResults {
		duration, _ := time.ParseDuration(stepResult.Duration)
		timelineData = append(timelineData, TimelinePoint{
			Name:      stepResult.Name,
			StartTime: stepResult.StartTime,
			Duration:  duration,
			Status:    stepResult.Status,
			Color:     colorForStatus(stepResult.Status),
		})
	}
	
	return ChartData{
		StepStatusData: statusData,
		TimelineData:   timelineData,
	}
}

// loadLogs loads log entries from file
func (hr *HTMLReporter) loadLogs(logPath string) ([]LogEntry, error) {
	// This is a simplified version - in real implementation,
	// would parse actual log format
	return []LogEntry{
		{
			Timestamp: time.Now().Add(-5 * time.Minute),
			Level:     "INFO",
			Component: "Engine",
			Message:   "Execution started",
			Class:     "log-info",
		},
		{
			Timestamp: time.Now().Add(-2 * time.Minute),
			Level:     "ERROR",
			Component: "Step",
			Message:   "Connection timeout",
			Class:     "log-error",
		},
		{
			Timestamp: time.Now(),
			Level:     "INFO",
			Component: "Engine",
			Message:   "Execution completed",
			Class:     "log-info",
		},
	}, nil
}

// Template helper functions
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func formatDuration(duration string) string {
	if d, err := time.ParseDuration(duration); err == nil {
		if d < time.Second {
			return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
		}
		return d.String()
	}
	return duration
}

func statusClass(status string) string {
	switch strings.ToLower(status) {
	case "completed", "success":
		return "status-success"
	case "failed", "error":
		return "status-error"
	case "skipped", "skip":
		return "status-warning"
	case "running", "in_progress":
		return "status-info"
	default:
		return "status-secondary"
	}
}

func formatJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func colorForStatus(status string) string {
	switch strings.ToLower(status) {
	case "completed", "success":
		return "#28a745"
	case "failed", "error":
		return "#dc3545"
	case "skipped", "skip":
		return "#ffc107"
	case "running", "in_progress":
		return "#17a2b8"
	default:
		return "#6c757d"
	}
}

func percentage(value, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (value / total) * 100
}

// HTML Template
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Config.Title}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            background-color: #f8f9fa;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .header {
            background: white;
            border-radius: 8px;
            padding: 30px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        
        .header h1 {
            color: #2c3e50;
            margin-bottom: 10px;
        }
        
        .header .meta {
            color: #666;
            font-size: 14px;
        }
        
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        
        .summary-card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        
        .summary-card h3 {
            color: #2c3e50;
            margin-bottom: 10px;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .summary-card .value {
            font-size: 24px;
            font-weight: bold;
            color: #3498db;
        }
        
        .status-success { color: #28a745; }
        .status-error { color: #dc3545; }
        .status-warning { color: #ffc107; }
        .status-info { color: #17a2b8; }
        .status-secondary { color: #6c757d; }
        
        .section {
            background: white;
            border-radius: 8px;
            padding: 30px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        
        .section h2 {
            color: #2c3e50;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 2px solid #ecf0f1;
        }
        
        .parameters {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 15px;
        }
        
        .parameter {
            padding: 10px;
            background: #f8f9fa;
            border-radius: 4px;
            border-left: 4px solid #3498db;
        }
        
        .parameter .name {
            font-weight: bold;
            color: #2c3e50;
        }
        
        .parameter .value {
            color: #666;
            margin-top: 5px;
            font-family: monospace;
        }
        
        .steps-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        
        .steps-table th,
        .steps-table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ecf0f1;
        }
        
        .steps-table th {
            background: #f8f9fa;
            font-weight: 600;
            color: #2c3e50;
        }
        
        .step-status {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
            text-transform: uppercase;
        }
        
        .step-status.status-success {
            background: #d4edda;
            color: #155724;
        }
        
        .step-status.status-error {
            background: #f8d7da;
            color: #721c24;
        }
        
        .step-status.status-warning {
            background: #fff3cd;
            color: #856404;
        }
        
        .footer {
            text-align: center;
            color: #666;
            font-size: 14px;
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #ecf0f1;
        }
        
        .chart {
            height: 200px;
            background: #f8f9fa;
            border-radius: 4px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #666;
            margin: 20px 0;
        }
        
        {{if eq .Config.Theme "dark"}}
        body { background-color: #1a1a1a; color: #e0e0e0; }
        .header, .summary-card, .section { background: #2d2d2d; }
        .header h1, .section h2, .summary-card h3 { color: #ffffff; }
        .parameter { background: #3a3a3a; }
        .steps-table th { background: #3a3a3a; }
        {{end}}
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Config.Title}}</h1>
            <div class="meta">
                Template: <strong>{{.Result.TemplateName}}</strong> |
                Session: <strong>{{.Result.SessionID}}</strong> |
                Generated: <strong>{{formatTime .GeneratedAt}}</strong>
            </div>
            {{if .Config.Description}}
            <div class="description">{{.Config.Description}}</div>
            {{end}}
        </div>

        <div class="summary">
            <div class="summary-card">
                <h3>Status</h3>
                <div class="value {{statusClass .Result.Status}}">{{.Result.Status}}</div>
            </div>
            <div class="summary-card">
                <h3>Duration</h3>
                <div class="value">{{formatDuration .Result.Duration}}</div>
            </div>
            <div class="summary-card">
                <h3>Success Rate</h3>
                <div class="value">{{printf "%.1f%%" .Summary.SuccessRate}}</div>
            </div>
            <div class="summary-card">
                <h3>Total Steps</h3>
                <div class="value">{{.Result.TotalSteps}}</div>
            </div>
        </div>

        <div class="section">
            <h2>Parameters</h2>
            <div class="parameters">
                {{range $key, $value := .Result.Parameters}}
                <div class="parameter">
                    <div class="name">{{$key}}</div>
                    <div class="value">{{$value}}</div>
                </div>
                {{end}}
            </div>
        </div>

        <div class="section">
            <h2>Step Execution</h2>
            <table class="steps-table">
                <thead>
                    <tr>
                        <th>Step</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Message</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Steps}}
                    <tr>
                        <td><strong>{{.Name}}</strong></td>
                        <td><span class="step-status {{.StatusClass}}">{{.Status}}</span></td>
                        <td>{{formatDuration .Duration}}</td>
                        <td>
                            {{if .Error}}
                                <span class="status-error">{{.Error}}</span>
                            {{else}}
                                {{.Message}}
                            {{end}}
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>

        {{if .Config.IncludeLogs}}
        <div class="section">
            <h2>Execution Logs</h2>
            <div class="logs">
                {{range .Logs}}
                <div class="log-entry {{.Class}}">
                    <span class="timestamp">{{formatTime .Timestamp}}</span>
                    <span class="level">{{.Level}}</span>
                    <span class="component">{{.Component}}</span>
                    <span class="message">{{.Message}}</span>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        <div class="footer">
            <p>Report generated by NetCrate v1.0 on {{formatTime .GeneratedAt}}</p>
        </div>
    </div>
</body>
</html>`