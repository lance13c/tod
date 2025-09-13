package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lance13c/tod/internal/config"
	"github.com/lance13c/tod/internal/llm"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show LLM usage and cost tracking",
	Long: `Display LLM usage statistics and costs for the current session, 
daily, weekly, or monthly periods. Track token usage and associated costs
across different LLM providers and models.`,
	RunE: runUsage,
}

var (
	usageDaily   bool
	usageWeekly  bool
	usageMonthly bool
	usageReset   bool
	usageExport  string
)

func init() {
	rootCmd.AddCommand(usageCmd)
	
	usageCmd.Flags().BoolVar(&usageDaily, "daily", false, "Show daily usage statistics")
	usageCmd.Flags().BoolVar(&usageWeekly, "weekly", false, "Show weekly usage statistics")
	usageCmd.Flags().BoolVar(&usageMonthly, "monthly", false, "Show monthly usage statistics")
	usageCmd.Flags().BoolVar(&usageReset, "reset", false, "Reset usage statistics")
	usageCmd.Flags().StringVar(&usageExport, "export", "", "Export usage data to file (json, csv)")
}

func runUsage(cmd *cobra.Command, args []string) error {
	// Load current usage data
	usageData, err := loadUsageData()
	if err != nil {
		return fmt.Errorf("failed to load usage data: %w", err)
	}
	
	// Handle reset flag
	if usageReset {
		return resetUsageData()
	}
	
	// Handle export flag
	if usageExport != "" {
		return exportUsageData(usageData, usageExport)
	}
	
	// Display usage based on flags
	switch {
	case usageDaily:
		displayDailyUsage(usageData)
	case usageWeekly:
		displayWeeklyUsage(usageData)
	case usageMonthly:
		displayMonthlyUsage(usageData)
	default:
		displaySessionUsage(usageData)
	}
	
	return nil
}

func loadUsageData() (*config.UsageConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	usageFile := filepath.Join(homeDir, ".tod", "usage.json")
	
	// Create default usage config if file doesn't exist
	if _, err := os.Stat(usageFile); os.IsNotExist(err) {
		return &config.UsageConfig{
			Session: config.SessionUsage{
				StartTime: time.Now(),
				Providers: make(map[string]config.ProviderUsage),
			},
			Daily:   make(map[string]config.DailyUsage),
			Weekly:  make(map[string]config.WeeklyUsage),
			Monthly: make(map[string]config.MonthlyUsage),
		}, nil
	}
	
	data, err := os.ReadFile(usageFile)
	if err != nil {
		return nil, err
	}
	
	var usage config.UsageConfig
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, err
	}
	
	return &usage, nil
}

func saveUsageData(usage *config.UsageConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	todDir := filepath.Join(homeDir, ".tod")
	if err := os.MkdirAll(todDir, 0755); err != nil {
		return err
	}
	
	usageFile := filepath.Join(todDir, "usage.json")
	data, err := json.MarshalIndent(usage, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(usageFile, data, 0644)
}

func displaySessionUsage(usage *config.UsageConfig) {
	fmt.Println("┌─ LLM Usage - Current Session ─┐")
	fmt.Printf("│ Started: %s │\n", usage.Session.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("│ Duration: %s │\n", time.Since(usage.Session.StartTime).Truncate(time.Second))
	fmt.Println("└────────────────────────────────┘")
	fmt.Println()
	
	if usage.Session.RequestCount == 0 {
		fmt.Println("No LLM requests made in this session.")
		return
	}
	
	fmt.Printf("Total Requests: %d\n", usage.Session.RequestCount)
	fmt.Printf("Total Tokens:   %s (%s input, %s output)\n", 
		llm.FormatTokens(usage.Session.TotalTokens),
		llm.FormatTokens(usage.Session.InputTokens),
		llm.FormatTokens(usage.Session.OutputTokens))
	fmt.Printf("Total Cost:     %s\n", llm.FormatCost(usage.Session.TotalCost))
	fmt.Println()
	
	if len(usage.Session.Providers) > 0 {
		fmt.Println("By Provider:")
		for provider, providerUsage := range usage.Session.Providers {
			fmt.Printf("  %s (%s):\n", provider, providerUsage.Model)
			fmt.Printf("    Requests: %d\n", providerUsage.RequestCount)
			fmt.Printf("    Tokens:   %s (%s input, %s output)\n", 
				llm.FormatTokens(providerUsage.TotalTokens),
				llm.FormatTokens(providerUsage.InputTokens),
				llm.FormatTokens(providerUsage.OutputTokens))
			fmt.Printf("    Cost:     %s\n", llm.FormatCost(providerUsage.TotalCost))
		}
	}
}

func displayDailyUsage(usage *config.UsageConfig) {
	fmt.Println("┌─ LLM Usage - Daily Summary ─┐")
	fmt.Println("└──────────────────────────────┘")
	fmt.Println()
	
	if len(usage.Daily) == 0 {
		fmt.Println("No daily usage data available.")
		return
	}
	
	// Sort dates
	var dates []string
	for date := range usage.Daily {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	
	totalCost := 0.0
	totalRequests := 0
	
	fmt.Printf("%-12s %8s %12s %12s %10s\n", "Date", "Requests", "Tokens", "Cost", "Models")
	fmt.Println("────────────────────────────────────────────────────────────")
	
	for _, date := range dates {
		daily := usage.Daily[date]
		models := len(daily.Providers)
		
		fmt.Printf("%-12s %8d %12s %12s %10d\n", 
			date, 
			daily.RequestCount,
			llm.FormatTokens(daily.TotalTokens),
			llm.FormatCost(daily.TotalCost),
			models)
		
		totalCost += daily.TotalCost
		totalRequests += daily.RequestCount
	}
	
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("%-12s %8d %12s %12s\n", "Total", totalRequests, "-", llm.FormatCost(totalCost))
}

func displayWeeklyUsage(usage *config.UsageConfig) {
	fmt.Println("┌─ LLM Usage - Weekly Summary ─┐")
	fmt.Println("└───────────────────────────────┘")
	fmt.Println()
	
	if len(usage.Weekly) == 0 {
		fmt.Println("No weekly usage data available.")
		return
	}
	
	// Sort weeks
	var weeks []string
	for week := range usage.Weekly {
		weeks = append(weeks, week)
	}
	sort.Strings(weeks)
	
	fmt.Printf("%-12s %8s %12s %12s\n", "Week", "Requests", "Tokens", "Cost")
	fmt.Println("──────────────────────────────────────────────────")
	
	totalCost := 0.0
	for _, week := range weeks {
		weekly := usage.Weekly[week]
		fmt.Printf("%-12s %8d %12s %12s\n", 
			week, 
			weekly.RequestCount,
			llm.FormatTokens(weekly.TotalTokens),
			llm.FormatCost(weekly.TotalCost))
		totalCost += weekly.TotalCost
	}
	
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("%-12s %8s %12s %12s\n", "Total", "-", "-", llm.FormatCost(totalCost))
}

func displayMonthlyUsage(usage *config.UsageConfig) {
	fmt.Println("┌─ LLM Usage - Monthly Summary ─┐")
	fmt.Println("└────────────────────────────────┘")
	fmt.Println()
	
	if len(usage.Monthly) == 0 {
		fmt.Println("No monthly usage data available.")
		return
	}
	
	// Sort months
	var months []string
	for month := range usage.Monthly {
		months = append(months, month)
	}
	sort.Strings(months)
	
	fmt.Printf("%-10s %8s %12s %12s\n", "Month", "Requests", "Tokens", "Cost")
	fmt.Println("────────────────────────────────────────────────")
	
	totalCost := 0.0
	for _, month := range months {
		monthly := usage.Monthly[month]
		fmt.Printf("%-10s %8d %12s %12s\n", 
			month, 
			monthly.RequestCount,
			llm.FormatTokens(monthly.TotalTokens),
			llm.FormatCost(monthly.TotalCost))
		totalCost += monthly.TotalCost
	}
	
	fmt.Println("────────────────────────────────────────────────")
	fmt.Printf("%-10s %8s %12s %12s\n", "Total", "-", "-", llm.FormatCost(totalCost))
}

func resetUsageData() error {
	fmt.Print("Are you sure you want to reset all usage data? This cannot be undone. (y/N): ")
	var response string
	fmt.Scanln(&response)
	
	if response != "y" && response != "Y" {
		fmt.Println("Reset cancelled.")
		return nil
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	usageFile := filepath.Join(homeDir, ".tod", "usage.json")
	if err := os.Remove(usageFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove usage file: %w", err)
	}
	
	fmt.Println("Usage data reset successfully.")
	return nil
}

func exportUsageData(usage *config.UsageConfig, format string) error {
	switch format {
	case "json":
		return exportJSON(usage)
	case "csv":
		return exportCSV(usage)
	default:
		return fmt.Errorf("unsupported export format: %s (supported: json, csv)", format)
	}
}

func exportJSON(usage *config.UsageConfig) error {
	filename := fmt.Sprintf("tod-usage-%s.json", time.Now().Format("2006-01-02"))
	data, err := json.MarshalIndent(usage, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}
	
	fmt.Printf("Usage data exported to %s\n", filename)
	return nil
}

func exportCSV(usage *config.UsageConfig) error {
	filename := fmt.Sprintf("tod-usage-%s.csv", time.Now().Format("2006-01-02"))
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	fmt.Fprintln(file, "Type,Period,Requests,InputTokens,OutputTokens,TotalTokens,Cost,Provider,Model")
	
	// Export daily data
	var dates []string
	for date := range usage.Daily {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	
	for _, date := range dates {
		daily := usage.Daily[date]
		for provider, providerUsage := range daily.Providers {
			fmt.Fprintf(file, "Daily,%s,%d,%d,%d,%d,%.6f,%s,%s\n",
				date,
				providerUsage.RequestCount,
				providerUsage.InputTokens,
				providerUsage.OutputTokens,
				providerUsage.TotalTokens,
				providerUsage.TotalCost,
				provider,
				providerUsage.Model)
		}
	}
	
	fmt.Printf("Usage data exported to %s\n", filename)
	return nil
}

// UpdateUsage updates the usage tracking data with a new request
func UpdateUsage(stats *llm.UsageStats) error {
	usage, err := loadUsageData()
	if err != nil {
		return err
	}
	
	// Update session usage
	usage.Session.TotalTokens += stats.TotalTokens
	usage.Session.InputTokens += stats.InputTokens
	usage.Session.OutputTokens += stats.OutputTokens
	usage.Session.TotalCost += stats.TotalCost
	usage.Session.RequestCount++
	
	// Update provider usage in session
	if usage.Session.Providers == nil {
		usage.Session.Providers = make(map[string]config.ProviderUsage)
	}
	
	providerKey := stats.Provider
	if existing, exists := usage.Session.Providers[providerKey]; exists {
		existing.TotalTokens += stats.TotalTokens
		existing.InputTokens += stats.InputTokens
		existing.OutputTokens += stats.OutputTokens
		existing.TotalCost += stats.TotalCost
		existing.RequestCount++
		usage.Session.Providers[providerKey] = existing
	} else {
		usage.Session.Providers[providerKey] = config.ProviderUsage{
			Model:        stats.Model,
			TotalTokens:  stats.TotalTokens,
			InputTokens:  stats.InputTokens,
			OutputTokens: stats.OutputTokens,
			TotalCost:    stats.TotalCost,
			RequestCount: 1,
		}
	}
	
	// Update daily, weekly, monthly usage
	now := time.Now()
	updateDailyUsage(usage, stats, now)
	updateWeeklyUsage(usage, stats, now)
	updateMonthlyUsage(usage, stats, now)
	
	return saveUsageData(usage)
}

func updateDailyUsage(usage *config.UsageConfig, stats *llm.UsageStats, now time.Time) {
	dateKey := now.Format("2006-01-02")
	
	if usage.Daily == nil {
		usage.Daily = make(map[string]config.DailyUsage)
	}
	
	daily := usage.Daily[dateKey]
	daily.Date = dateKey
	daily.TotalTokens += stats.TotalTokens
	daily.InputTokens += stats.InputTokens
	daily.OutputTokens += stats.OutputTokens
	daily.TotalCost += stats.TotalCost
	daily.RequestCount++
	
	if daily.Providers == nil {
		daily.Providers = make(map[string]config.ProviderUsage)
	}
	
	providerKey := stats.Provider
	if existing, exists := daily.Providers[providerKey]; exists {
		existing.TotalTokens += stats.TotalTokens
		existing.InputTokens += stats.InputTokens
		existing.OutputTokens += stats.OutputTokens
		existing.TotalCost += stats.TotalCost
		existing.RequestCount++
		daily.Providers[providerKey] = existing
	} else {
		daily.Providers[providerKey] = config.ProviderUsage{
			Model:        stats.Model,
			TotalTokens:  stats.TotalTokens,
			InputTokens:  stats.InputTokens,
			OutputTokens: stats.OutputTokens,
			TotalCost:    stats.TotalCost,
			RequestCount: 1,
		}
	}
	
	usage.Daily[dateKey] = daily
}

func updateWeeklyUsage(usage *config.UsageConfig, stats *llm.UsageStats, now time.Time) {
	year, week := now.ISOWeek()
	weekKey := fmt.Sprintf("%d-W%02d", year, week)
	
	if usage.Weekly == nil {
		usage.Weekly = make(map[string]config.WeeklyUsage)
	}
	
	weekly := usage.Weekly[weekKey]
	weekly.Week = weekKey
	weekly.TotalTokens += stats.TotalTokens
	weekly.InputTokens += stats.InputTokens
	weekly.OutputTokens += stats.OutputTokens
	weekly.TotalCost += stats.TotalCost
	weekly.RequestCount++
	
	if weekly.Providers == nil {
		weekly.Providers = make(map[string]config.ProviderUsage)
	}
	
	providerKey := stats.Provider
	if existing, exists := weekly.Providers[providerKey]; exists {
		existing.TotalTokens += stats.TotalTokens
		existing.InputTokens += stats.InputTokens
		existing.OutputTokens += stats.OutputTokens
		existing.TotalCost += stats.TotalCost
		existing.RequestCount++
		weekly.Providers[providerKey] = existing
	} else {
		weekly.Providers[providerKey] = config.ProviderUsage{
			Model:        stats.Model,
			TotalTokens:  stats.TotalTokens,
			InputTokens:  stats.InputTokens,
			OutputTokens: stats.OutputTokens,
			TotalCost:    stats.TotalCost,
			RequestCount: 1,
		}
	}
	
	usage.Weekly[weekKey] = weekly
}

func updateMonthlyUsage(usage *config.UsageConfig, stats *llm.UsageStats, now time.Time) {
	monthKey := now.Format("2006-01")
	
	if usage.Monthly == nil {
		usage.Monthly = make(map[string]config.MonthlyUsage)
	}
	
	monthly := usage.Monthly[monthKey]
	monthly.Month = monthKey
	monthly.TotalTokens += stats.TotalTokens
	monthly.InputTokens += stats.InputTokens
	monthly.OutputTokens += stats.OutputTokens
	monthly.TotalCost += stats.TotalCost
	monthly.RequestCount++
	
	if monthly.Providers == nil {
		monthly.Providers = make(map[string]config.ProviderUsage)
	}
	
	providerKey := stats.Provider
	if existing, exists := monthly.Providers[providerKey]; exists {
		existing.TotalTokens += stats.TotalTokens
		existing.InputTokens += stats.InputTokens
		existing.OutputTokens += stats.OutputTokens
		existing.TotalCost += stats.TotalCost
		existing.RequestCount++
		monthly.Providers[providerKey] = existing
	} else {
		monthly.Providers[providerKey] = config.ProviderUsage{
			Model:        stats.Model,
			TotalTokens:  stats.TotalTokens,
			InputTokens:  stats.InputTokens,
			OutputTokens: stats.OutputTokens,
			TotalCost:    stats.TotalCost,
			RequestCount: 1,
		}
	}
	
	usage.Monthly[monthKey] = monthly
}