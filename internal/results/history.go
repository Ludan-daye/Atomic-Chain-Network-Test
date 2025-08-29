package results

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExecutionResult represents a complete execution result
type ExecutionResult struct {
	// Metadata
	SessionID    string    `json:"session_id"`
	TemplateName string    `json:"template_name"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     string    `json:"duration"`
	Status       string    `json:"status"` // success, failed, partial
	
	// Parameters and configuration
	Parameters map[string]interface{} `json:"parameters"`
	
	// Execution details
	TotalSteps     int `json:"total_steps"`
	CompletedSteps int `json:"completed_steps"`
	FailedSteps    int `json:"failed_steps"`
	SkippedSteps   int `json:"skipped_steps"`
	ErrorCount     int `json:"error_count"`
	
	// Results data
	StepResults map[string]*StepResultData `json:"step_results"`
	
	// File paths
	LogPath    string `json:"log_path"`
	ResultPath string `json:"result_path"`
	
	// Tags for categorization
	Tags []string `json:"tags"`
}

// StepResultData represents the result of a single step
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

// FilterCriteria defines search and filter options
type FilterCriteria struct {
	// Time range
	FromTime *time.Time
	ToTime   *time.Time
	
	// Template and session filters
	TemplateName string
	SessionID    string
	Status       string
	
	// Tag filters
	Tags []string
	
	// Parameter filters
	ParameterFilters map[string]interface{}
	
	// Result limits
	Limit  int
	Offset int
	
	// Sorting
	SortBy    string // "start_time", "duration", "template", "status"
	SortOrder string // "asc", "desc"
}

// HistoryManager manages execution result history
type HistoryManager struct {
	historyDir string
	indexPath  string
	results    map[string]*ExecutionResult
}

// NewHistoryManager creates a new history manager
func NewHistoryManager() (*HistoryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	historyDir := filepath.Join(homeDir, ".netcrate", "results")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, err
	}
	
	manager := &HistoryManager{
		historyDir: historyDir,
		indexPath:  filepath.Join(historyDir, "index.json"),
		results:    make(map[string]*ExecutionResult),
	}
	
	// Load existing results
	if err := manager.LoadResults(); err != nil {
		return nil, err
	}
	
	return manager, nil
}

// SaveResult saves an execution result to history
func (hm *HistoryManager) SaveResult(result *ExecutionResult) error {
	// Generate result file path
	timestamp := result.StartTime.Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.json", result.TemplateName, timestamp)
	resultPath := filepath.Join(hm.historyDir, filename)
	
	// Update result path
	result.ResultPath = resultPath
	
	// Save to file
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(resultPath, data, 0644); err != nil {
		return err
	}
	
	// Add to memory cache
	hm.results[result.SessionID] = result
	
	// Update index
	return hm.saveIndex()
}

// LoadResults loads all results from disk
func (hm *HistoryManager) LoadResults() error {
	// Clear existing results
	hm.results = make(map[string]*ExecutionResult)
	
	// Walk through result files
	err := filepath.Walk(hm.historyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		if !info.IsDir() && strings.HasSuffix(path, ".json") && path != hm.indexPath {
			result, err := hm.loadResultFromFile(path)
			if err != nil {
				fmt.Printf("[WARN] Failed to load result from %s: %v\n", path, err)
				return nil // Continue walking
			}
			
			hm.results[result.SessionID] = result
		}
		
		return nil
	})
	
	return err
}

// loadResultFromFile loads a single result from file
func (hm *HistoryManager) loadResultFromFile(filePath string) (*ExecutionResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var result ExecutionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	// Update file path
	result.ResultPath = filePath
	
	return &result, nil
}

// saveIndex saves the index of all results
func (hm *HistoryManager) saveIndex() error {
	index := make(map[string]interface{})
	index["version"] = "1.0"
	index["last_updated"] = time.Now()
	index["total_results"] = len(hm.results)
	
	// Create summary for each result
	summaries := make([]map[string]interface{}, 0, len(hm.results))
	for _, result := range hm.results {
		summary := map[string]interface{}{
			"session_id":    result.SessionID,
			"template_name": result.TemplateName,
			"start_time":    result.StartTime,
			"duration":      result.Duration,
			"status":        result.Status,
			"result_path":   result.ResultPath,
			"tags":          result.Tags,
		}
		summaries = append(summaries, summary)
	}
	
	index["results"] = summaries
	
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(hm.indexPath, data, 0644)
}

// ListResults returns all results matching the filter criteria
func (hm *HistoryManager) ListResults(criteria *FilterCriteria) ([]*ExecutionResult, error) {
	var filtered []*ExecutionResult
	
	// Apply filters
	for _, result := range hm.results {
		if hm.matchesCriteria(result, criteria) {
			filtered = append(filtered, result)
		}
	}
	
	// Sort results
	if criteria.SortBy != "" {
		hm.sortResults(filtered, criteria.SortBy, criteria.SortOrder)
	} else {
		// Default sort by start time (newest first)
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].StartTime.After(filtered[j].StartTime)
		})
	}
	
	// Apply limit and offset
	if criteria.Offset > 0 {
		if criteria.Offset >= len(filtered) {
			return []*ExecutionResult{}, nil
		}
		filtered = filtered[criteria.Offset:]
	}
	
	if criteria.Limit > 0 && criteria.Limit < len(filtered) {
		filtered = filtered[:criteria.Limit]
	}
	
	return filtered, nil
}

// matchesCriteria checks if a result matches the filter criteria
func (hm *HistoryManager) matchesCriteria(result *ExecutionResult, criteria *FilterCriteria) bool {
	// Time range filter
	if criteria.FromTime != nil && result.StartTime.Before(*criteria.FromTime) {
		return false
	}
	if criteria.ToTime != nil && result.StartTime.After(*criteria.ToTime) {
		return false
	}
	
	// Template name filter
	if criteria.TemplateName != "" && !strings.Contains(strings.ToLower(result.TemplateName), strings.ToLower(criteria.TemplateName)) {
		return false
	}
	
	// Session ID filter
	if criteria.SessionID != "" && !strings.Contains(strings.ToLower(result.SessionID), strings.ToLower(criteria.SessionID)) {
		return false
	}
	
	// Status filter
	if criteria.Status != "" && result.Status != criteria.Status {
		return false
	}
	
	// Tag filters
	if len(criteria.Tags) > 0 {
		hasMatchingTag := false
		for _, reqTag := range criteria.Tags {
			for _, resultTag := range result.Tags {
				if strings.EqualFold(reqTag, resultTag) {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				break
			}
		}
		if !hasMatchingTag {
			return false
		}
	}
	
	// Parameter filters
	if len(criteria.ParameterFilters) > 0 {
		for paramName, expectedValue := range criteria.ParameterFilters {
			if actualValue, exists := result.Parameters[paramName]; !exists || actualValue != expectedValue {
				return false
			}
		}
	}
	
	return true
}

// sortResults sorts the results based on criteria
func (hm *HistoryManager) sortResults(results []*ExecutionResult, sortBy, sortOrder string) {
	ascending := sortOrder != "desc"
	
	switch sortBy {
	case "start_time":
		sort.Slice(results, func(i, j int) bool {
			if ascending {
				return results[i].StartTime.Before(results[j].StartTime)
			}
			return results[i].StartTime.After(results[j].StartTime)
		})
	case "duration":
		sort.Slice(results, func(i, j int) bool {
			dur1, _ := time.ParseDuration(results[i].Duration)
			dur2, _ := time.ParseDuration(results[j].Duration)
			if ascending {
				return dur1 < dur2
			}
			return dur1 > dur2
		})
	case "template":
		sort.Slice(results, func(i, j int) bool {
			if ascending {
				return results[i].TemplateName < results[j].TemplateName
			}
			return results[i].TemplateName > results[j].TemplateName
		})
	case "status":
		sort.Slice(results, func(i, j int) bool {
			if ascending {
				return results[i].Status < results[j].Status
			}
			return results[i].Status > results[j].Status
		})
	}
}

// GetResult retrieves a specific result by session ID
func (hm *HistoryManager) GetResult(sessionID string) (*ExecutionResult, bool) {
	result, exists := hm.results[sessionID]
	return result, exists
}

// DeleteResult removes a result from history
func (hm *HistoryManager) DeleteResult(sessionID string) error {
	result, exists := hm.results[sessionID]
	if !exists {
		return fmt.Errorf("result not found: %s", sessionID)
	}
	
	// Remove result file
	if err := os.Remove(result.ResultPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	
	// Remove log file if it exists
	if result.LogPath != "" {
		if err := os.Remove(result.LogPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("[WARN] Failed to remove log file %s: %v\n", result.LogPath, err)
		}
	}
	
	// Remove from memory cache
	delete(hm.results, sessionID)
	
	// Update index
	return hm.saveIndex()
}

// GetStats returns statistical information about stored results
func (hm *HistoryManager) GetStats() map[string]interface{} {
	if len(hm.results) == 0 {
		return map[string]interface{}{
			"total_results": 0,
		}
	}
	
	// Basic counts
	statusCounts := make(map[string]int)
	templateCounts := make(map[string]int)
	tagCounts := make(map[string]int)
	
	var totalDuration time.Duration
	oldest := time.Now()
	newest := time.Time{}
	
	for _, result := range hm.results {
		// Status counts
		statusCounts[result.Status]++
		
		// Template counts
		templateCounts[result.TemplateName]++
		
		// Tag counts
		for _, tag := range result.Tags {
			tagCounts[tag]++
		}
		
		// Duration
		if dur, err := time.ParseDuration(result.Duration); err == nil {
			totalDuration += dur
		}
		
		// Time range
		if result.StartTime.Before(oldest) {
			oldest = result.StartTime
		}
		if result.StartTime.After(newest) {
			newest = result.StartTime
		}
	}
	
	avgDuration := time.Duration(0)
	if len(hm.results) > 0 {
		avgDuration = totalDuration / time.Duration(len(hm.results))
	}
	
	return map[string]interface{}{
		"total_results":   len(hm.results),
		"status_counts":   statusCounts,
		"template_counts": templateCounts,
		"tag_counts":      tagCounts,
		"oldest_result":   oldest.Format("2006-01-02 15:04:05"),
		"newest_result":   newest.Format("2006-01-02 15:04:05"),
		"total_duration":  totalDuration.String(),
		"average_duration": avgDuration.String(),
		"history_dir":     hm.historyDir,
		"index_path":      hm.indexPath,
	}
}

// CleanupOldResults removes results older than specified duration
func (hm *HistoryManager) CleanupOldResults(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	var toDelete []string
	
	// Find results to delete
	for sessionID, result := range hm.results {
		if result.StartTime.Before(cutoff) {
			toDelete = append(toDelete, sessionID)
		}
	}
	
	// Delete results
	for _, sessionID := range toDelete {
		if err := hm.DeleteResult(sessionID); err != nil {
			fmt.Printf("[WARN] Failed to delete result %s: %v\n", sessionID, err)
		}
	}
	
	return len(toDelete), nil
}