package monitor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ReportOptions configures report generation
type ReportOptions struct {
	// TimeRange for the report
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// Format options
	Format ReportFormat `json:"format"`

	// Include options
	IncludeSystemMetrics bool `json:"include_system_metrics"`
	IncludeOperations    bool `json:"include_operations"`
	IncludeTimeSeries    bool `json:"include_time_series"`
	IncludeAggregates    bool `json:"include_aggregates"`

	// Filtering options
	MetricFilter []string `json:"metric_filter"` // Only include these metrics

	// Aggregation options
	AggregationWindow time.Duration `json:"aggregation_window"`
}

// ReportFormat represents the output format for reports
type ReportFormat string

const (
	ReportFormatJSON     ReportFormat = "json"
	ReportFormatText     ReportFormat = "text"
	ReportFormatMarkdown ReportFormat = "markdown"
	ReportFormatCSV      ReportFormat = "csv"
)

// DefaultReportOptions returns default report options
func DefaultReportOptions() ReportOptions {
	return ReportOptions{
		StartTime:            time.Now().Add(-1 * time.Hour),
		EndTime:              time.Now(),
		Format:               ReportFormatJSON,
		IncludeSystemMetrics: true,
		IncludeOperations:    true,
		IncludeTimeSeries:    false,
		IncludeAggregates:    true,
		AggregationWindow:    5 * time.Minute,
	}
}

// PerformanceReport contains comprehensive performance information
type PerformanceReport struct {
	GeneratedAt     time.Time                        `json:"generated_at"`
	TimeRange       TimeRange                        `json:"time_range"`
	Summary         ReportSummary                    `json:"summary"`
	SystemMetrics   *SystemMetricsReport             `json:"system_metrics,omitempty"`
	Operations      []OperationReport                `json:"operations,omitempty"`
	TimeSeries      map[string][]TimeSeriesDataPoint `json:"time_series,omitempty"`
	Aggregates      map[string]Aggregates            `json:"aggregates,omitempty"`
	Recommendations []string                         `json:"recommendations,omitempty"`
}

// TimeRange represents a time range for the report
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ReportSummary provides a high-level summary of performance
type ReportSummary struct {
	Duration        time.Duration `json:"duration"`
	TotalOperations int64         `json:"total_operations"`
	SuccessfulOps   int64         `json:"successful_operations"`
	FailedOps       int64         `json:"failed_operations"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	PeakMemoryUsage uint64        `json:"peak_memory_usage_bytes"`
	PeakGoroutines  int           `json:"peak_goroutines"`
	ProcessingRate  float64       `json:"processing_rate_lines_per_sec"`
	OverallHealth   HealthStatus  `json:"overall_health"`
}

// HealthStatus represents the overall system health
type HealthStatus string

const (
	HealthStatusGood     HealthStatus = "good"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
)

// SystemMetricsReport contains system-level metrics
type SystemMetricsReport struct {
	Memory     MemoryReport     `json:"memory"`
	CPU        CPUReport        `json:"cpu"`
	Processing ProcessingReport `json:"processing"`
}

// MemoryReport contains memory usage analysis
type MemoryReport struct {
	Current    MemoryMetrics `json:"current"`
	Peak       MemoryMetrics `json:"peak"`
	Trend      string        `json:"trend"`
	GCPressure string        `json:"gc_pressure"`
}

// CPUReport contains CPU usage analysis
type CPUReport struct {
	Current        CPUMetrics `json:"current"`
	Peak           CPUMetrics `json:"peak"`
	GoroutineTrend string     `json:"goroutine_trend"`
	ResourceUsage  string     `json:"resource_usage"`
}

// ProcessingReport contains processing performance analysis
type ProcessingReport struct {
	Current    ProcessingMetrics  `json:"current"`
	Throughput ThroughputAnalysis `json:"throughput"`
	Efficiency string             `json:"efficiency"`
}

// ThroughputAnalysis provides throughput metrics analysis
type ThroughputAnalysis struct {
	PeakLinesPerSecond  float64 `json:"peak_lines_per_second"`
	PeakBytesPerSecond  float64 `json:"peak_bytes_per_second"`
	AvgLinesPerSecond   float64 `json:"avg_lines_per_second"`
	AvgBytesPerSecond   float64 `json:"avg_bytes_per_second"`
	ThroughputStability string  `json:"throughput_stability"`
}

// OperationReport contains operation-specific performance data
type OperationReport struct {
	Operation   OperationType    `json:"operation"`
	Metrics     OperationMetrics `json:"metrics"`
	Performance string           `json:"performance_assessment"`
	Bottlenecks []string         `json:"bottlenecks"`
	Trends      OperationTrends  `json:"trends"`
}

// OperationTrends contains trend analysis for operations
type OperationTrends struct {
	LatencyTrend    string `json:"latency_trend"`
	ThroughputTrend string `json:"throughput_trend"`
	ErrorRateTrend  string `json:"error_rate_trend"`
}

// ReportGenerator generates performance reports from metrics data
type ReportGenerator struct {
	store *MetricsStore
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(store *MetricsStore) *ReportGenerator {
	return &ReportGenerator{
		store: store,
	}
}

// GenerateReport generates a performance report based on the given options
func (rg *ReportGenerator) GenerateReport(options *ReportOptions) (*PerformanceReport, error) {
	report := &PerformanceReport{
		GeneratedAt: time.Now(),
		TimeRange: TimeRange{
			Start: options.StartTime,
			End:   options.EndTime,
		},
	}

	// Generate summary
	summary, err := rg.generateSummary(options)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}
	report.Summary = summary

	// Generate system metrics report
	if options.IncludeSystemMetrics {
		systemMetrics, err := rg.generateSystemMetricsReport(options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate system metrics: %w", err)
		}
		report.SystemMetrics = systemMetrics
	}

	// Generate operations report
	if options.IncludeOperations {
		operations, err := rg.generateOperationsReport(options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate operations report: %w", err)
		}
		report.Operations = operations
	}

	// Include time series data
	if options.IncludeTimeSeries {
		timeSeries := rg.store.GetMetricsInRange(options.StartTime, options.EndTime)
		report.TimeSeries = rg.filterTimeSeries(timeSeries, options.MetricFilter)
	}

	// Include aggregates
	if options.IncludeAggregates {
		aggregates, err := rg.generateAggregates(options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate aggregates: %w", err)
		}
		report.Aggregates = aggregates
	}

	// Generate recommendations
	recommendations := rg.generateRecommendations(report)
	report.Recommendations = recommendations

	return report, nil
}

// generateSummary generates a high-level summary of performance
func (rg *ReportGenerator) generateSummary(options *ReportOptions) (ReportSummary, error) {
	summary := ReportSummary{
		Duration: options.EndTime.Sub(options.StartTime),
	}

	// Get operation metrics
	opMetrics := rg.store.GetMetricsInRange(options.StartTime, options.EndTime)

	// Calculate operation counts
	if opData, exists := opMetrics["operation.duration"]; exists {
		summary.TotalOperations = int64(len(opData))
		// Would need error tracking to calculate success/failure rates
		summary.SuccessfulOps = summary.TotalOperations
		summary.FailedOps = 0

		// Calculate average response time
		if len(opData) > 0 {
			totalTime := int64(0)
			for _, dp := range opData {
				totalTime += int64(dp.Value)
			}
			summary.AvgResponseTime = time.Duration(totalTime / int64(len(opData)))
		}
	}

	// Get peak memory usage
	if memData, exists := opMetrics["memory.current_alloc"]; exists {
		peak := uint64(0)
		for _, dp := range memData {
			if uint64(dp.Value) > peak {
				peak = uint64(dp.Value)
			}
		}
		summary.PeakMemoryUsage = peak
	}

	// Get peak goroutines
	if goroutineData, exists := opMetrics["cpu.num_goroutines"]; exists {
		peak := 0
		for _, dp := range goroutineData {
			if int(dp.Value) > peak {
				peak = int(dp.Value)
			}
		}
		summary.PeakGoroutines = peak
	}

	// Get processing rate
	if rateData, exists := opMetrics["processing.lines_per_second"]; exists && len(rateData) > 0 {
		total := 0.0
		for _, dp := range rateData {
			total += dp.Value
		}
		summary.ProcessingRate = total / float64(len(rateData))
	}

	// Determine overall health
	summary.OverallHealth = rg.calculateOverallHealth(&summary)

	return summary, nil
}

// calculateOverallHealth determines the overall system health based on metrics
func (rg *ReportGenerator) calculateOverallHealth(summary *ReportSummary) HealthStatus {
	// Simple health assessment based on basic metrics
	issues := 0

	// Check memory usage (if > 80% of available, it's a concern)
	if summary.PeakMemoryUsage > 800*1024*1024 { // 800MB threshold
		issues++
	}

	// Check goroutine count (if > 1000, it's a concern)
	if summary.PeakGoroutines > 1000 {
		issues++
	}

	// Check error rate
	if summary.TotalOperations > 0 {
		errorRate := float64(summary.FailedOps) / float64(summary.TotalOperations)
		if errorRate > 0.05 { // 5% error rate
			issues++
		}
	}

	// Check average response time (if > 1s, it's slow)
	if summary.AvgResponseTime > time.Second {
		issues++
	}

	switch {
	case issues >= 3:
		return HealthStatusCritical
	case issues >= 1:
		return HealthStatusWarning
	default:
		return HealthStatusGood
	}
}

// generateSystemMetricsReport generates system metrics analysis
func (rg *ReportGenerator) generateSystemMetricsReport(options *ReportOptions) (*SystemMetricsReport, error) {
	// This would analyze memory, CPU, and processing metrics
	// For brevity, returning a basic structure
	return &SystemMetricsReport{
		Memory: MemoryReport{
			Trend:      "stable",
			GCPressure: "low",
		},
		CPU: CPUReport{
			GoroutineTrend: "stable",
			ResourceUsage:  "normal",
		},
		Processing: ProcessingReport{
			Efficiency: "good",
			Throughput: ThroughputAnalysis{
				ThroughputStability: "stable",
			},
		},
	}, nil
}

// generateOperationsReport generates operation-specific performance reports
func (rg *ReportGenerator) generateOperationsReport(options *ReportOptions) ([]OperationReport, error) {
	operations := []OperationType{
		OperationParse, OperationAnalyze, OperationAI,
		OperationFileIO, OperationPattern, OperationInsight, OperationTimeline,
	}

	reports := make([]OperationReport, 0, len(operations))

	for _, op := range operations {
		report := OperationReport{
			Operation:   op,
			Performance: "good", // Would be calculated based on actual metrics
			Bottlenecks: []string{},
			Trends: OperationTrends{
				LatencyTrend:    "stable",
				ThroughputTrend: "stable",
				ErrorRateTrend:  "stable",
			},
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// generateAggregates generates statistical aggregates for all metrics
func (rg *ReportGenerator) generateAggregates(options *ReportOptions) (map[string]Aggregates, error) {
	aggregates := make(map[string]Aggregates)

	seriesNames := rg.store.GetAllSeries()

	for _, name := range seriesNames {
		if len(options.MetricFilter) > 0 && !contains(options.MetricFilter, name) {
			continue
		}

		agg, exists := rg.store.CalculateAggregates(name, options.StartTime, options.EndTime)
		if exists {
			aggregates[name] = agg
		}
	}

	return aggregates, nil
}

// filterTimeSeries filters time series data based on metric filter
func (rg *ReportGenerator) filterTimeSeries(timeSeries map[string][]TimeSeriesDataPoint, filter []string) map[string][]TimeSeriesDataPoint {
	if len(filter) == 0 {
		return timeSeries
	}

	filtered := make(map[string][]TimeSeriesDataPoint)
	for name, data := range timeSeries {
		if contains(filter, name) {
			filtered[name] = data
		}
	}

	return filtered
}

// generateRecommendations generates performance recommendations based on the report
func (rg *ReportGenerator) generateRecommendations(report *PerformanceReport) []string {
	var recommendations []string

	// Memory recommendations
	if report.Summary.PeakMemoryUsage > 500*1024*1024 { // 500MB
		recommendations = append(recommendations, "Consider optimizing memory usage or increasing available memory")
	}

	// Goroutine recommendations
	if report.Summary.PeakGoroutines > 500 {
		recommendations = append(recommendations, "Monitor goroutine usage to prevent goroutine leaks")
	}

	// Performance recommendations
	if report.Summary.AvgResponseTime > 500*time.Millisecond {
		recommendations = append(recommendations, "Investigate slow operations to improve response times")
	}

	// Processing rate recommendations
	if report.Summary.ProcessingRate < 1000 { // lines per second
		recommendations = append(recommendations, "Consider optimizing log parsing performance")
	}

	// Health-based recommendations
	switch report.Summary.OverallHealth {
	case HealthStatusWarning:
		recommendations = append(recommendations, "System showing warning signs - monitor closely")
	case HealthStatusCritical:
		recommendations = append(recommendations, "System performance is critical - immediate attention required")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System performance is within normal parameters")
	}

	return recommendations
}

// FormatReport formats a report according to the specified format
func (rg *ReportGenerator) FormatReport(report *PerformanceReport, format ReportFormat) (string, error) {
	switch format {
	case ReportFormatJSON:
		return rg.formatJSON(report)
	case ReportFormatText:
		return rg.formatText(report)
	case ReportFormatMarkdown:
		return rg.formatMarkdown(report)
	case ReportFormatCSV:
		return rg.formatCSV(report)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// formatJSON formats the report as JSON
func (rg *ReportGenerator) formatJSON(report *PerformanceReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// formatText formats the report as plain text
func (rg *ReportGenerator) formatText(report *PerformanceReport) (string, error) {
	var sb strings.Builder

	sb.WriteString("LogSum Performance Report\n")
	sb.WriteString("========================\n\n")

	sb.WriteString(fmt.Sprintf("Generated: %s\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Time Range: %s to %s\n",
		report.TimeRange.Start.Format(time.RFC3339),
		report.TimeRange.End.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Duration: %s\n\n", report.Summary.Duration))

	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Total Operations: %d\n", report.Summary.TotalOperations))
	sb.WriteString(fmt.Sprintf("  Successful: %d\n", report.Summary.SuccessfulOps))
	sb.WriteString(fmt.Sprintf("  Failed: %d\n", report.Summary.FailedOps))
	sb.WriteString(fmt.Sprintf("  Avg Response Time: %s\n", report.Summary.AvgResponseTime))
	sb.WriteString(fmt.Sprintf("  Peak Memory: %d bytes\n", report.Summary.PeakMemoryUsage))
	sb.WriteString(fmt.Sprintf("  Peak Goroutines: %d\n", report.Summary.PeakGoroutines))
	sb.WriteString(fmt.Sprintf("  Processing Rate: %.2f lines/sec\n", report.Summary.ProcessingRate))
	sb.WriteString(fmt.Sprintf("  Overall Health: %s\n\n", report.Summary.OverallHealth))

	if len(report.Recommendations) > 0 {
		sb.WriteString("Recommendations:\n")
		for _, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("  - %s\n", rec))
		}
	}

	return sb.String(), nil
}

// formatMarkdown formats the report as Markdown
func (rg *ReportGenerator) formatMarkdown(report *PerformanceReport) (string, error) {
	var sb strings.Builder

	sb.WriteString("# LogSum Performance Report\n\n")

	sb.WriteString(fmt.Sprintf("**Generated:** %s  \n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Time Range:** %s to %s  \n",
		report.TimeRange.Start.Format(time.RFC3339),
		report.TimeRange.End.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n\n", report.Summary.Duration))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Operations | %d |\n", report.Summary.TotalOperations))
	sb.WriteString(fmt.Sprintf("| Successful | %d |\n", report.Summary.SuccessfulOps))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", report.Summary.FailedOps))
	sb.WriteString(fmt.Sprintf("| Avg Response Time | %s |\n", report.Summary.AvgResponseTime))
	sb.WriteString(fmt.Sprintf("| Peak Memory | %d bytes |\n", report.Summary.PeakMemoryUsage))
	sb.WriteString(fmt.Sprintf("| Peak Goroutines | %d |\n", report.Summary.PeakGoroutines))
	sb.WriteString(fmt.Sprintf("| Processing Rate | %.2f lines/sec |\n", report.Summary.ProcessingRate))
	sb.WriteString(fmt.Sprintf("| Overall Health | %s |\n\n", report.Summary.OverallHealth))

	if len(report.Recommendations) > 0 {
		sb.WriteString("## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("- %s\n", rec))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// formatCSV formats the report as CSV
func (rg *ReportGenerator) formatCSV(report *PerformanceReport) (string, error) {
	var sb strings.Builder

	sb.WriteString("metric,value\n")
	sb.WriteString(fmt.Sprintf("generated_at,%s\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("start_time,%s\n", report.TimeRange.Start.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("end_time,%s\n", report.TimeRange.End.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("duration_ns,%d\n", report.Summary.Duration.Nanoseconds()))
	sb.WriteString(fmt.Sprintf("total_operations,%d\n", report.Summary.TotalOperations))
	sb.WriteString(fmt.Sprintf("successful_operations,%d\n", report.Summary.SuccessfulOps))
	sb.WriteString(fmt.Sprintf("failed_operations,%d\n", report.Summary.FailedOps))
	sb.WriteString(fmt.Sprintf("avg_response_time_ns,%d\n", report.Summary.AvgResponseTime.Nanoseconds()))
	sb.WriteString(fmt.Sprintf("peak_memory_bytes,%d\n", report.Summary.PeakMemoryUsage))
	sb.WriteString(fmt.Sprintf("peak_goroutines,%d\n", report.Summary.PeakGoroutines))
	sb.WriteString(fmt.Sprintf("processing_rate_lines_per_sec,%.2f\n", report.Summary.ProcessingRate))
	sb.WriteString(fmt.Sprintf("overall_health,%s\n", report.Summary.OverallHealth))

	return sb.String(), nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
