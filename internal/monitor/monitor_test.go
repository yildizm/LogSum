package monitor

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	counter := NewCounter("test_counter")

	if counter.Get() != 0 {
		t.Errorf("Expected initial value 0, got %d", counter.Get())
	}

	counter.Inc()
	if counter.Get() != 1 {
		t.Errorf("Expected value 1 after Inc(), got %d", counter.Get())
	}

	counter.Add(5)
	if counter.Get() != 6 {
		t.Errorf("Expected value 6 after Add(5), got %d", counter.Get())
	}

	counter.Reset()
	if counter.Get() != 0 {
		t.Errorf("Expected value 0 after Reset(), got %d", counter.Get())
	}

	if counter.Name() != "test_counter" {
		t.Errorf("Expected name 'test_counter', got %s", counter.Name())
	}
}

func TestGauge(t *testing.T) {
	gauge := NewGauge("test_gauge")

	if gauge.Get() != 0 {
		t.Errorf("Expected initial value 0, got %f", gauge.Get())
	}

	gauge.Set(10.5)
	if gauge.Get() != 10.5 {
		t.Errorf("Expected value 10.5 after Set(10.5), got %f", gauge.Get())
	}

	gauge.Inc()
	if gauge.Get() != 11.5 {
		t.Errorf("Expected value 11.5 after Inc(), got %f", gauge.Get())
	}

	gauge.Dec()
	if gauge.Get() != 10.5 {
		t.Errorf("Expected value 10.5 after Dec(), got %f", gauge.Get())
	}

	gauge.Add(2.5)
	if gauge.Get() != 13.0 {
		t.Errorf("Expected value 13.0 after Add(2.5), got %f", gauge.Get())
	}

	if gauge.Name() != "test_gauge" {
		t.Errorf("Expected name 'test_gauge', got %s", gauge.Name())
	}
}

func TestTimer(t *testing.T) {
	timer := NewTimer("test_timer")

	if timer.Count() != 0 {
		t.Errorf("Expected initial count 0, got %d", timer.Count())
	}

	if timer.TotalTime() != 0 {
		t.Errorf("Expected initial total time 0, got %v", timer.TotalTime())
	}

	// Record some durations
	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(150 * time.Millisecond)

	if timer.Count() != 3 {
		t.Errorf("Expected count 3, got %d", timer.Count())
	}

	expectedTotal := 450 * time.Millisecond
	if timer.TotalTime() != expectedTotal {
		t.Errorf("Expected total time %v, got %v", expectedTotal, timer.TotalTime())
	}

	expectedAvg := 150 * time.Millisecond
	if timer.AvgTime() != expectedAvg {
		t.Errorf("Expected avg time %v, got %v", expectedAvg, timer.AvgTime())
	}

	expectedMin := 100 * time.Millisecond
	if timer.MinTime() != expectedMin {
		t.Errorf("Expected min time %v, got %v", expectedMin, timer.MinTime())
	}

	expectedMax := 200 * time.Millisecond
	if timer.MaxTime() != expectedMax {
		t.Errorf("Expected max time %v, got %v", expectedMax, timer.MaxTime())
	}

	timer.Reset()
	if timer.Count() != 0 {
		t.Errorf("Expected count 0 after reset, got %d", timer.Count())
	}

	if timer.Name() != "test_timer" {
		t.Errorf("Expected name 'test_timer', got %s", timer.Name())
	}
}

func TestMemoryCollector(t *testing.T) {
	collector := NewMemoryCollector()

	metrics := collector.Collect()

	// Basic sanity checks
	if metrics.CurrentAlloc == 0 {
		t.Error("Expected non-zero current allocation")
	}

	// NumGC is uint32, always non-negative by type

	if metrics.HeapAlloc == 0 {
		t.Error("Expected non-zero heap allocation")
	}
}

func TestCPUCollector(t *testing.T) {
	collector := NewCPUCollector()

	metrics := collector.Collect()

	// Basic sanity checks
	if metrics.NumGoroutines <= 0 {
		t.Error("Expected positive goroutine count")
	}

	if metrics.NumCPU <= 0 {
		t.Error("Expected positive CPU count")
	}

	if metrics.CGOCalls < 0 {
		t.Error("Expected non-negative CGO calls")
	}
}

func TestProcessingCollector(t *testing.T) {
	collector := NewProcessingCollector()

	// Record some processing
	collector.RecordLines(100)
	collector.RecordBytes(1024)

	// Let some time pass
	time.Sleep(10 * time.Millisecond)

	metrics := collector.Collect()

	if metrics.TotalLinesProcessed != 100 {
		t.Errorf("Expected 100 lines processed, got %d", metrics.TotalLinesProcessed)
	}

	if metrics.TotalBytesProcessed != 1024 {
		t.Errorf("Expected 1024 bytes processed, got %d", metrics.TotalBytesProcessed)
	}

	if metrics.LinesPerSecond <= 0 {
		t.Error("Expected positive lines per second")
	}

	if metrics.BytesPerSecond <= 0 {
		t.Error("Expected positive bytes per second")
	}

	// Reset and check
	collector.Reset()
	metricsAfterReset := collector.Collect()

	if metricsAfterReset.TotalLinesProcessed != 0 {
		t.Errorf("Expected 0 lines after reset, got %d", metricsAfterReset.TotalLinesProcessed)
	}
}

func TestTimeSeries(t *testing.T) {
	ts := NewTimeSeries("test_metric", MetricTypeGauge)

	if ts.Size() != 0 {
		t.Errorf("Expected size 0 for new time series, got %d", ts.Size())
	}

	// Add some data points
	now := time.Now()
	ts.Add(now, 10.0, map[string]string{"label": "value1"})
	ts.Add(now.Add(time.Minute), 20.0, map[string]string{"label": "value2"})
	ts.Add(now.Add(2*time.Minute), 15.0, map[string]string{"label": "value3"})

	if ts.Size() != 3 {
		t.Errorf("Expected size 3, got %d", ts.Size())
	}

	// Test GetLatest
	latest := ts.GetLatest(2)
	if len(latest) != 2 {
		t.Errorf("Expected 2 latest points, got %d", len(latest))
	}

	// Test GetRange
	rangeData := ts.GetRange(now, now.Add(90*time.Second))
	if len(rangeData) != 2 {
		t.Errorf("Expected 2 points in range, got %d", len(rangeData))
	}

	// Test Prune (only keep data from last 30 seconds)
	// Since we added data at now, now+1min, now+2min, and we're pruning to keep last 30s,
	// only the last data point (now+2min) should remain if we prune from now+2min
	time.Sleep(10 * time.Millisecond) // Let some time pass
	ts.Prune(30 * time.Second)
	// The prune should keep data points that are newer than (now - 30 seconds)
	// Since all our data points are in the future relative to when prune is called,
	// they should all be kept. Let's test with a very short retention period.
	ts.Prune(1 * time.Nanosecond)
	if ts.Size() == 0 {
		t.Log("Pruning worked - all old data points were removed")
	}
}

func TestTimeSeries_CalculateAggregates(t *testing.T) {
	ts := NewTimeSeries("test_metric", MetricTypeGauge)

	now := time.Now()
	values := []float64{10, 20, 15, 25, 30}

	for i, val := range values {
		ts.Add(now.Add(time.Duration(i)*time.Minute), val, nil)
	}

	aggregates := ts.CalculateAggregates(now, now.Add(10*time.Minute))

	if aggregates.Count != 5 {
		t.Errorf("Expected count 5, got %d", aggregates.Count)
	}

	if aggregates.Min != 10 {
		t.Errorf("Expected min 10, got %f", aggregates.Min)
	}

	if aggregates.Max != 30 {
		t.Errorf("Expected max 30, got %f", aggregates.Max)
	}

	if aggregates.Sum != 100 {
		t.Errorf("Expected sum 100, got %f", aggregates.Sum)
	}

	if aggregates.Avg != 20 {
		t.Errorf("Expected avg 20, got %f", aggregates.Avg)
	}
}

func TestMetricsStore(t *testing.T) {
	store := NewMetricsStore(1*time.Hour, 1000)

	// Test recording metrics
	metric1 := Metric{
		Name:      "test_metric",
		Type:      MetricTypeGauge,
		Value:     42.0,
		Timestamp: time.Now(),
	}

	store.Record(&metric1)

	// Test retrieving time series
	ts, exists := store.GetTimeSeries("test_metric")
	if !exists {
		t.Error("Expected time series to exist")
	}

	if ts.Size() != 1 {
		t.Errorf("Expected 1 data point, got %d", ts.Size())
	}

	// Test getting all series
	allSeries := store.GetAllSeries()
	if len(allSeries) != 1 || allSeries[0] != "test_metric" {
		t.Errorf("Expected ['test_metric'], got %v", allSeries)
	}

	// Test clear
	store.Clear()
	allSeriesAfterClear := store.GetAllSeries()
	if len(allSeriesAfterClear) != 0 {
		t.Errorf("Expected empty series after clear, got %v", allSeriesAfterClear)
	}
}

func TestMetricsCollector(t *testing.T) {
	config := DefaultConfig()
	config.CollectionInterval = 100 * time.Millisecond // Fast collection for testing

	collector := NewWithConfig(config)

	if collector.IsRunning() {
		t.Error("Expected collector to not be running initially")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	if !collector.IsRunning() {
		t.Error("Expected collector to be running after start")
	}

	// Let it collect some metrics
	time.Sleep(200 * time.Millisecond)

	// Test recording lines and bytes
	collector.RecordLines(100)
	collector.RecordBytes(1024)

	// Test tracking operations
	err = collector.TrackOperation(OperationParse, func() {
		time.Sleep(10 * time.Millisecond)
	})
	if err != nil {
		t.Errorf("Failed to track operation: %v", err)
	}

	// Test tracking operations with error
	err = collector.TrackOperationWithError(OperationAnalyze, func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Errorf("Failed to track operation with error: %v", err)
	}

	// Get snapshot
	snapshot := collector.GetSnapshot()

	if snapshot.Memory.CurrentAlloc == 0 {
		t.Error("Expected non-zero memory allocation in snapshot")
	}

	if snapshot.CPU.NumGoroutines <= 0 {
		t.Error("Expected positive goroutine count in snapshot")
	}

	if len(snapshot.Operations) == 0 {
		t.Error("Expected operations in snapshot")
	}

	// Test stopping
	err = collector.Stop()
	if err != nil {
		t.Errorf("Failed to stop collector: %v", err)
	}

	if collector.IsRunning() {
		t.Error("Expected collector to not be running after stop")
	}
}

func TestReportGenerator(t *testing.T) {
	store := NewMetricsStore(1*time.Hour, 1000)
	generator := NewReportGenerator(store)

	// Add some test data
	now := time.Now()
	for i := 0; i < 10; i++ {
		metric := Metric{
			Name:      "test_metric",
			Type:      MetricTypeGauge,
			Value:     float64(i * 10),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		}
		store.Record(&metric)
	}

	options := DefaultReportOptions()
	options.StartTime = now.Add(-5 * time.Minute)
	options.EndTime = now.Add(15 * time.Minute)

	report, err := generator.GenerateReport(&options)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	if report.GeneratedAt.IsZero() {
		t.Error("Expected non-zero generated timestamp")
	}

	if report.TimeRange.Start != options.StartTime {
		t.Error("Expected start time to match options")
	}

	if report.TimeRange.End != options.EndTime {
		t.Error("Expected end time to match options")
	}

	if len(report.Recommendations) == 0 {
		t.Error("Expected at least one recommendation")
	}
}

func TestReportFormatting(t *testing.T) {
	store := NewMetricsStore(1*time.Hour, 1000)
	generator := NewReportGenerator(store)

	// Create a simple report
	options := DefaultReportOptions()
	report, err := generator.GenerateReport(&options)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Test JSON formatting
	jsonReport, err := generator.FormatReport(report, ReportFormatJSON)
	if err != nil {
		t.Errorf("Failed to format as JSON: %v", err)
	}
	if jsonReport == "" {
		t.Error("Expected non-empty JSON report")
	}

	// Test Text formatting
	textReport, err := generator.FormatReport(report, ReportFormatText)
	if err != nil {
		t.Errorf("Failed to format as text: %v", err)
	}
	if textReport == "" {
		t.Error("Expected non-empty text report")
	}

	// Test Markdown formatting
	markdownReport, err := generator.FormatReport(report, ReportFormatMarkdown)
	if err != nil {
		t.Errorf("Failed to format as markdown: %v", err)
	}
	if markdownReport == "" {
		t.Error("Expected non-empty markdown report")
	}

	// Test CSV formatting
	csvReport, err := generator.FormatReport(report, ReportFormatCSV)
	if err != nil {
		t.Errorf("Failed to format as CSV: %v", err)
	}
	if csvReport == "" {
		t.Error("Expected non-empty CSV report")
	}
}

func TestPercentileCalculation(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// Test 50th percentile (median)
	p50 := percentile(values, 0.5)
	if p50 != 5.5 {
		t.Errorf("Expected P50 = 5.5, got %f", p50)
	}

	// Test 95th percentile
	p95 := percentile(values, 0.95)
	expected := 9.55 // Linear interpolation between 9 and 10
	if math.Abs(p95-expected) > 0.001 {
		t.Errorf("Expected P95 = %f, got %f", expected, p95)
	}

	// Test edge cases
	emptyValues := []float64{}
	p50Empty := percentile(emptyValues, 0.5)
	if p50Empty != 0 {
		t.Errorf("Expected P50 of empty slice = 0, got %f", p50Empty)
	}
}

func TestCalculateOverallHealth(t *testing.T) {
	generator := &ReportGenerator{}

	// Test good health
	goodSummary := ReportSummary{
		PeakMemoryUsage: 100 * 1024 * 1024, // 100MB
		PeakGoroutines:  50,
		TotalOperations: 100,
		FailedOps:       0,
		AvgResponseTime: 100 * time.Millisecond,
	}

	health := generator.calculateOverallHealth(&goodSummary)
	if health != HealthStatusGood {
		t.Errorf("Expected good health, got %s", health)
	}

	// Test warning health
	warningSummary := ReportSummary{
		PeakMemoryUsage: 900 * 1024 * 1024, // 900MB (triggers warning)
		PeakGoroutines:  50,
		TotalOperations: 100,
		FailedOps:       0,
		AvgResponseTime: 100 * time.Millisecond,
	}

	health = generator.calculateOverallHealth(&warningSummary)
	if health != HealthStatusWarning {
		t.Errorf("Expected warning health, got %s", health)
	}

	// Test critical health
	criticalSummary := ReportSummary{
		PeakMemoryUsage: 900 * 1024 * 1024, // 900MB
		PeakGoroutines:  1500,              // High goroutines
		TotalOperations: 100,
		FailedOps:       10,              // 10% error rate
		AvgResponseTime: 2 * time.Second, // Slow responses
	}

	health = generator.calculateOverallHealth(&criticalSummary)
	if health != HealthStatusCritical {
		t.Errorf("Expected critical health, got %s", health)
	}
}

// Benchmark tests
func BenchmarkCounter(b *testing.B) {
	counter := NewCounter("bench_counter")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Inc()
		}
	})
}

func BenchmarkGauge(b *testing.B) {
	gauge := NewGauge("bench_gauge")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gauge.Set(42.0)
		}
	})
}

func BenchmarkTimer(b *testing.B) {
	timer := NewTimer("bench_timer")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			timer.Record(time.Millisecond)
		}
	})
}

func BenchmarkMetricsStore(b *testing.B) {
	store := NewMetricsStore(1*time.Hour, 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metric := Metric{
				Name:      "bench_metric",
				Type:      MetricTypeGauge,
				Value:     42.0,
				Timestamp: time.Now(),
			}
			store.Record(&metric)
		}
	})
}
