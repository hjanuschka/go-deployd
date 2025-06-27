package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	USER_GENERATED
)

var levelNames = map[LogLevel]string{
	DEBUG:          "DEBUG",
	INFO:           "INFO",
	WARN:           "WARN",
	ERROR:          "ERROR",
	USER_GENERATED: "USER",
}

var levelColors = map[LogLevel]string{
	DEBUG:          "\033[36m", // Cyan
	INFO:           "\033[32m", // Green
	WARN:           "\033[33m", // Yellow
	ERROR:          "\033[31m", // Red
	USER_GENERATED: "\033[35m", // Magenta
}

const colorReset = "\033[0m"

type Fields map[string]interface{}

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source,omitempty"`
	Component   string                 `json:"component,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Caller      string                 `json:"caller,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Environment string                 `json:"environment,omitempty"`
}

type Logger struct {
	mu          sync.RWMutex
	logDir      string
	file        *os.File
	logChan     chan *LogEntry
	stopChan    chan struct{}
	wg          sync.WaitGroup
	devMode     bool
	minLevel    LogLevel
	output      io.Writer
	component   string
	fields      Fields
	sensitiveKeys []string
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Configuration for logger initialization
type Config struct {
	LogDir        string
	DevMode       bool
	MinLevel      LogLevel
	Component     string
	SensitiveKeys []string
}

func InitializeLogger(config Config) error {
	var initErr error
	once.Do(func() {
		chanSize := 1000
		if !config.DevMode {
			chanSize = 10000 // Larger buffer for production
		}

		// Default sensitive keys that should never be logged
		defaultSensitiveKeys := []string{
			"password", "secret", "key", "token", "auth",
			"masterkey", "master_key", "private", "credential",
		}
		
		sensitiveKeys := append(defaultSensitiveKeys, config.SensitiveKeys...)

		globalLogger = &Logger{
			logDir:        config.LogDir,
			logChan:       make(chan *LogEntry, chanSize),
			stopChan:      make(chan struct{}),
			devMode:       config.DevMode,
			minLevel:      config.MinLevel,
			output:        os.Stdout,
			component:     config.Component,
			fields:        make(Fields),
			sensitiveKeys: sensitiveKeys,
		}

		// Create log directory if it doesn't exist
		if err := os.MkdirAll(config.LogDir, 0755); err != nil {
			// Log to stderr if we can't create log directory
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
			initErr = err
			return
		}

		// Open today's log file
		if err := globalLogger.openLogFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			// Continue even if file opening fails - we'll still have console output
		}

		// Start the async logging goroutine
		globalLogger.wg.Add(1)
		go globalLogger.logWorker()

		// Log initialization
		globalLogger.Info("Logger initialized", Fields{
			"dev_mode":  config.DevMode,
			"log_dir":   config.LogDir,
			"min_level": levelNames[config.MinLevel],
			"component": config.Component,
		})
	})
	return initErr
}

func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default config if not initialized
		// Use environment variables for configuration
		devMode := getEnvironment() == "development"
		logLevel := getLogLevelFromEnv()
		
		InitializeLogger(Config{
			LogDir:   "./logs",
			DevMode:  devMode,
			MinLevel: logLevel,
		})
	}
	return globalLogger
}

// WithComponent creates a new logger instance with a specific component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		logDir:        l.logDir,
		logChan:       l.logChan,
		stopChan:      l.stopChan,
		devMode:       l.devMode,
		minLevel:      l.minLevel,
		output:        l.output,
		component:     component,
		fields:        make(Fields),
		sensitiveKeys: l.sensitiveKeys,
	}
}

// WithFields creates a new logger instance with additional fields
func (l *Logger) WithFields(fields Fields) *Logger {
	newFields := make(Fields)
	// Copy existing fields
	for k, v := range l.fields {
		newFields[k] = v
	}
	// Add new fields
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &Logger{
		logDir:        l.logDir,
		logChan:       l.logChan,
		stopChan:      l.stopChan,
		devMode:       l.devMode,
		minLevel:      l.minLevel,
		output:        l.output,
		component:     l.component,
		fields:        newFields,
		sensitiveKeys: l.sensitiveKeys,
	}
}

// WithContext extracts common fields from context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := make(Fields)
	
	// Extract trace ID if present
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}
	
	// Extract user ID if present
	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}
	
	// Extract request ID if present
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}
	
	return l.WithFields(fields)
}

// openLogFile opens or creates today's log file
func (l *Logger) openLogFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close existing file if open
	if l.file != nil {
		l.file.Close()
	}

	// Create filename with today's date
	today := time.Now().Format("2006-01-02")
	filename := filepath.Join(l.logDir, fmt.Sprintf("%s.jsonl", today))

	// Open file in append mode, create if it doesn't exist
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = file
	return nil
}

// getCaller returns the caller information
func getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	
	// Get only the last two directories and filename
	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		file = strings.Join(parts[len(parts)-2:], "/")
	}
	
	return fmt.Sprintf("%s:%d", file, line)
}

// getStackTrace returns the current stack trace
func getStackTrace() string {
	buf := make([]byte, 1024*16)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// redactSensitiveData recursively redacts sensitive information from data
func (l *Logger) redactSensitiveData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		redacted := make(map[string]interface{})
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			shouldRedact := false
			
			// Check if key contains any sensitive keywords
			for _, sensitive := range l.sensitiveKeys {
				if strings.Contains(lowerKey, strings.ToLower(sensitive)) {
					shouldRedact = true
					break
				}
			}
			
			if shouldRedact {
				redacted[key] = "[REDACTED]"
			} else {
				redacted[key] = l.redactSensitiveData(value)
			}
		}
		return redacted
	case []interface{}:
		redacted := make([]interface{}, len(v))
		for i, item := range v {
			redacted[i] = l.redactSensitiveData(item)
		}
		return redacted
	default:
		return v
	}
}

// formatConsoleOutput formats log entry for console output in dev mode
func (l *Logger) formatConsoleOutput(entry *LogEntry) string {
	level := INFO
	for k, v := range levelNames {
		if v == entry.Level {
			level = k
			break
		}
	}
	
	color := levelColors[level]
	timestamp := entry.Timestamp.Format("15:04:05.000")
	
	// Build the message
	var parts []string
	
	// Add timestamp and level with color
	parts = append(parts, fmt.Sprintf("%s%s [%s]%s", color, timestamp, entry.Level, colorReset))
	
	// Add component if present
	if entry.Component != "" {
		parts = append(parts, fmt.Sprintf("[%s]", entry.Component))
	}
	
	// Add message
	parts = append(parts, entry.Message)
	
	// Add key structured data in a readable format (not full JSON)
	if len(entry.Data) > 0 {
		var readableParts []string
		
		// Show important fields in a readable way
		for key, value := range entry.Data {
			switch key {
			case "collection", "event", "eventType", "type", "method", "error", "duration", "durationMs":
				readableParts = append(readableParts, fmt.Sprintf("%s=%v", key, value))
			case "source":
				// Skip source as it's often redundant with component
				continue
			default:
				// For debug level, show more details
				if level == DEBUG {
					if str, ok := value.(string); ok && len(str) > 50 {
						readableParts = append(readableParts, fmt.Sprintf("%s=%.50s...", key, str))
					} else {
						readableParts = append(readableParts, fmt.Sprintf("%s=%v", key, value))
					}
				}
			}
		}
		
		if len(readableParts) > 0 {
			parts = append(parts, fmt.Sprintf("(%s)", strings.Join(readableParts, ", ")))
		}
	}
	
	// Add caller in dev mode for errors and debug
	if l.devMode && entry.Caller != "" && (level == ERROR || level == DEBUG) {
		parts = append(parts, fmt.Sprintf("@%s", entry.Caller))
	}
	
	return strings.Join(parts, " ")
}

func (l *Logger) log(level LogLevel, message string, fields Fields) {
	if level < l.minLevel {
		return
	}

	// Merge logger fields with provided fields
	mergedData := make(map[string]interface{})
	for k, v := range l.fields {
		mergedData[k] = v
	}
	for k, v := range fields {
		mergedData[k] = v
	}

	// Redact sensitive data
	mergedData = l.redactSensitiveData(mergedData).(map[string]interface{})

	entry := &LogEntry{
		Timestamp:   time.Now(),
		Level:       levelNames[level],
		Message:     message,
		Component:   l.component,
		Data:        mergedData,
		Caller:      getCaller(3),
		Environment: getEnvironment(),
	}

	// Add trace ID if present in fields
	if traceID, ok := mergedData["trace_id"]; ok {
		entry.TraceID = fmt.Sprintf("%v", traceID)
	}

	// Add stack trace for errors
	if level == ERROR {
		entry.StackTrace = getStackTrace()
	}

	// Console output
	if l.devMode {
		fmt.Fprintln(l.output, l.formatConsoleOutput(entry))
	}

	// Send to async writer
	select {
	case l.logChan <- entry:
		// Successfully queued
	default:
		// Channel full, log to stderr
		fmt.Fprintf(os.Stderr, "Log channel full, dropping entry: %s\n", message)
	}
}

// Log methods
func (l *Logger) Debug(message string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(DEBUG, message, f)
}

func (l *Logger) Info(message string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(INFO, message, f)
}

func (l *Logger) Warn(message string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(WARN, message, f)
}

func (l *Logger) Error(message string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(ERROR, message, f)
}

func (l *Logger) UserGenerated(message string, fields ...Fields) {
	f := mergeFields(fields...)
	l.log(USER_GENERATED, message, f)
}

// Helper function to merge multiple Fields
func mergeFields(fields ...Fields) Fields {
	result := make(Fields)
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

// logWorker runs in a goroutine and handles async logging
func (l *Logger) logWorker() {
	defer l.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var batch []*LogEntry
	batchSize := 100

	for {
		select {
		case entry := <-l.logChan:
			batch = append(batch, entry)

			if len(batch) >= batchSize {
				l.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				l.flushBatch(batch)
				batch = batch[:0]
			}

			// Check if we need to rotate log file (midnight)
			l.checkLogRotation()

		case <-l.stopChan:
			if len(batch) > 0 {
				l.flushBatch(batch)
			}
			return
		}
	}
}

// checkLogRotation checks if we need to open a new log file
func (l *Logger) checkLogRotation() {
	currentDate := time.Now().Format("2006-01-02")
	l.mu.RLock()
	if l.file != nil {
		currentFile := filepath.Base(l.file.Name())
		expectedFile := fmt.Sprintf("%s.jsonl", currentDate)
		if currentFile != expectedFile {
			l.mu.RUnlock()
			l.openLogFile()
			return
		}
	}
	l.mu.RUnlock()
}

func (l *Logger) flushBatch(batch []*LogEntry) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.file == nil {
		return
	}

	// Write all entries in batch
	for _, entry := range batch {
		if jsonData, err := json.Marshal(entry); err == nil {
			l.file.WriteString(string(jsonData) + "\n")
		}
	}

	// Sync once per batch
	l.file.Sync()
}

// Shutdown gracefully stops the logger
func (l *Logger) Shutdown() {
	if l.stopChan != nil {
		close(l.stopChan)
		l.wg.Wait()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Sync()
		l.file.Close()
		l.file = nil
	}
}

// Global shutdown function
func Shutdown() {
	if globalLogger != nil {
		globalLogger.Shutdown()
	}
}

// File reading methods
func (l *Logger) GetLogFiles() ([]string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.logDir == "" {
		return []string{}, nil
	}

	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return []string{}, err
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jsonl") {
			logFiles = append(logFiles, file.Name())
		}
	}

	sort.Strings(logFiles)
	return logFiles, nil
}

func (l *Logger) ReadLogs(filename string, level string) ([]LogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var logFile string
	if filename == "" {
		today := time.Now().Format("2006-01-02")
		logFile = filepath.Join(l.logDir, fmt.Sprintf("%s.jsonl", today))
	} else {
		logFile = filepath.Join(l.logDir, filename)
	}

	file, err := os.Open(logFile)
	if err != nil {
		return []LogEntry{}, nil
	}
	defer file.Close()

	var logs []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if level != "" && entry.Level != level {
			continue
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

func (l *Logger) GetLogPath(filename string) string {
	if filename == "" {
		today := time.Now().Format("2006-01-02")
		filename = fmt.Sprintf("%s.jsonl", today)
	}
	return filepath.Join(l.logDir, filename)
}

// Helper function to get environment
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	return env
}

// Helper function to parse log level from environment variable
func getLogLevelFromEnv() LogLevel {
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "USER":
		return USER_GENERATED
	default:
		// Default based on environment
		if getEnvironment() == "development" {
			return DEBUG
		}
		return INFO
	}
}

// Global convenience functions that maintain backward compatibility
func Debug(message string, source string, data map[string]interface{}) {
	fields := Fields{"source": source}
	for k, v := range data {
		fields[k] = v
	}
	GetLogger().Debug(message, fields)
}

func Info(message string, source string, data map[string]interface{}) {
	fields := Fields{"source": source}
	for k, v := range data {
		fields[k] = v
	}
	GetLogger().Info(message, fields)
}

func Warn(message string, source string, data map[string]interface{}) {
	fields := Fields{"source": source}
	for k, v := range data {
		fields[k] = v
	}
	GetLogger().Warn(message, fields)
}

func Error(message string, source string, data map[string]interface{}) {
	fields := Fields{"source": source}
	for k, v := range data {
		fields[k] = v
	}
	GetLogger().Error(message, fields)
}

func UserGenerated(message string, source string, data map[string]interface{}) {
	fields := Fields{"source": source}
	for k, v := range data {
		fields[k] = v
	}
	GetLogger().UserGenerated(message, fields)
}