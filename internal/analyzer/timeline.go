package analyzer

import (
	"sort"
	"time"

	"github.com/yildizm/LogSum/internal/common"
)

// TimelineGenerator creates timeline analysis from log entries
type TimelineGenerator struct{}

// NewTimelineGenerator creates a new timeline generator
func NewTimelineGenerator() *TimelineGenerator {
	return &TimelineGenerator{}
}

// GenerateTimeline creates a timeline analysis with specified bucket size
func (g *TimelineGenerator) GenerateTimeline(entries []*common.LogEntry, bucketSize time.Duration) *Timeline {
	if len(entries) == 0 || bucketSize <= 0 {
		return &Timeline{
			Buckets:    []TimeBucket{},
			BucketSize: bucketSize,
		}
	}

	// Sort entries by timestamp to ensure proper timeline ordering
	sortedEntries := make([]*common.LogEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Timestamp.Before(sortedEntries[j].Timestamp)
	})

	// Find time range
	startTime := sortedEntries[0].Timestamp.Truncate(bucketSize)
	endTime := sortedEntries[len(sortedEntries)-1].Timestamp.Truncate(bucketSize).Add(bucketSize)

	// Create buckets for the entire time range
	buckets := g.createBuckets(startTime, endTime, bucketSize)

	// Distribute entries into buckets
	g.distributeEntries(sortedEntries, buckets, bucketSize)

	return &Timeline{
		Buckets:    buckets,
		BucketSize: bucketSize,
	}
}

// createBuckets creates empty time buckets for the given range
func (g *TimelineGenerator) createBuckets(startTime, endTime time.Time, bucketSize time.Duration) []TimeBucket {
	var buckets []TimeBucket

	for current := startTime; current.Before(endTime); current = current.Add(bucketSize) {
		bucket := TimeBucket{
			Start:      current,
			End:        current.Add(bucketSize),
			EntryCount: 0,
			ErrorCount: 0,
			WarnCount:  0,
		}
		buckets = append(buckets, bucket)
	}

	return buckets
}

// distributeEntries distributes log entries into appropriate time buckets
func (g *TimelineGenerator) distributeEntries(entries []*common.LogEntry, buckets []TimeBucket, bucketSize time.Duration) {
	for _, entry := range entries {
		bucketIndex := g.findBucketIndex(entry.Timestamp, buckets, bucketSize)
		if bucketIndex >= 0 && bucketIndex < len(buckets) {
			buckets[bucketIndex].EntryCount++

			// Count by severity level
			switch entry.LogLevel {
			case common.LevelError, common.LevelFatal:
				buckets[bucketIndex].ErrorCount++
			case common.LevelWarn:
				buckets[bucketIndex].WarnCount++
			}
		}
	}
}

// findBucketIndex finds the appropriate bucket index for a timestamp
func (g *TimelineGenerator) findBucketIndex(timestamp time.Time, buckets []TimeBucket, bucketSize time.Duration) int {
	if len(buckets) == 0 {
		return -1
	}

	// Calculate the bucket index based on time difference from start
	startTime := buckets[0].Start
	timeDiff := timestamp.Sub(startTime)
	index := int(timeDiff / bucketSize)

	// Ensure index is within bounds
	if index < 0 {
		return 0
	}
	if index >= len(buckets) {
		return len(buckets) - 1
	}

	return index
}

// GetTimelineStats returns statistical summary of the timeline
func (g *TimelineGenerator) GetTimelineStats(timeline *Timeline) TimelineStats {
	if timeline == nil || len(timeline.Buckets) == 0 {
		return TimelineStats{}
	}

	stats := TimelineStats{
		TotalBuckets: len(timeline.Buckets),
		BucketSize:   timeline.BucketSize,
	}

	var totalEntries, totalErrors, totalWarns int
	var maxEntries, maxErrors int
	activeBuckets := 0

	for _, bucket := range timeline.Buckets {
		totalEntries += bucket.EntryCount
		totalErrors += bucket.ErrorCount
		totalWarns += bucket.WarnCount

		if bucket.EntryCount > 0 {
			activeBuckets++
		}

		if bucket.EntryCount > maxEntries {
			maxEntries = bucket.EntryCount
			stats.PeakActivityTime = bucket.Start
		}

		if bucket.ErrorCount > maxErrors {
			maxErrors = bucket.ErrorCount
			stats.PeakErrorTime = bucket.Start
		}
	}

	stats.TotalEntries = totalEntries
	stats.TotalErrors = totalErrors
	stats.TotalWarnings = totalWarns
	stats.ActiveBuckets = activeBuckets
	stats.MaxEntriesPerBucket = maxEntries
	stats.MaxErrorsPerBucket = maxErrors

	if activeBuckets > 0 {
		stats.AvgEntriesPerBucket = float64(totalEntries) / float64(activeBuckets)
		stats.AvgErrorsPerBucket = float64(totalErrors) / float64(activeBuckets)
	}

	if totalEntries > 0 {
		stats.ErrorRate = float64(totalErrors) / float64(totalEntries)
		stats.WarningRate = float64(totalWarns) / float64(totalEntries)
	}

	return stats
}

// DetectTrends identifies trends in the timeline data
func (g *TimelineGenerator) DetectTrends(timeline *Timeline) []TimelineTrend {
	if timeline == nil || len(timeline.Buckets) < 3 {
		return []TimelineTrend{}
	}

	var trends []TimelineTrend

	// Detect entry count trends
	entryTrend := g.detectTrendInSeries("entry_count", timeline.Buckets, func(b TimeBucket) float64 {
		return float64(b.EntryCount)
	})
	if entryTrend.Strength > 0.3 { // Only report significant trends
		trends = append(trends, entryTrend)
	}

	// Detect error rate trends
	errorRateTrend := g.detectTrendInSeries("error_rate", timeline.Buckets, func(b TimeBucket) float64 {
		if b.EntryCount > 0 {
			return float64(b.ErrorCount) / float64(b.EntryCount)
		}
		return 0
	})
	if errorRateTrend.Strength > 0.3 {
		trends = append(trends, errorRateTrend)
	}

	return trends
}

// detectTrendInSeries detects trends in a time series using simple linear regression
func (g *TimelineGenerator) detectTrendInSeries(name string, buckets []TimeBucket, getValue func(TimeBucket) float64) TimelineTrend {
	n := len(buckets)
	if n < 3 {
		return TimelineTrend{Name: name, Type: "none", Strength: 0}
	}

	// Calculate simple linear regression
	var sumX, sumY, sumXY, sumXX float64

	for i, bucket := range buckets {
		x := float64(i)
		y := getValue(bucket)

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Calculate slope (trend direction)
	nFloat := float64(n)
	slope := (nFloat*sumXY - sumX*sumY) / (nFloat*sumXX - sumX*sumX)

	// Calculate correlation coefficient (trend strength)
	meanX := sumX / nFloat
	meanY := sumY / nFloat

	var ssX, ssY, ssXY float64
	for i, bucket := range buckets {
		x := float64(i)
		y := getValue(bucket)

		ssX += (x - meanX) * (x - meanX)
		ssY += (y - meanY) * (y - meanY)
		ssXY += (x - meanX) * (y - meanY)
	}

	var correlation float64
	if ssX > 0 && ssY > 0 {
		correlation = ssXY / (ssX * ssY)
		if correlation < 0 {
			correlation = -correlation // Use absolute correlation as strength
		}
	}

	// Determine trend type
	trendType := "stable"
	if slope > 0.1 {
		trendType = "increasing"
	} else if slope < -0.1 {
		trendType = "decreasing"
	}

	return TimelineTrend{
		Name:      name,
		Type:      trendType,
		Strength:  correlation,
		Slope:     slope,
		StartTime: buckets[0].Start,
		EndTime:   buckets[len(buckets)-1].End,
	}
}

// TimelineStats provides statistical summary of timeline data
type TimelineStats struct {
	TotalBuckets        int           `json:"total_buckets"`
	ActiveBuckets       int           `json:"active_buckets"`
	TotalEntries        int           `json:"total_entries"`
	TotalErrors         int           `json:"total_errors"`
	TotalWarnings       int           `json:"total_warnings"`
	BucketSize          time.Duration `json:"bucket_size"`
	AvgEntriesPerBucket float64       `json:"avg_entries_per_bucket"`
	AvgErrorsPerBucket  float64       `json:"avg_errors_per_bucket"`
	MaxEntriesPerBucket int           `json:"max_entries_per_bucket"`
	MaxErrorsPerBucket  int           `json:"max_errors_per_bucket"`
	ErrorRate           float64       `json:"error_rate"`
	WarningRate         float64       `json:"warning_rate"`
	PeakActivityTime    time.Time     `json:"peak_activity_time"`
	PeakErrorTime       time.Time     `json:"peak_error_time"`
}

// TimelineTrend represents a detected trend in timeline data
type TimelineTrend struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`     // "increasing", "decreasing", "stable"
	Strength  float64   `json:"strength"` // 0-1, confidence in trend
	Slope     float64   `json:"slope"`    // rate of change
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
