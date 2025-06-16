package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yildizm/LogSum/internal/monitor"
)

func main() {
	// Create a monitor with default configuration
	collector := monitor.New()

	// Start monitoring
	ctx := context.Background()
	if err := collector.Start(ctx); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}
	defer func() {
		if err := collector.Stop(); err != nil {
			log.Printf("Error stopping collector: %v", err)
		}
	}()

	fmt.Println("🚀 LogSum Performance Monitor Demo")
	fmt.Println("===================================")

	// Simulate some log processing work
	simulateLogProcessing(collector)

	// Let the monitor collect metrics for a few seconds
	fmt.Println("\n📊 Collecting performance metrics...")
	time.Sleep(3 * time.Second)

	// Get a current snapshot
	snapshot := collector.GetSnapshot()
	fmt.Printf("\n📈 Current Performance Snapshot:\n")
	fmt.Printf("Memory Usage: %d bytes\n", snapshot.Memory.CurrentAlloc)
	fmt.Printf("Goroutines: %d\n", snapshot.CPU.NumGoroutines)
	fmt.Printf("Total Lines Processed: %d\n", snapshot.Processing.TotalLinesProcessed)
	fmt.Printf("Processing Rate: %.2f lines/sec\n", snapshot.Processing.LinesPerSecond)

	if len(snapshot.Operations) > 0 {
		fmt.Printf("\n⚡ Operation Performance:\n")
		for _, op := range snapshot.Operations {
			if op.Count > 0 {
				avgTime := time.Duration(op.TotalTime / op.Count)
				fmt.Printf("- %s: %d operations, avg %v\n",
					op.Operation, op.Count, avgTime)
			}
		}
	}

	// Generate a performance report
	fmt.Printf("\n📋 Generating Performance Report...\n")
	store := collector.GetStore()
	generator := monitor.NewReportGenerator(store)

	options := monitor.DefaultReportOptions()
	options.StartTime = time.Now().Add(-5 * time.Minute)
	options.EndTime = time.Now()
	options.Format = monitor.ReportFormatMarkdown

	report, err := generator.GenerateReport(&options)
	if err != nil {
		log.Printf("Failed to generate report: %v", err)
	} else {
		// Format as markdown
		formatted, err := generator.FormatReport(report, monitor.ReportFormatMarkdown)
		if err != nil {
			log.Printf("Failed to format report: %v", err)
		} else {
			fmt.Printf("\n%s\n", formatted)
		}
	}

	fmt.Println("✅ Demo completed!")
}

func simulateLogProcessing(collector *monitor.MetricsCollector) {
	fmt.Println("\n🔄 Simulating log processing operations...")

	// Simulate parsing operations
	for i := 0; i < 5; i++ {
		err := collector.TrackOperation(monitor.OperationParse, func() {
			// Simulate parsing work
			time.Sleep(10 * time.Millisecond)
			collector.RecordLines(100)
			collector.RecordBytes(2048)
		})
		if err != nil {
			log.Printf("Error tracking parse operation: %v", err)
		}
	}

	// Simulate analysis operations
	for i := 0; i < 3; i++ {
		err := collector.TrackOperation(monitor.OperationAnalyze, func() {
			// Simulate analysis work
			time.Sleep(25 * time.Millisecond)
		})
		if err != nil {
			log.Printf("Error tracking analyze operation: %v", err)
		}
	}

	// Simulate AI operations (slower)
	err := collector.TrackOperation(monitor.OperationAI, func() {
		// Simulate AI processing
		time.Sleep(100 * time.Millisecond)
	})
	if err != nil {
		log.Printf("Error tracking AI operation: %v", err)
	}

	// Record some custom metrics
	customMetric := monitor.Metric{
		Name:  "custom.processing_quality",
		Type:  monitor.MetricTypeGauge,
		Value: 0.95, // 95% quality score
		Labels: map[string]string{
			"component": "parser",
			"version":   "1.0",
		},
	}
	if err := collector.RecordMetric(&customMetric); err != nil {
		log.Printf("Error recording custom metric: %v", err)
	}

	fmt.Println("✓ Simulated processing complete")
}
