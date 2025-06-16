package monitor

import (
	"context"
	"sync"
	"time"
)

// Collector interface for collecting metrics
type Collector interface {
	// Start begins metric collection
	Start(ctx context.Context) error

	// Stop stops metric collection
	Stop() error

	// TrackOperation tracks an operation with timing
	TrackOperation(operation OperationType, fn func()) error

	// TrackOperationWithError tracks an operation that may return an error
	TrackOperationWithError(operation OperationType, fn func() error) error

	// RecordMetric records a custom metric
	RecordMetric(metric Metric) error

	// GetSnapshot returns a current metrics snapshot
	GetSnapshot() MetricsSnapshot

	// IsRunning returns true if the collector is actively collecting
	IsRunning() bool
}

// MonitorConfig holds configuration for the monitor
type MonitorConfig struct {
	// CollectionInterval is how often to collect system metrics
	CollectionInterval time.Duration

	// RetentionPeriod is how long to keep metrics data
	RetentionPeriod time.Duration

	// MaxDataPoints is the maximum number of data points per series
	MaxDataPoints int

	// BufferSize is the size of the metric collection buffer
	BufferSize int

	// EnableMemoryMetrics enables memory metric collection
	EnableMemoryMetrics bool

	// EnableCPUMetrics enables CPU metric collection
	EnableCPUMetrics bool

	// EnableProcessingMetrics enables processing metric collection
	EnableProcessingMetrics bool
}

// DefaultConfig returns a default monitor configuration
func DefaultConfig() MonitorConfig {
	return MonitorConfig{
		CollectionInterval:      5 * time.Second,
		RetentionPeriod:         1 * time.Hour,
		MaxDataPoints:           1000,
		BufferSize:              100,
		EnableMemoryMetrics:     true,
		EnableCPUMetrics:        true,
		EnableProcessingMetrics: true,
	}
}

// MetricsCollector implements the Collector interface
type MetricsCollector struct {
	config              MonitorConfig
	store               *MetricsStore
	memoryCollector     *MemoryCollector
	cpuCollector        *CPUCollector
	processingCollector *ProcessingCollector
	operationTimers     map[OperationType]*Timer
	metricBuffer        chan Metric

	// Operation tracking
	operationMutex sync.RWMutex

	// State management
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mutex   sync.RWMutex
}

// New creates a new metrics collector with default configuration
func New() *MetricsCollector {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new metrics collector with custom configuration
func NewWithConfig(config MonitorConfig) *MetricsCollector {
	return &MetricsCollector{
		config:              config,
		store:               NewMetricsStore(config.RetentionPeriod, config.MaxDataPoints),
		memoryCollector:     NewMemoryCollector(),
		cpuCollector:        NewCPUCollector(),
		processingCollector: NewProcessingCollector(),
		operationTimers:     make(map[OperationType]*Timer),
		metricBuffer:        make(chan Metric, config.BufferSize),
	}
}

// Start begins metric collection
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.running {
		return nil // Already running
	}

	mc.ctx, mc.cancel = context.WithCancel(ctx)
	mc.running = true

	// Initialize operation timers
	operations := []OperationType{
		OperationParse, OperationAnalyze, OperationAI,
		OperationFileIO, OperationPattern, OperationInsight, OperationTimeline,
	}

	for _, op := range operations {
		mc.operationTimers[op] = NewTimer(string(op))
	}

	// Start metric collection goroutines
	mc.wg.Add(2)
	go mc.collectSystemMetrics()
	go mc.processMetricBuffer()

	return nil
}

// Stop stops metric collection
func (mc *MetricsCollector) Stop() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if !mc.running {
		return nil // Not running
	}

	mc.running = false
	mc.cancel()

	// Close the metric buffer
	close(mc.metricBuffer)

	// Wait for all goroutines to finish
	mc.wg.Wait()

	return nil
}

// IsRunning returns true if the collector is actively collecting
func (mc *MetricsCollector) IsRunning() bool {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.running
}

// collectSystemMetrics periodically collects system metrics
func (mc *MetricsCollector) collectSystemMetrics() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.collectAndRecord()
		}
	}
}

// collectAndRecord collects all enabled metrics and records them
func (mc *MetricsCollector) collectAndRecord() {
	now := time.Now()

	// Collect memory metrics
	if mc.config.EnableMemoryMetrics {
		memMetrics := mc.memoryCollector.Collect()
		mc.recordMemoryMetrics(&memMetrics, now)
	}

	// Collect CPU metrics
	if mc.config.EnableCPUMetrics {
		cpuMetrics := mc.cpuCollector.Collect()
		mc.recordCPUMetrics(&cpuMetrics, now)
	}

	// Collect processing metrics
	if mc.config.EnableProcessingMetrics {
		procMetrics := mc.processingCollector.Collect()
		mc.recordProcessingMetrics(&procMetrics, now)
	}

	// Prune old data periodically
	mc.store.Prune()
}

// recordMemoryMetrics records memory metrics to the store
func (mc *MetricsCollector) recordMemoryMetrics(memMetrics *MemoryMetrics, timestamp time.Time) {
	metrics := []Metric{
		{Name: "memory.current_alloc", Type: MetricTypeGauge, Value: float64(memMetrics.CurrentAlloc), Timestamp: timestamp},
		{Name: "memory.total_alloc", Type: MetricTypeCounter, Value: float64(memMetrics.TotalAlloc), Timestamp: timestamp},
		{Name: "memory.sys", Type: MetricTypeGauge, Value: float64(memMetrics.Sys), Timestamp: timestamp},
		{Name: "memory.num_gc", Type: MetricTypeCounter, Value: float64(memMetrics.NumGC), Timestamp: timestamp},
		{Name: "memory.heap_alloc", Type: MetricTypeGauge, Value: float64(memMetrics.HeapAlloc), Timestamp: timestamp},
		{Name: "memory.heap_sys", Type: MetricTypeGauge, Value: float64(memMetrics.HeapSys), Timestamp: timestamp},
		{Name: "memory.heap_inuse", Type: MetricTypeGauge, Value: float64(memMetrics.HeapInuse), Timestamp: timestamp},
		{Name: "memory.stack_inuse", Type: MetricTypeGauge, Value: float64(memMetrics.StackInuse), Timestamp: timestamp},
	}

	for _, metric := range metrics {
		select {
		case mc.metricBuffer <- metric:
		default:
			// Buffer full, drop metric to avoid blocking
		}
	}
}

// recordCPUMetrics records CPU metrics to the store
func (mc *MetricsCollector) recordCPUMetrics(cpuMetrics *CPUMetrics, timestamp time.Time) {
	metrics := []Metric{
		{Name: "cpu.num_goroutines", Type: MetricTypeGauge, Value: float64(cpuMetrics.NumGoroutines), Timestamp: timestamp},
		{Name: "cpu.num_cpu", Type: MetricTypeGauge, Value: float64(cpuMetrics.NumCPU), Timestamp: timestamp},
		{Name: "cpu.cgo_calls", Type: MetricTypeCounter, Value: float64(cpuMetrics.CGOCalls), Timestamp: timestamp},
	}

	for _, metric := range metrics {
		select {
		case mc.metricBuffer <- metric:
		default:
			// Buffer full, drop metric to avoid blocking
		}
	}
}

// recordProcessingMetrics records processing metrics to the store
func (mc *MetricsCollector) recordProcessingMetrics(procMetrics *ProcessingMetrics, timestamp time.Time) {
	metrics := []Metric{
		{Name: "processing.lines_per_second", Type: MetricTypeGauge, Value: procMetrics.LinesPerSecond, Timestamp: timestamp},
		{Name: "processing.bytes_per_second", Type: MetricTypeGauge, Value: procMetrics.BytesPerSecond, Timestamp: timestamp},
		{Name: "processing.total_lines", Type: MetricTypeCounter, Value: float64(procMetrics.TotalLinesProcessed), Timestamp: timestamp},
		{Name: "processing.total_bytes", Type: MetricTypeCounter, Value: float64(procMetrics.TotalBytesProcessed), Timestamp: timestamp},
	}

	for _, metric := range metrics {
		select {
		case mc.metricBuffer <- metric:
		default:
			// Buffer full, drop metric to avoid blocking
		}
	}
}

// processMetricBuffer processes metrics from the buffer
func (mc *MetricsCollector) processMetricBuffer() {
	defer mc.wg.Done()

	for metric := range mc.metricBuffer {
		mc.store.Record(&metric)
	}
}

// TrackOperation tracks an operation with timing
func (mc *MetricsCollector) TrackOperation(operation OperationType, fn func()) error {
	return mc.TrackOperationWithError(operation, func() error {
		fn()
		return nil
	})
}

// TrackOperationWithError tracks an operation that may return an error
func (mc *MetricsCollector) TrackOperationWithError(operation OperationType, fn func() error) error {
	start := time.Now()

	// Execute the operation
	err := fn()

	duration := time.Since(start)

	// Record timing
	mc.operationMutex.RLock()
	timer, exists := mc.operationTimers[operation]
	mc.operationMutex.RUnlock()

	if exists {
		timer.Record(duration)
	}

	// Record operation metric
	now := time.Now()
	labels := map[string]string{
		"operation": string(operation),
	}

	if err != nil {
		labels["status"] = "error"
	} else {
		labels["status"] = "success"
	}

	metric := Metric{
		Name:      "operation.duration",
		Type:      MetricTypeTiming,
		Value:     float64(duration.Nanoseconds()),
		Timestamp: now,
		Labels:    labels,
	}

	// Send to buffer (non-blocking)
	select {
	case mc.metricBuffer <- metric:
	default:
		// Buffer full, drop metric
	}

	return err
}

// RecordMetric records a custom metric
func (mc *MetricsCollector) RecordMetric(metric *Metric) error {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	select {
	case mc.metricBuffer <- *metric:
		return nil
	default:
		// Buffer full, return error
		return ErrBufferFull
	}
}

// GetSnapshot returns a current metrics snapshot
func (mc *MetricsCollector) GetSnapshot() MetricsSnapshot {
	now := time.Now()

	snapshot := MetricsSnapshot{
		Timestamp: now,
	}

	// Collect current system metrics
	if mc.config.EnableMemoryMetrics {
		snapshot.Memory = mc.memoryCollector.Collect()
	}

	if mc.config.EnableCPUMetrics {
		snapshot.CPU = mc.cpuCollector.Collect()
	}

	if mc.config.EnableProcessingMetrics {
		snapshot.Processing = mc.processingCollector.Collect()
	}

	// Collect operation metrics
	mc.operationMutex.RLock()
	operations := make([]OperationMetrics, 0, len(mc.operationTimers))
	for operation, timer := range mc.operationTimers {
		opMetrics := OperationMetrics{
			Operation:    operation,
			Count:        timer.Count(),
			TotalTime:    timer.TotalTime().Nanoseconds(),
			MinTime:      timer.MinTime().Nanoseconds(),
			MaxTime:      timer.MaxTime().Nanoseconds(),
			LastTime:     0,             // Would need additional tracking
			ErrorCount:   0,             // Would need additional tracking
			SuccessCount: timer.Count(), // Simplified
		}
		operations = append(operations, opMetrics)
	}
	mc.operationMutex.RUnlock()

	snapshot.Operations = operations

	return snapshot
}

// RecordLines records the number of lines processed
func (mc *MetricsCollector) RecordLines(count int64) {
	mc.processingCollector.RecordLines(count)
}

// RecordBytes records the number of bytes processed
func (mc *MetricsCollector) RecordBytes(count int64) {
	mc.processingCollector.RecordBytes(count)
}

// GetStore returns the metrics store for advanced queries
func (mc *MetricsCollector) GetStore() *MetricsStore {
	return mc.store
}

// Custom errors
var (
	ErrBufferFull = &MonitorError{Message: "metric buffer is full"}
)

// MonitorError represents a monitoring error
type MonitorError struct {
	Message string
}

func (e *MonitorError) Error() string {
	return e.Message
}
