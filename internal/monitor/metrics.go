package monitor

import (
	"math"
	"runtime"
	"sync/atomic"
	"time"
)

// MetricType represents the type of metric being collected
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTiming    MetricType = "timing"
)

// OperationType represents different operation types being monitored
type OperationType string

const (
	OperationParse    OperationType = "parse"
	OperationAnalyze  OperationType = "analyze"
	OperationAI       OperationType = "ai"
	OperationFileIO   OperationType = "file_io"
	OperationPattern  OperationType = "pattern"
	OperationInsight  OperationType = "insight"
	OperationTimeline OperationType = "timeline"
)

// Metric represents a single metric data point
type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryMetrics holds memory-related performance metrics
type MemoryMetrics struct {
	CurrentAlloc uint64 `json:"current_alloc"`  // bytes currently allocated
	TotalAlloc   uint64 `json:"total_alloc"`    // total bytes allocated
	Sys          uint64 `json:"sys"`            // total bytes from system
	NumGC        uint32 `json:"num_gc"`         // number of garbage collections
	HeapAlloc    uint64 `json:"heap_alloc"`     // bytes allocated in heap
	HeapSys      uint64 `json:"heap_sys"`       // bytes obtained from system
	HeapInuse    uint64 `json:"heap_inuse"`     // bytes in in-use spans
	StackInuse   uint64 `json:"stack_inuse"`    // bytes in stack spans
	PauseTotalNs uint64 `json:"pause_total_ns"` // total GC pause time
	LastGCTime   int64  `json:"last_gc_time"`   // time of last GC
}

// CPUMetrics holds CPU-related performance metrics
type CPUMetrics struct {
	NumGoroutines int     `json:"num_goroutines"` // number of goroutines
	NumCPU        int     `json:"num_cpu"`        // number of CPUs
	CGOCalls      int64   `json:"cgo_calls"`      // number of cgo calls
	UserTime      float64 `json:"user_time"`      // user CPU time in seconds
	SystemTime    float64 `json:"system_time"`    // system CPU time in seconds
}

// ProcessingMetrics holds processing performance metrics
type ProcessingMetrics struct {
	LinesPerSecond      float64 `json:"lines_per_second"`
	BytesPerSecond      float64 `json:"bytes_per_second"`
	TotalLinesProcessed int64   `json:"total_lines_processed"`
	TotalBytesProcessed int64   `json:"total_bytes_processed"`
	ProcessingDuration  int64   `json:"processing_duration_ns"`
}

// OperationMetrics holds metrics for specific operations
type OperationMetrics struct {
	Operation    OperationType `json:"operation"`
	Count        int64         `json:"count"`
	TotalTime    int64         `json:"total_time_ns"`
	MinTime      int64         `json:"min_time_ns"`
	MaxTime      int64         `json:"max_time_ns"`
	LastTime     int64         `json:"last_time_ns"`
	ErrorCount   int64         `json:"error_count"`
	SuccessCount int64         `json:"success_count"`
}

// MetricsSnapshot represents a point-in-time snapshot of all metrics
type MetricsSnapshot struct {
	Timestamp  time.Time          `json:"timestamp"`
	Memory     MemoryMetrics      `json:"memory"`
	CPU        CPUMetrics         `json:"cpu"`
	Processing ProcessingMetrics  `json:"processing"`
	Operations []OperationMetrics `json:"operations"`
}

// Counter is a thread-safe counter metric
type Counter struct {
	value int64
	name  string
}

// NewCounter creates a new counter metric
func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Add adds the given value to the counter
func (c *Counter) Add(value int64) {
	atomic.AddInt64(&c.value, value)
}

// Get returns the current counter value
func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// Reset resets the counter to 0
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// Name returns the counter name
func (c *Counter) Name() string {
	return c.name
}

// Gauge is a thread-safe gauge metric that can go up and down
type Gauge struct {
	value uint64 // using uint64 to store float64 bits atomically
	name  string
}

// NewGauge creates a new gauge metric
func NewGauge(name string) *Gauge {
	return &Gauge{name: name}
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value float64) {
	atomic.StoreUint64(&g.value, math.Float64bits(value))
}

// Get returns the current gauge value
func (g *Gauge) Get() float64 {
	return math.Float64frombits(atomic.LoadUint64(&g.value))
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	g.Add(1)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	g.Add(-1)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(value float64) {
	for {
		old := atomic.LoadUint64(&g.value)
		oldFloat := math.Float64frombits(old)
		newFloat := oldFloat + value
		newBits := math.Float64bits(newFloat)
		if atomic.CompareAndSwapUint64(&g.value, old, newBits) {
			break
		}
	}
}

// Name returns the gauge name
func (g *Gauge) Name() string {
	return g.name
}

// Timer is a thread-safe timer for measuring operation durations
type Timer struct {
	count     int64
	totalTime int64
	minTime   int64
	maxTime   int64
	name      string
}

// NewTimer creates a new timer metric
func NewTimer(name string) *Timer {
	return &Timer{
		name:    name,
		minTime: int64(^uint64(0) >> 1), // max int64
	}
}

// Record records a duration measurement
func (t *Timer) Record(duration time.Duration) {
	nanos := duration.Nanoseconds()

	atomic.AddInt64(&t.count, 1)
	atomic.AddInt64(&t.totalTime, nanos)

	// Update min time
	for {
		current := atomic.LoadInt64(&t.minTime)
		if nanos >= current {
			break
		}
		if atomic.CompareAndSwapInt64(&t.minTime, current, nanos) {
			break
		}
	}

	// Update max time
	for {
		current := atomic.LoadInt64(&t.maxTime)
		if nanos <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&t.maxTime, current, nanos) {
			break
		}
	}
}

// Count returns the number of recorded measurements
func (t *Timer) Count() int64 {
	return atomic.LoadInt64(&t.count)
}

// TotalTime returns the total time of all measurements
func (t *Timer) TotalTime() time.Duration {
	return time.Duration(atomic.LoadInt64(&t.totalTime))
}

// MinTime returns the minimum recorded time
func (t *Timer) MinTime() time.Duration {
	minTime := atomic.LoadInt64(&t.minTime)
	if minTime == int64(^uint64(0)>>1) {
		return 0
	}
	return time.Duration(minTime)
}

// MaxTime returns the maximum recorded time
func (t *Timer) MaxTime() time.Duration {
	return time.Duration(atomic.LoadInt64(&t.maxTime))
}

// AvgTime returns the average time of all measurements
func (t *Timer) AvgTime() time.Duration {
	count := atomic.LoadInt64(&t.count)
	if count == 0 {
		return 0
	}
	total := atomic.LoadInt64(&t.totalTime)
	return time.Duration(total / count)
}

// Reset resets all timer metrics
func (t *Timer) Reset() {
	atomic.StoreInt64(&t.count, 0)
	atomic.StoreInt64(&t.totalTime, 0)
	atomic.StoreInt64(&t.minTime, int64(^uint64(0)>>1))
	atomic.StoreInt64(&t.maxTime, 0)
}

// Name returns the timer name
func (t *Timer) Name() string {
	return t.name
}

// MemoryCollector collects memory metrics from the Go runtime
type MemoryCollector struct{}

// NewMemoryCollector creates a new memory metrics collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{}
}

// Collect collects current memory metrics
func (mc *MemoryCollector) Collect() MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemoryMetrics{
		CurrentAlloc: m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		NumGC:        m.NumGC,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapInuse:    m.HeapInuse,
		StackInuse:   m.StackInuse,
		PauseTotalNs: m.PauseTotalNs,
		LastGCTime:   int64(m.LastGC), // #nosec G115
	}
}

// CPUCollector collects CPU-related metrics
type CPUCollector struct{}

// NewCPUCollector creates a new CPU metrics collector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{}
}

// Collect collects current CPU metrics
func (cc *CPUCollector) Collect() CPUMetrics {
	return CPUMetrics{
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		CGOCalls:      runtime.NumCgoCall(),
		// UserTime and SystemTime would require platform-specific code
		// For now, set to 0 - can be enhanced later
		UserTime:   0,
		SystemTime: 0,
	}
}

// ProcessingCollector tracks processing performance metrics
type ProcessingCollector struct {
	linesProcessed *Counter
	bytesProcessed *Counter
	startTime      time.Time
}

// NewProcessingCollector creates a new processing metrics collector
func NewProcessingCollector() *ProcessingCollector {
	return &ProcessingCollector{
		linesProcessed: NewCounter("lines_processed"),
		bytesProcessed: NewCounter("bytes_processed"),
		startTime:      time.Now(),
	}
}

// RecordLines records the number of lines processed
func (pc *ProcessingCollector) RecordLines(count int64) {
	pc.linesProcessed.Add(count)
}

// RecordBytes records the number of bytes processed
func (pc *ProcessingCollector) RecordBytes(count int64) {
	pc.bytesProcessed.Add(count)
}

// Collect collects current processing metrics
func (pc *ProcessingCollector) Collect() ProcessingMetrics {
	duration := time.Since(pc.startTime)
	durationSeconds := duration.Seconds()

	totalLines := pc.linesProcessed.Get()
	totalBytes := pc.bytesProcessed.Get()

	var linesPerSecond, bytesPerSecond float64
	if durationSeconds > 0 {
		linesPerSecond = float64(totalLines) / durationSeconds
		bytesPerSecond = float64(totalBytes) / durationSeconds
	}

	return ProcessingMetrics{
		LinesPerSecond:      linesPerSecond,
		BytesPerSecond:      bytesPerSecond,
		TotalLinesProcessed: totalLines,
		TotalBytesProcessed: totalBytes,
		ProcessingDuration:  duration.Nanoseconds(),
	}
}

// Reset resets the processing collector
func (pc *ProcessingCollector) Reset() {
	pc.linesProcessed.Reset()
	pc.bytesProcessed.Reset()
	pc.startTime = time.Now()
}
