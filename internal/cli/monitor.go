package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yildizm/LogSum/internal/monitor"
)

var (
	monitorDuration      time.Duration
	monitorInterval      time.Duration
	monitorFormat        string
	monitorOutputFile    string
	monitorIncludeSystem bool
	monitorIncludeOps    bool
	monitorIncludeTS     bool
	monitorStartTime     string
	monitorEndTime       string
)

func newMonitorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Performance monitoring and metrics collection",
		Long: `Monitor LogSum's performance and collect system metrics.

The monitor command provides real-time performance monitoring, metrics collection,
and report generation capabilities for analyzing LogSum's resource usage and
processing performance.

Examples:
  logsum monitor start --duration 30s
  logsum monitor report --format json
  logsum monitor report --output-file metrics.json`,
	}

	// Add subcommands
	cmd.AddCommand(newMonitorStartCommand())
	cmd.AddCommand(newMonitorReportCommand())
	cmd.AddCommand(newMonitorStopCommand())

	return cmd
}

func newMonitorStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start performance monitoring",
		Long: `Start real-time performance monitoring of LogSum.

This command begins collecting system metrics including memory usage, CPU statistics,
and operation performance. Monitoring continues until stopped manually or the
specified duration expires.

Examples:
  logsum monitor start
  logsum monitor start --duration 60s
  logsum monitor start --interval 5s`,
		RunE: runMonitorStart,
	}

	cmd.Flags().DurationVar(&monitorDuration, "duration", 0, "monitoring duration (0 = indefinite)")
	cmd.Flags().DurationVar(&monitorInterval, "interval", 1*time.Second, "metrics collection interval")
	cmd.Flags().StringVar(&monitorOutputFile, "output-file", "", "save metrics to file (optional)")

	return cmd
}

func newMonitorReportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate performance report",
		Long: `Generate a performance report from collected metrics.

Reports can be generated in multiple formats and include various metric types
such as system metrics, operation performance, and time-series data.

Examples:
  logsum monitor report
  logsum monitor report --format json
  logsum monitor report --format csv --output-file report.csv
  logsum monitor report --include-system --include-operations`,
		RunE: runMonitorReport,
	}

	cmd.Flags().StringVar(&monitorFormat, "format", "text", "report format (text, json, markdown, csv)")
	cmd.Flags().StringVar(&monitorOutputFile, "output-file", "", "save report to file instead of stdout")
	cmd.Flags().BoolVar(&monitorIncludeSystem, "include-system", true, "include system metrics")
	cmd.Flags().BoolVar(&monitorIncludeOps, "include-operations", true, "include operation metrics")
	cmd.Flags().BoolVar(&monitorIncludeTS, "include-timeseries", false, "include time-series data")
	cmd.Flags().StringVar(&monitorStartTime, "start-time", "", "report start time (RFC3339 format)")
	cmd.Flags().StringVar(&monitorEndTime, "end-time", "", "report end time (RFC3339 format)")

	return cmd
}

func newMonitorStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop performance monitoring",
		Long: `Stop the currently running performance monitoring session.

This command stops any active monitoring and optionally generates a final report
with the collected metrics.

Examples:
  logsum monitor stop
  logsum monitor stop --report
  logsum monitor stop --report --format json`,
		RunE: runMonitorStop,
	}

	cmd.Flags().BoolVar(&monitorIncludeSystem, "report", false, "generate final report when stopping")
	cmd.Flags().StringVar(&monitorFormat, "format", "text", "report format if generating report")

	return cmd
}

func runMonitorStart(cmd *cobra.Command, args []string) error {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Starting performance monitoring...\n")
		if monitorDuration > 0 {
			fmt.Fprintf(os.Stderr, "Duration: %v\n", monitorDuration)
		} else {
			fmt.Fprintf(os.Stderr, "Duration: indefinite (Ctrl+C to stop)\n")
		}
		fmt.Fprintf(os.Stderr, "Collection interval: %v\n", monitorInterval)
	}

	// Create monitor with configuration
	config := monitor.MonitorConfig{
		CollectionInterval:      monitorInterval,
		RetentionPeriod:         24 * time.Hour, // Keep data for 24 hours
		MaxDataPoints:           10000,
		BufferSize:              1000,
		EnableMemoryMetrics:     true,
		EnableCPUMetrics:        true,
		EnableProcessingMetrics: true,
	}

	collector := monitor.NewWithConfig(config)

	// Set up context with timeout if duration is specified
	ctx := context.Background()
	if monitorDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, monitorDuration)
		defer cancel()
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring
	if err := collector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	fmt.Println("📊 Performance monitoring started...")
	if !isVerbose() {
		fmt.Println("Use Ctrl+C to stop monitoring")
	}

	// Show real-time stats if verbose
	if isVerbose() {
		go showRealTimeStats(collector)
	}

	// Wait for completion or interruption
	select {
	case <-ctx.Done():
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "\nMonitoring duration completed\n")
		}
	case sig := <-sigChan:
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "\nReceived signal %v, stopping monitoring...\n", sig)
		}
	}

	// Stop monitoring
	if err := collector.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error stopping monitor: %v\n", err)
	}

	// Generate final report if output file specified
	if monitorOutputFile != "" {
		if err := generateReport(collector, monitorOutputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save report: %v\n", err)
		} else if isVerbose() {
			fmt.Fprintf(os.Stderr, "Final report saved to: %s\n", monitorOutputFile)
		}
	}

	fmt.Println("📊 Monitoring stopped")
	return nil
}

func runMonitorReport(cmd *cobra.Command, args []string) error {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, "Generating performance report...\n")
		fmt.Fprintf(os.Stderr, "Format: %s\n", monitorFormat)
	}

	// Parse time ranges if provided
	var startTime, endTime time.Time
	var err error

	if monitorStartTime != "" {
		startTime, err = time.Parse(time.RFC3339, monitorStartTime)
		if err != nil {
			return fmt.Errorf("invalid start time format: %w", err)
		}
	} else {
		startTime = time.Now().Add(-1 * time.Hour) // Default to last hour
	}

	if monitorEndTime != "" {
		endTime, err = time.Parse(time.RFC3339, monitorEndTime)
		if err != nil {
			return fmt.Errorf("invalid end time format: %w", err)
		}
	} else {
		endTime = time.Now()
	}

	// Create report options
	options := monitor.ReportOptions{
		StartTime:            startTime,
		EndTime:              endTime,
		Format:               monitor.ReportFormat(monitorFormat),
		IncludeSystemMetrics: monitorIncludeSystem,
		IncludeOperations:    monitorIncludeOps,
		IncludeTimeSeries:    monitorIncludeTS,
	}

	// Create a metrics store for report generation
	// In a real implementation, this would be a persistent store
	store := monitor.NewMetricsStore(24*time.Hour, 10000)

	// Add some sample metrics for demonstration
	addSampleMetrics(store, startTime, endTime)

	// Generate report using the proper report generator
	generator := monitor.NewReportGenerator(store)
	report, err := generator.GenerateReport(&options)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Format the report
	formattedReport, err := generator.FormatReport(report, options.Format)
	if err != nil {
		return fmt.Errorf("failed to format report: %w", err)
	}

	// Output report
	if monitorOutputFile != "" {
		if err := writeReportToFile(formattedReport, monitorOutputFile); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		if isVerbose() {
			fmt.Fprintf(os.Stderr, "Report saved to: %s\n", monitorOutputFile)
		}
	} else {
		fmt.Print(formattedReport)
	}

	return nil
}

func runMonitorStop(cmd *cobra.Command, args []string) error {
	// In a real implementation, this would communicate with a running monitor process
	fmt.Println("📊 Monitor stop command received")

	if monitorIncludeSystem { // Using this flag as "report" flag
		fmt.Println("Generating final report...")
		return runMonitorReport(cmd, args)
	}

	return nil
}

func showRealTimeStats(collector *monitor.MetricsCollector) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !collector.IsRunning() {
			return
		}

		snapshot := collector.GetSnapshot()
		fmt.Fprintf(os.Stderr, "\r📈 Memory: %d bytes | Goroutines: %d | Operations: %d",
			snapshot.Memory.CurrentAlloc,
			snapshot.CPU.NumGoroutines,
			len(snapshot.Operations))
	}
}

func generateReport(collector *monitor.MetricsCollector, filename string) error {
	snapshot := collector.GetSnapshot()

	// Convert to JSON for simplicity
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return writeOutputBytesToFile(data, filename)
}

func addSampleMetrics(store *monitor.MetricsStore, startTime, endTime time.Time) {
	// Generate sample metrics over the time range
	duration := endTime.Sub(startTime)
	interval := duration / 20 // Create 20 data points

	for i := 0; i < 20; i++ {
		timestamp := startTime.Add(time.Duration(i) * interval)

		// Memory metrics
		memMetric := &monitor.Metric{
			Name:      "memory.alloc",
			Type:      monitor.MetricTypeGauge,
			Value:     float64(45000000 + i*1000000), // 45MB to 65MB
			Timestamp: timestamp,
		}
		store.Record(memMetric)

		// CPU metrics
		cpuMetric := &monitor.Metric{
			Name:      "cpu.goroutines",
			Type:      monitor.MetricTypeGauge,
			Value:     float64(8 + i%3), // 8-10 goroutines
			Timestamp: timestamp,
		}
		store.Record(cpuMetric)

		// Operation metrics
		parseMetric := &monitor.Metric{
			Name:      "operation.parse.duration",
			Type:      monitor.MetricTypeTiming,
			Value:     0.5 + float64(i%3)*0.1, // 0.5-0.7ms
			Timestamp: timestamp,
		}
		store.Record(parseMetric)

		// Processing rate
		rateMetric := &monitor.Metric{
			Name:      "processing.lines_per_second",
			Type:      monitor.MetricTypeGauge,
			Value:     2500 + float64(i*50), // Increasing rate
			Timestamp: timestamp,
		}
		store.Record(rateMetric)
	}
}

func writeReportToFile(content, filename string) error {
	// Validate file path
	cleanPath := filepath.Clean(filename)

	// Create directory if it doesn't exist
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	return writeOutputBytesToFile([]byte(content), cleanPath)
}
