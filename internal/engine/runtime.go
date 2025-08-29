package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogLevel defines logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ErrorStrategy defines how to handle errors during execution
type ErrorStrategy string

const (
	ErrorStrategyContinue ErrorStrategy = "continue" // Continue execution, log error
	ErrorStrategySkip     ErrorStrategy = "skip"     // Skip this step, continue with next
	ErrorStrategyFail     ErrorStrategy = "fail"     // Stop execution immediately (default)
)

// RuntimeLogger handles logging during template execution
type RuntimeLogger struct {
	logLevel  LogLevel
	logFile   *os.File
	logPath   string
	sessionID string
	verbose   bool
}

// ExecutionContext holds runtime execution state
type ExecutionContext struct {
	TemplateName string
	SessionID    string
	StartTime    time.Time
	StepResults  map[string]*StepResult
	Logger       *RuntimeLogger
	Parameters   map[string]interface{}
	
	// Error handling
	ErrorCount    int
	SkippedSteps  []string
	FailedSteps   []string
	ContinueOnError bool
}

// StepResult holds the result of a single step execution
type StepResult struct {
	Name      string
	Status    StepStatus
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
	Output    interface{}
	Message   string
}

// StepStatus represents the status of step execution
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusFailed    StepStatus = "failed"
)

// NewRuntimeLogger creates a new runtime logger
func NewRuntimeLogger(sessionID string, verbose bool) (*RuntimeLogger, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	// Create logs directory
	logsDir := filepath.Join(homeDir, ".netcrate", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, err
	}
	
	// Create log file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(logsDir, fmt.Sprintf("netcrate-%s-%s.log", sessionID, timestamp))
	
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	
	logger := &RuntimeLogger{
		logLevel:  LogLevelInfo,
		logFile:   logFile,
		logPath:   logPath,
		sessionID: sessionID,
		verbose:   verbose,
	}
	
	// Set debug level if verbose
	if verbose {
		logger.logLevel = LogLevelDebug
	}
	
	logger.Log(LogLevelInfo, "Logger", "Runtime logger initialized", map[string]interface{}{
		"session_id": sessionID,
		"log_path":   logPath,
		"verbose":    verbose,
	})
	
	return logger, nil
}

// Log writes a log entry
func (l *RuntimeLogger) Log(level LogLevel, component string, message string, data map[string]interface{}) {
	if level < l.logLevel {
		return
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logEntry := fmt.Sprintf("[%s] %s [%s] %s", timestamp, level.String(), component, message)
	
	// Add data if provided
	if len(data) > 0 {
		var parts []string
		for k, v := range data {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		logEntry += fmt.Sprintf(" {%s}", strings.Join(parts, ", "))
	}
	
	// Write to log file
	if l.logFile != nil {
		fmt.Fprintln(l.logFile, logEntry)
		l.logFile.Sync()
	}
	
	// Print to console if verbose or error level
	if l.verbose || level >= LogLevelWarn {
		fmt.Println(logEntry)
	}
}

// Debug logs debug message
func (l *RuntimeLogger) Debug(component string, message string, data map[string]interface{}) {
	l.Log(LogLevelDebug, component, message, data)
}

// Info logs info message
func (l *RuntimeLogger) Info(component string, message string, data map[string]interface{}) {
	l.Log(LogLevelInfo, component, message, data)
}

// Warn logs warning message
func (l *RuntimeLogger) Warn(component string, message string, data map[string]interface{}) {
	l.Log(LogLevelWarn, component, message, data)
}

// Error logs error message
func (l *RuntimeLogger) Error(component string, message string, data map[string]interface{}) {
	l.Log(LogLevelError, component, message, data)
}

// Close closes the logger
func (l *RuntimeLogger) Close() error {
	if l.logFile != nil {
		l.Info("Logger", "Runtime logger closing", map[string]interface{}{
			"session_id": l.sessionID,
		})
		return l.logFile.Close()
	}
	return nil
}

// GetLogPath returns the log file path
func (l *RuntimeLogger) GetLogPath() string {
	return l.logPath
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(templateName, sessionID string, parameters map[string]interface{}, verbose bool) (*ExecutionContext, error) {
	logger, err := NewRuntimeLogger(sessionID, verbose)
	if err != nil {
		return nil, err
	}
	
	ctx := &ExecutionContext{
		TemplateName: templateName,
		SessionID:    sessionID,
		StartTime:    time.Now(),
		StepResults:  make(map[string]*StepResult),
		Logger:       logger,
		Parameters:   parameters,
		ErrorCount:   0,
		SkippedSteps: make([]string, 0),
		FailedSteps:  make([]string, 0),
		ContinueOnError: false,
	}
	
	ctx.Logger.Info("ExecutionContext", "Template execution started", map[string]interface{}{
		"template": templateName,
		"session_id": sessionID,
		"parameters": parameters,
	})
	
	return ctx, nil
}

// StartStep initializes a new step execution
func (ctx *ExecutionContext) StartStep(stepName string) *StepResult {
	result := &StepResult{
		Name:      stepName,
		Status:    StepStatusRunning,
		StartTime: time.Now(),
	}
	
	ctx.StepResults[stepName] = result
	
	ctx.Logger.Info("Step", "Step started", map[string]interface{}{
		"step": stepName,
		"template": ctx.TemplateName,
	})
	
	return result
}

// CompleteStep marks a step as completed
func (ctx *ExecutionContext) CompleteStep(stepName string, output interface{}, message string) {
	if result, exists := ctx.StepResults[stepName]; exists {
		result.Status = StepStatusCompleted
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Output = output
		result.Message = message
		
		ctx.Logger.Info("Step", "Step completed", map[string]interface{}{
			"step": stepName,
			"duration": result.Duration.String(),
			"message": message,
		})
	}
}

// FailStep marks a step as failed
func (ctx *ExecutionContext) FailStep(stepName string, err error, strategy ErrorStrategy) {
	if result, exists := ctx.StepResults[stepName]; exists {
		result.Status = StepStatusFailed
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Error = err
		
		ctx.ErrorCount++
		ctx.FailedSteps = append(ctx.FailedSteps, stepName)
		
		ctx.Logger.Error("Step", "Step failed", map[string]interface{}{
			"step": stepName,
			"error": err.Error(),
			"strategy": string(strategy),
			"duration": result.Duration.String(),
		})
	}
}

// SkipStep marks a step as skipped
func (ctx *ExecutionContext) SkipStep(stepName string, reason string) {
	if result, exists := ctx.StepResults[stepName]; exists {
		result.Status = StepStatusSkipped
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Message = reason
		
		ctx.SkippedSteps = append(ctx.SkippedSteps, stepName)
		
		ctx.Logger.Warn("Step", "Step skipped", map[string]interface{}{
			"step": stepName,
			"reason": reason,
		})
	}
}

// HandleStepError processes a step error according to the specified strategy
func (ctx *ExecutionContext) HandleStepError(stepName string, err error, strategy ErrorStrategy) bool {
	ctx.FailStep(stepName, err, strategy)
	
	switch strategy {
	case ErrorStrategyContinue:
		ctx.Logger.Warn("ErrorHandler", "Continuing execution after error", map[string]interface{}{
			"step": stepName,
			"error": err.Error(),
		})
		return true // Continue execution
		
	case ErrorStrategySkip:
		ctx.Logger.Warn("ErrorHandler", "Skipping remaining dependencies due to error", map[string]interface{}{
			"step": stepName,
			"error": err.Error(),
		})
		return true // Continue execution but skip dependent steps
		
	case ErrorStrategyFail:
		fallthrough
	default:
		ctx.Logger.Error("ErrorHandler", "Stopping execution due to error", map[string]interface{}{
			"step": stepName,
			"error": err.Error(),
		})
		return false // Stop execution
	}
}

// ShouldSkipStep determines if a step should be skipped based on dependencies
func (ctx *ExecutionContext) ShouldSkipStep(stepName, dependsOn string) (bool, string) {
	if dependsOn == "" {
		return false, ""
	}
	
	// Check if dependency failed and had skip strategy
	if result, exists := ctx.StepResults[dependsOn]; exists {
		if result.Status == StepStatusFailed {
			return true, fmt.Sprintf("dependency '%s' failed", dependsOn)
		}
		if result.Status == StepStatusSkipped {
			return true, fmt.Sprintf("dependency '%s' was skipped", dependsOn)
		}
	} else {
		return true, fmt.Sprintf("dependency '%s' not found", dependsOn)
	}
	
	return false, ""
}

// GetExecutionSummary returns a summary of the execution
func (ctx *ExecutionContext) GetExecutionSummary() map[string]interface{} {
	totalSteps := len(ctx.StepResults)
	completedSteps := 0
	failedSteps := 0
	skippedSteps := 0
	
	for _, result := range ctx.StepResults {
		switch result.Status {
		case StepStatusCompleted:
			completedSteps++
		case StepStatusFailed:
			failedSteps++
		case StepStatusSkipped:
			skippedSteps++
		}
	}
	
	duration := time.Since(ctx.StartTime)
	
	summary := map[string]interface{}{
		"template":        ctx.TemplateName,
		"session_id":      ctx.SessionID,
		"start_time":      ctx.StartTime,
		"duration":        duration.String(),
		"total_steps":     totalSteps,
		"completed_steps": completedSteps,
		"failed_steps":    failedSteps,
		"skipped_steps":   skippedSteps,
		"error_count":     ctx.ErrorCount,
		"status":          ctx.getOverallStatus(),
		"log_path":        ctx.Logger.GetLogPath(),
	}
	
	return summary
}

// getOverallStatus determines overall execution status
func (ctx *ExecutionContext) getOverallStatus() string {
	if len(ctx.FailedSteps) > 0 {
		return "failed"
	}
	if len(ctx.SkippedSteps) > 0 {
		return "partial"
	}
	return "success"
}

// PrintExecutionSummary prints execution summary to console
func (ctx *ExecutionContext) PrintExecutionSummary() {
	summary := ctx.GetExecutionSummary()
	
	fmt.Println("\\nExecution Summary:")
	fmt.Println("==================")
	fmt.Printf("Template: %s\\n", summary["template"])
	fmt.Printf("Session:  %s\\n", summary["session_id"])
	fmt.Printf("Duration: %s\\n", summary["duration"])
	fmt.Printf("Status:   %s\\n", summary["status"])
	fmt.Printf("Steps:    %d total, %d completed, %d failed, %d skipped\\n",
		summary["total_steps"], summary["completed_steps"], 
		summary["failed_steps"], summary["skipped_steps"])
	fmt.Printf("Log:      %s\\n", summary["log_path"])
	
	// Show failed steps
	if len(ctx.FailedSteps) > 0 {
		fmt.Printf("\\nFailed Steps:\\n")
		for _, step := range ctx.FailedSteps {
			if result := ctx.StepResults[step]; result != nil {
				fmt.Printf("  • %s: %s\\n", step, result.Error.Error())
			}
		}
	}
	
	// Show skipped steps
	if len(ctx.SkippedSteps) > 0 {
		fmt.Printf("\\nSkipped Steps:\\n")
		for _, step := range ctx.SkippedSteps {
			if result := ctx.StepResults[step]; result != nil {
				fmt.Printf("  • %s: %s\\n", step, result.Message)
			}
		}
	}
}

// Close closes the execution context and logger
func (ctx *ExecutionContext) Close() error {
	ctx.Logger.Info("ExecutionContext", "Template execution completed", ctx.GetExecutionSummary())
	return ctx.Logger.Close()
}