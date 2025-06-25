package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type MetricType int

const (
	RequestMetric MetricType = iota
	DatabaseMetric
	HookMetric
	ErrorMetric
)

type Metric struct {
	ID        string                 `json:"id"`
	Type      MetricType             `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Method    string                 `json:"method,omitempty"`
	Path      string                 `json:"path,omitempty"`
	Status    int                    `json:"status,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type AggregatedMetric struct {
	Timestamp    time.Time `json:"timestamp"`
	Collection   string    `json:"collection"`
	Period       string    `json:"period"`
	Count        int64     `json:"count"`
	AvgDuration  float64   `json:"avg_duration"`
	MinDuration  float64   `json:"min_duration"`
	MaxDuration  float64   `json:"max_duration"`
	ErrorCount   int64     `json:"error_count"`
	ErrorRate    float64   `json:"error_rate"`
	RequestCount int64     `json:"request_count"`
	DatabaseOps  int64     `json:"database_ops"`
	HookCalls    int64     `json:"hook_calls"`
}

type MetricsData struct {
	DetailedMetrics []Metric                       `json:"detailed_metrics"`
	HourlyAgg       map[string]AggregatedMetric    `json:"hourly_agg"`
	DailyAgg        map[string]AggregatedMetric    `json:"daily_agg"`
	MonthlyAgg      map[string]AggregatedMetric    `json:"monthly_agg"`
	EventMetrics    map[string][]Metric            `json:"event_metrics"`
	LastSave        time.Time                      `json:"last_save"`
}

type Collector struct {
	mu              sync.RWMutex
	detailedMetrics []Metric
	hourlyAgg       map[string]AggregatedMetric  // key: "collection:hour"
	dailyAgg        map[string]AggregatedMetric  // key: "collection:day"
	monthlyAgg      map[string]AggregatedMetric  // key: "collection:month"
	eventMetrics    map[string][]Metric          // key: "collection.event"
	startTime       time.Time
	dataPath        string
	lastFlush       time.Time
	flushInterval   time.Duration
}

func NewCollector() *Collector {
	c := &Collector{
		detailedMetrics: make([]Metric, 0),
		hourlyAgg:       make(map[string]AggregatedMetric),
		dailyAgg:        make(map[string]AggregatedMetric),
		monthlyAgg:      make(map[string]AggregatedMetric),
		eventMetrics:    make(map[string][]Metric),
		startTime:       time.Now(),
		dataPath:        "resources/metrics.json",
		flushInterval:   5 * time.Minute, // Flush every 5 minutes
		lastFlush:       time.Now(),
	}
	
	// Load existing data on startup
	c.loadFromDisk()
	
	// Start periodic flush routine
	go c.periodicFlush()
	
	return c
}

func (c *Collector) loadFromDisk() {
	if _, err := os.Stat(c.dataPath); os.IsNotExist(err) {
		return // File doesn't exist, start fresh
	}

	data, err := os.ReadFile(c.dataPath)
	if err != nil {
		fmt.Printf("Warning: Failed to read metrics file: %v\n", err)
		return
	}

	var metricsData MetricsData
	if err := json.Unmarshal(data, &metricsData); err != nil {
		fmt.Printf("Warning: Failed to parse metrics file: %v\n", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Restore data, but clean up old entries
	now := time.Now()
	cutoff24h := now.Add(-24 * time.Hour)
	cutoff7d := now.Add(-7 * 24 * time.Hour)
	cutoff6m := now.AddDate(0, -6, 0)
	cutoff12m := now.AddDate(-1, 0, 0)

	// Restore detailed metrics (keep only last 24h)
	for _, metric := range metricsData.DetailedMetrics {
		if metric.Timestamp.After(cutoff24h) {
			c.detailedMetrics = append(c.detailedMetrics, metric)
		}
	}

	// Restore hourly aggregations (keep only last 7 days)
	for key, agg := range metricsData.HourlyAgg {
		if agg.Timestamp.After(cutoff7d) {
			c.hourlyAgg[key] = agg
		}
	}

	// Restore daily aggregations (keep only last 6 months)
	for key, agg := range metricsData.DailyAgg {
		if agg.Timestamp.After(cutoff6m) {
			c.dailyAgg[key] = agg
		}
	}

	// Restore monthly aggregations (keep only last 12 months)
	for key, agg := range metricsData.MonthlyAgg {
		if agg.Timestamp.After(cutoff12m) {
			c.monthlyAgg[key] = agg
		}
	}

	// Restore event metrics (keep only last 24h)
	for eventKey, metrics := range metricsData.EventMetrics {
		var filteredEvents []Metric
		for _, metric := range metrics {
			if metric.Timestamp.After(cutoff24h) {
				filteredEvents = append(filteredEvents, metric)
			}
		}
		if len(filteredEvents) > 0 {
			c.eventMetrics[eventKey] = filteredEvents
		}
	}

	fmt.Printf("📊 Loaded metrics: %d detailed, %d hourly, %d daily, %d monthly aggregations\n",
		len(c.detailedMetrics), len(c.hourlyAgg), len(c.dailyAgg), len(c.monthlyAgg))
}

func (c *Collector) periodicFlush() {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flushToDisk()
		}
	}
}

func (c *Collector) flushToDisk() {
	c.mu.RLock()
	data := MetricsData{
		DetailedMetrics: c.detailedMetrics,
		HourlyAgg:       c.hourlyAgg,
		DailyAgg:        c.dailyAgg,
		MonthlyAgg:      c.monthlyAgg,
		EventMetrics:    c.eventMetrics,
		LastSave:        time.Now(),
	}
	c.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(c.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create metrics directory: %v\n", err)
		return
	}

	// Write to temporary file first, then rename (atomic operation)
	tempPath := c.dataPath + ".tmp"
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Warning: Failed to marshal metrics data: %v\n", err)
		return
	}

	if err := os.WriteFile(tempPath, jsonData, 0644); err != nil {
		fmt.Printf("Warning: Failed to write metrics file: %v\n", err)
		return
	}

	if err := os.Rename(tempPath, c.dataPath); err != nil {
		fmt.Printf("Warning: Failed to rename metrics file: %v\n", err)
		os.Remove(tempPath) // Clean up temp file
		return
	}

	c.mu.Lock()
	c.lastFlush = time.Now()
	c.mu.Unlock()
}

func (c *Collector) RecordMetric(metric Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric.ID = generateID()
	metric.Timestamp = time.Now()
	
	// Extract collection from metadata or path
	collection := c.extractCollection(metric)
	
	c.detailedMetrics = append(c.detailedMetrics, metric)
	
	// Store event-specific metrics for hook events
	if metric.Type == HookMetric {
		if eventFull, ok := metric.Metadata["event_full"].(string); ok {
			c.eventMetrics[eventFull] = append(c.eventMetrics[eventFull], metric)
		}
	}
	
	c.updateAggregated(metric, collection)
	c.cleanup()
}

func (c *Collector) extractCollection(metric Metric) string {
	// Try metadata first
	if metric.Metadata != nil {
		if collection, ok := metric.Metadata["collection"].(string); ok && collection != "" {
			return collection
		}
	}

	// Try to extract from path
	if metric.Path != "" {
		// Remove leading slash and take first segment
		path := metric.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		
		// Split by slash and take first part
		for i, char := range path {
			if char == '/' {
				if i > 0 {
					segment := path[:i]
					if segment != "_dashboard" && segment != "_admin" {
						return segment
					}
				}
				break
			}
		}
		
		// If no slash found, use the whole path (minus leading slash)
		if path != "_dashboard" && path != "_admin" && path != "" {
			return path
		}
	}

	return "system"
}

func (c *Collector) updateAggregated(metric Metric, collection string) {
	now := metric.Timestamp
	
	// Update hourly aggregation
	hour := now.Truncate(time.Hour)
	hourKey := collection + ":" + hour.Format("2006-01-02T15")
	c.updateAggregatedMetric(hourKey, hour, collection, "hourly", metric)
	
	// Update daily aggregation  
	day := now.Truncate(24 * time.Hour)
	dayKey := collection + ":" + day.Format("2006-01-02")
	c.updateAggregatedMetric(dayKey, day, collection, "daily", metric)
	
	// Update monthly aggregation
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthKey := collection + ":" + month.Format("2006-01")
	c.updateAggregatedMetric(monthKey, month, collection, "monthly", metric)
}

func (c *Collector) updateAggregatedMetric(key string, timestamp time.Time, collection, period string, metric Metric) {
	var aggMap map[string]AggregatedMetric
	switch period {
	case "hourly":
		aggMap = c.hourlyAgg
	case "daily":
		aggMap = c.dailyAgg
	case "monthly":
		aggMap = c.monthlyAgg
	default:
		return
	}
	
	agg, exists := aggMap[key]
	if !exists {
		agg = AggregatedMetric{
			Timestamp:   timestamp,
			Collection:  collection,
			Period:      period,
			MinDuration: float64(metric.Duration.Nanoseconds()),
			MaxDuration: float64(metric.Duration.Nanoseconds()),
		}
	}

	agg.Count++
	duration := float64(metric.Duration.Nanoseconds())
	
	// Update average duration
	if agg.Count == 1 {
		agg.AvgDuration = duration
	} else {
		agg.AvgDuration = (agg.AvgDuration*float64(agg.Count-1) + duration) / float64(agg.Count)
	}
	
	// Update min/max duration
	if duration < agg.MinDuration {
		agg.MinDuration = duration
	}
	if duration > agg.MaxDuration {
		agg.MaxDuration = duration
	}

	// Count by type
	switch metric.Type {
	case RequestMetric:
		agg.RequestCount++
		if metric.Status >= 400 {
			agg.ErrorCount++
		}
	case DatabaseMetric:
		agg.DatabaseOps++
		if metric.Error != "" {
			agg.ErrorCount++
		}
	case HookMetric:
		agg.HookCalls++
		if metric.Error != "" {
			agg.ErrorCount++
		}
	case ErrorMetric:
		agg.ErrorCount++
	}

	// Calculate error rate
	if agg.Count > 0 {
		agg.ErrorRate = float64(agg.ErrorCount) / float64(agg.Count) * 100
	}

	aggMap[key] = agg
}

func (c *Collector) cleanup() {
	now := time.Now()
	
	// Clean detailed metrics older than 24 hours
	cutoff24h := now.Add(-24 * time.Hour)
	var filtered []Metric
	for _, metric := range c.detailedMetrics {
		if metric.Timestamp.After(cutoff24h) {
			filtered = append(filtered, metric)
		}
	}
	c.detailedMetrics = filtered

	// Clean hourly aggregations older than 7 days
	cutoff7d := now.Add(-7 * 24 * time.Hour)
	for key, agg := range c.hourlyAgg {
		if agg.Timestamp.Before(cutoff7d) {
			delete(c.hourlyAgg, key)
		}
	}

	// Clean daily aggregations older than 6 months
	cutoff6m := now.AddDate(0, -6, 0)
	for key, agg := range c.dailyAgg {
		if agg.Timestamp.Before(cutoff6m) {
			delete(c.dailyAgg, key)
		}
	}

	// Clean monthly aggregations older than 12 months
	cutoff12m := now.AddDate(-1, 0, 0)
	for key, agg := range c.monthlyAgg {
		if agg.Timestamp.Before(cutoff12m) {
			delete(c.monthlyAgg, key)
		}
	}

	// Clean event metrics older than 24 hours
	for eventKey, metrics := range c.eventMetrics {
		var filteredEvents []Metric
		for _, metric := range metrics {
			if metric.Timestamp.After(cutoff24h) {
				filteredEvents = append(filteredEvents, metric)
			}
		}
		if len(filteredEvents) > 0 {
			c.eventMetrics[eventKey] = filteredEvents
		} else {
			delete(c.eventMetrics, eventKey)
		}
	}
}

func (c *Collector) GetDetailedMetrics(since time.Time) []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Metric
	for _, metric := range c.detailedMetrics {
		if metric.Timestamp.After(since) {
			result = append(result, metric)
		}
	}
	return result
}

func (c *Collector) GetDetailedMetricsByCollection(collection string, since time.Time) []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Metric
	for _, metric := range c.detailedMetrics {
		if metric.Timestamp.After(since) {
			metricCollection := c.extractCollection(metric)
			if collection == "overall" || collection == "all" || metricCollection == collection {
				result = append(result, metric)
			}
		}
	}
	return result
}

func (c *Collector) GetAggregatedMetrics(period string, collection string, since time.Time) []AggregatedMetric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var aggMap map[string]AggregatedMetric
	switch period {
	case "hourly":
		aggMap = c.hourlyAgg
	case "daily":
		aggMap = c.dailyAgg
	case "monthly":
		aggMap = c.monthlyAgg
	default:
		return []AggregatedMetric{}
	}

	var result []AggregatedMetric
	for _, agg := range aggMap {
		if agg.Timestamp.After(since) {
			if collection == "overall" || collection == "all" || collection == "" || agg.Collection == collection {
				result = append(result, agg)
			}
		}
	}
	return result
}

func (c *Collector) GetEventMetrics(collection string) map[string][]Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if collection == "overall" || collection == "all" || collection == "" {
		return c.eventMetrics
	}

	result := make(map[string][]Metric)
	for eventKey, metrics := range c.eventMetrics {
		if len(eventKey) > len(collection) && eventKey[:len(collection)] == collection {
			result[eventKey] = metrics
		}
	}
	return result
}

func (c *Collector) GetCollections() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	collections := make(map[string]bool)
	collections["overall"] = true
	collections["all"] = true
	collections["system"] = true

	// Get collections from recent metrics
	for _, metric := range c.detailedMetrics {
		collection := c.extractCollection(metric)
		if collection != "system" {
			collections[collection] = true
		}
	}

	result := make([]string, 0, len(collections))
	for collection := range collections {
		result = append(result, collection)
	}
	return result
}

func (c *Collector) GetSystemStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	uptime := now.Sub(c.startTime)
	
	// Count metrics in last hour
	lastHour := now.Add(-time.Hour)
	var hourlyCount int64
	var hourlyErrors int64
	
	for _, metric := range c.detailedMetrics {
		if metric.Timestamp.After(lastHour) {
			hourlyCount++
			if (metric.Type == RequestMetric && metric.Status >= 400) ||
			   (metric.Type != RequestMetric && metric.Error != "") {
				hourlyErrors++
			}
		}
	}

	return map[string]interface{}{
		"uptime_seconds":       uptime.Seconds(),
		"total_metrics":        len(c.detailedMetrics),
		"hourly_requests":      hourlyCount,
		"hourly_error_rate":    func() float64 {
			if hourlyCount > 0 {
				return float64(hourlyErrors) / float64(hourlyCount) * 100
			}
			return 0
		}(),
		"aggregated_periods":   len(c.hourlyAgg) + len(c.dailyAgg) + len(c.monthlyAgg),
		"collections":          len(c.GetCollections()),
		"event_types":          len(c.eventMetrics),
	}
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000000")
}