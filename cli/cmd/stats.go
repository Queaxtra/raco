package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type storedStats struct {
	TotalRequests   int     `json:"total_requests"`
	SuccessCount    int     `json:"success_count"`
	FailureCount    int     `json:"failure_count"`
	SuccessRate     float64 `json:"success_rate"`
	AverageDuration int64   `json:"average_duration_ms"`
	MinDuration     int64   `json:"min_duration_ms"`
	MaxDuration     int64   `json:"max_duration_ms"`
}

func RunStats(ctx *Context, args []string) int {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	outputFmt := fs.String("o", "text", "Output format: text, json")
	clear := fs.Bool("clear", false, "Clear statistics")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	statsPath := filepath.Join(ctx.StoragePath, "stats.json")

	if *clear {
		os.Remove(statsPath)
		fmt.Println("Statistics cleared")
		return 0
	}

	data, err := os.ReadFile(statsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No statistics available")
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	var stats storedStats
	if err := json.Unmarshal(data, &stats); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if *outputFmt == "json" {
		output, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(output))
		return 0
	}

	printStatsText(stats)
	return 0
}

func printStatsText(stats storedStats) {
	fmt.Println("Request Statistics")
	fmt.Println("------------------")
	fmt.Printf("Total Requests:    %d\n", stats.TotalRequests)
	fmt.Printf("Successful:        %d\n", stats.SuccessCount)
	fmt.Printf("Failed:            %d\n", stats.FailureCount)
	fmt.Printf("Success Rate:      %.2f%%\n", stats.SuccessRate)
	fmt.Printf("Average Duration:  %dms\n", stats.AverageDuration)
	fmt.Printf("Min Duration:      %dms\n", stats.MinDuration)
	fmt.Printf("Max Duration:      %dms\n", stats.MaxDuration)
}
