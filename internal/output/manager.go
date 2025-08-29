package output

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/netcrate/netcrate/internal/quick"
)

// RunInfo holds metadata about a saved run
type RunInfo struct {
	RunID     string    `json:"run_id"`
	StartTime time.Time `json:"start_time"`
	Duration  float64   `json:"duration"`
	Type      string    `json:"type"`      // "quick", "ops"
	Summary   string    `json:"summary"`   // Brief description
	FilePath  string    `json:"file_path"` // Path to result file
}

// ListRuns returns all saved runs from ~/.netcrate/runs/
func ListRuns() ([]RunInfo, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	runsDir := filepath.Join(homeDir, ".netcrate", "runs")
	
	// Check if runs directory exists
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		return []RunInfo{}, nil // No runs yet
	}

	var runs []RunInfo

	err = filepath.WalkDir(runsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for result.json files
		if d.Name() == "result.json" {
			runInfo, err := parseRunFile(path)
			if err != nil {
				fmt.Printf("Warning: Failed to parse %s: %v\n", path, err)
				return nil // Skip this file but continue
			}
			runs = append(runs, runInfo)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan runs directory: %w", err)
	}

	// Sort by start time, newest first
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.After(runs[j].StartTime)
	})

	return runs, nil
}

// GetLastRun returns the most recent run
func GetLastRun() (*RunInfo, error) {
	runs, err := ListRuns()
	if err != nil {
		return nil, err
	}

	if len(runs) == 0 {
		return nil, fmt.Errorf("no saved runs found")
	}

	return &runs[0], nil
}

// GetRunByID finds a specific run by its ID
func GetRunByID(runID string) (*RunInfo, error) {
	runs, err := ListRuns()
	if err != nil {
		return nil, err
	}

	for _, run := range runs {
		if run.RunID == runID {
			return &run, nil
		}
	}

	return nil, fmt.Errorf("run with ID '%s' not found", runID)
}

// LoadQuickResult loads a quick mode result from file
func LoadQuickResult(runInfo *RunInfo) (*quick.QuickResult, error) {
	file, err := os.Open(runInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open result file: %w", err)
	}
	defer file.Close()

	var result quick.QuickResult
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}

	return &result, nil
}

// parseRunFile extracts metadata from a result.json file
func parseRunFile(filePath string) (RunInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return RunInfo{}, err
	}
	defer file.Close()

	var result quick.QuickResult
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&result)
	if err != nil {
		return RunInfo{}, err
	}

	// Generate summary
	summary := generateSummary(&result)

	return RunInfo{
		RunID:     result.RunID,
		StartTime: result.StartTime,
		Duration:  result.Duration,
		Type:      "quick",
		Summary:   summary,
		FilePath:  filePath,
	}, nil
}

// generateSummary creates a brief description of the run results
func generateSummary(result *quick.QuickResult) string {
	if result.Summary.HostsDiscovered == 0 {
		return "No hosts discovered"
	}

	parts := []string{
		fmt.Sprintf("%d hosts", result.Summary.HostsDiscovered),
	}

	if result.Summary.OpenPorts > 0 {
		parts = append(parts, fmt.Sprintf("%d ports", result.Summary.OpenPorts))
	}

	if len(result.Summary.CriticalPorts) > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", len(result.Summary.CriticalPorts)))
	}

	return strings.Join(parts, ", ")
}

// PrintRunsList displays a formatted list of runs
func PrintRunsList(runs []RunInfo) {
	if len(runs) == 0 {
		fmt.Println("No saved runs found.")
		fmt.Println("Run 'netcrate quick' to create your first scan.")
		return
	}

	fmt.Printf("📁 Saved Runs (%d total)\n", len(runs))
	fmt.Println("========================")
	fmt.Printf("%-20s %-12s %-8s %-25s %s\n", 
		"Run ID", "Type", "Duration", "Date", "Summary")
	fmt.Println(strings.Repeat("-", 85))

	for _, run := range runs {
		durationStr := fmt.Sprintf("%.1fs", run.Duration)
		dateStr := run.StartTime.Format("2006-01-02 15:04:05")
		
		fmt.Printf("%-20s %-12s %-8s %-25s %s\n",
			run.RunID, run.Type, durationStr, dateStr, run.Summary)
	}

	fmt.Printf("\nUse 'netcrate output show --run <run-id>' to view details\n")
	fmt.Printf("Use 'netcrate output show --last' to view the latest run\n")
}

// PrintRunDetails displays detailed information about a specific run
func PrintRunDetails(runInfo *RunInfo) error {
	result, err := LoadQuickResult(runInfo)
	if err != nil {
		return fmt.Errorf("failed to load run details: %w", err)
	}

	// Use the existing QuickSummary printer
	quick.PrintQuickSummary(result)

	return nil
}

// CleanOldRuns removes runs older than the specified number of days
func CleanOldRuns(daysToKeep int) (int, error) {
	runs, err := ListRuns()
	if err != nil {
		return 0, err
	}

	cutoffTime := time.Now().AddDate(0, 0, -daysToKeep)
	var cleaned int

	for _, run := range runs {
		if run.StartTime.Before(cutoffTime) {
			// Delete the entire run directory
			runDir := filepath.Dir(run.FilePath)
			err := os.RemoveAll(runDir)
			if err != nil {
				fmt.Printf("Warning: Failed to remove %s: %v\n", runDir, err)
				continue
			}
			cleaned++
		}
	}

	return cleaned, nil
}