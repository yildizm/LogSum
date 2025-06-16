package monitor

import (
	"sort"
	"sync"
	"time"
)

// TimeSeriesDataPoint represents a single data point in a time series
type TimeSeriesDataPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// TimeSeries represents a time series of data points
type TimeSeries struct {
	Name       string                `json:"name"`
	MetricType MetricType            `json:"type"`
	DataPoints []TimeSeriesDataPoint `json:"data_points"`
	mutex      sync.RWMutex
}

// NewTimeSeries creates a new time series
func NewTimeSeries(name string, metricType MetricType) *TimeSeries {
	return &TimeSeries{
		Name:       name,
		MetricType: metricType,
		DataPoints: make([]TimeSeriesDataPoint, 0),
	}
}

// Add adds a data point to the time series
func (ts *TimeSeries) Add(timestamp time.Time, value float64, labels map[string]string) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	dataPoint := TimeSeriesDataPoint{
		Timestamp: timestamp,
		Value:     value,
		Labels:    labels,
	}

	ts.DataPoints = append(ts.DataPoints, dataPoint)

	// Keep series sorted by timestamp
	sort.Slice(ts.DataPoints, func(i, j int) bool {
		return ts.DataPoints[i].Timestamp.Before(ts.DataPoints[j].Timestamp)
	})
}

// GetRange returns data points within the specified time range
func (ts *TimeSeries) GetRange(start, end time.Time) []TimeSeriesDataPoint {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	var result []TimeSeriesDataPoint
	for _, dp := range ts.DataPoints {
		if (dp.Timestamp.Equal(start) || dp.Timestamp.After(start)) &&
			(dp.Timestamp.Equal(end) || dp.Timestamp.Before(end)) {
			result = append(result, dp)
		}
	}

	return result
}

// GetLatest returns the most recent data points (up to limit)
func (ts *TimeSeries) GetLatest(limit int) []TimeSeriesDataPoint {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	if len(ts.DataPoints) == 0 {
		return []TimeSeriesDataPoint{}
	}

	start := len(ts.DataPoints) - limit
	if start < 0 {
		start = 0
	}

	result := make([]TimeSeriesDataPoint, len(ts.DataPoints[start:]))
	copy(result, ts.DataPoints[start:])
	return result
}

// Prune removes data points older than the retention period
func (ts *TimeSeries) Prune(retentionPeriod time.Duration) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)

	// Find first index that should be kept
	keepIndex := 0
	for i, dp := range ts.DataPoints {
		if dp.Timestamp.After(cutoff) {
			keepIndex = i
			break
		}
		keepIndex = i + 1
	}

	// Keep only recent data points
	if keepIndex > 0 && keepIndex < len(ts.DataPoints) {
		ts.DataPoints = ts.DataPoints[keepIndex:]
	} else if keepIndex >= len(ts.DataPoints) {
		ts.DataPoints = ts.DataPoints[:0] // Clear all
	}
}

// Size returns the number of data points in the series
func (ts *TimeSeries) Size() int {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return len(ts.DataPoints)
}

// Aggregates represents statistical aggregates for a time series
type Aggregates struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Sum   float64 `json:"sum"`
	Count int     `json:"count"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
}

// CalculateAggregates calculates statistical aggregates for the time series data
func (ts *TimeSeries) CalculateAggregates(start, end time.Time) Aggregates {
	dataPoints := ts.GetRange(start, end)

	if len(dataPoints) == 0 {
		return Aggregates{}
	}

	values := make([]float64, len(dataPoints))
	for i, dp := range dataPoints {
		values[i] = dp.Value
	}

	return calculateAggregatesFromValues(values)
}

// calculateAggregatesFromValues calculates aggregates from a slice of values
func calculateAggregatesFromValues(values []float64) Aggregates {
	if len(values) == 0 {
		return Aggregates{}
	}

	// Sort values for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate basic stats
	minVal := sorted[0]
	maxVal := sorted[len(sorted)-1]
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	avg := sum / float64(len(values))

	// Calculate percentiles
	p50 := percentile(sorted, 0.50)
	p95 := percentile(sorted, 0.95)
	p99 := percentile(sorted, 0.99)

	return Aggregates{
		Min:   minVal,
		Max:   maxVal,
		Avg:   avg,
		Sum:   sum,
		Count: len(values),
		P50:   p50,
		P95:   p95,
		P99:   p99,
	}
}

// percentile calculates the nth percentile of sorted values
func percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := p * float64(len(sortedValues)-1)
	lowerIdx := int(index)
	upperIdx := lowerIdx + 1

	if upperIdx >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	// Linear interpolation
	weight := index - float64(lowerIdx)
	return sortedValues[lowerIdx]*(1-weight) + sortedValues[upperIdx]*weight
}

// MetricsStore manages multiple time series and provides storage operations
type MetricsStore struct {
	series          map[string]*TimeSeries
	retentionPeriod time.Duration
	maxDataPoints   int
	mutex           sync.RWMutex
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore(retentionPeriod time.Duration, maxDataPoints int) *MetricsStore {
	return &MetricsStore{
		series:          make(map[string]*TimeSeries),
		retentionPeriod: retentionPeriod,
		maxDataPoints:   maxDataPoints,
	}
}

// Record records a metric data point
func (ms *MetricsStore) Record(metric *Metric) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Get or create time series
	ts, exists := ms.series[metric.Name]
	if !exists {
		ts = NewTimeSeries(metric.Name, metric.Type)
		ms.series[metric.Name] = ts
	}

	// Add data point
	ts.Add(metric.Timestamp, metric.Value, metric.Labels)

	// Enforce max data points limit
	ms.enforceDataPointsLimit(ts)
}

// enforceDataPointsLimit removes oldest data points if limit is exceeded
func (ms *MetricsStore) enforceDataPointsLimit(ts *TimeSeries) {
	if ms.maxDataPoints <= 0 {
		return
	}

	if ts.Size() > ms.maxDataPoints {
		// Remove oldest data points
		excess := ts.Size() - ms.maxDataPoints
		ts.DataPoints = ts.DataPoints[excess:]
	}
}

// GetTimeSeries returns a time series by name
func (ms *MetricsStore) GetTimeSeries(name string) (*TimeSeries, bool) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	ts, exists := ms.series[name]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid concurrent access issues
	tsCopy := &TimeSeries{
		Name:       ts.Name,
		MetricType: ts.MetricType,
		DataPoints: make([]TimeSeriesDataPoint, len(ts.DataPoints)),
	}
	copy(tsCopy.DataPoints, ts.DataPoints)

	return tsCopy, true
}

// GetAllSeries returns all time series names
func (ms *MetricsStore) GetAllSeries() []string {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	names := make([]string, 0, len(ms.series))
	for name := range ms.series {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetMetrics returns recent metrics for all series
func (ms *MetricsStore) GetMetrics(limit int) map[string][]TimeSeriesDataPoint {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	result := make(map[string][]TimeSeriesDataPoint)
	for name, ts := range ms.series {
		result[name] = ts.GetLatest(limit)
	}

	return result
}

// GetMetricsInRange returns metrics within a time range for all series
func (ms *MetricsStore) GetMetricsInRange(start, end time.Time) map[string][]TimeSeriesDataPoint {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	result := make(map[string][]TimeSeriesDataPoint)
	for name, ts := range ms.series {
		result[name] = ts.GetRange(start, end)
	}

	return result
}

// CalculateAggregates calculates aggregates for a time series within a time range
func (ms *MetricsStore) CalculateAggregates(seriesName string, start, end time.Time) (Aggregates, bool) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	ts, exists := ms.series[seriesName]
	if !exists {
		return Aggregates{}, false
	}

	return ts.CalculateAggregates(start, end), true
}

// Prune removes old data points from all time series
func (ms *MetricsStore) Prune() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	for _, ts := range ms.series {
		ts.Prune(ms.retentionPeriod)
	}
}

// Clear removes all data from the store
func (ms *MetricsStore) Clear() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	ms.series = make(map[string]*TimeSeries)
}

// Size returns the total number of data points across all series
func (ms *MetricsStore) Size() int {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	total := 0
	for _, ts := range ms.series {
		total += ts.Size()
	}

	return total
}

// Stats returns statistics about the metrics store
func (ms *MetricsStore) Stats() map[string]interface{} {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_series":     len(ms.series),
		"total_datapoints": ms.Size(),
		"retention_period": ms.retentionPeriod.String(),
		"max_datapoints":   ms.maxDataPoints,
	}

	seriesStats := make(map[string]int)
	for name, ts := range ms.series {
		seriesStats[name] = ts.Size()
	}
	stats["series_sizes"] = seriesStats

	return stats
}
