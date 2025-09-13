package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger levels
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	// Global logger instance
	globalLogger *Logger
	once         sync.Once
	
	// Default log settings
	defaultLogDir  = ".tod/logs"
	defaultLogFile = "tod.log"
	maxLogSize     = int64(10 * 1024 * 1024) // 10MB
	maxLogAge      = 7 * 24 * time.Hour      // 7 days
)

// Logger represents the application logger
type Logger struct {
	mu         sync.Mutex
	file       *os.File
	logger     *log.Logger
	level      int
	projectDir string
	logPath    string
	
	// Rotation settings
	maxSize    int64
	currentSize int64
}

// Initialize sets up the global logger
func Initialize(projectDir string) error {
	var initErr error
	once.Do(func() {
		globalLogger = &Logger{
			level:      INFO,
			projectDir: projectDir,
			maxSize:    maxLogSize,
		}
		initErr = globalLogger.init()
	})
	return initErr
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with current directory if not already initialized
		Initialize(".")
	}
	return globalLogger
}

// init initializes the logger
func (l *Logger) init() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Create log directory
	logDir := filepath.Join(l.projectDir, defaultLogDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Create or open log file
	l.logPath = filepath.Join(logDir, defaultLogFile)
	return l.openLogFile()
}

// openLogFile opens or creates the log file
func (l *Logger) openLogFile() error {
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	
	// Get current file size
	info, err := file.Stat()
	if err == nil {
		l.currentSize = info.Size()
	}
	
	l.file = file
	l.logger = log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	
	return nil
}

// rotateIfNeeded checks if log rotation is needed and rotates if necessary
func (l *Logger) rotateIfNeeded() error {
	if l.currentSize < l.maxSize {
		return nil
	}
	
	// Close current file
	if l.file != nil {
		l.file.Close()
	}
	
	// Rotate log file
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := filepath.Join(filepath.Dir(l.logPath), fmt.Sprintf("tod-%s.log", timestamp))
	
	if err := os.Rename(l.logPath, rotatedPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}
	
	// Open new log file
	if err := l.openLogFile(); err != nil {
		return err
	}
	
	// Clean old logs asynchronously
	go l.cleanOldLogs()
	
	return nil
}

// cleanOldLogs removes log files older than maxLogAge
func (l *Logger) cleanOldLogs() {
	logDir := filepath.Dir(l.logPath)
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	
	cutoff := time.Now().Add(-maxLogAge)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		// Skip current log file
		if file.Name() == defaultLogFile {
			continue
		}
		
		// Check if file is a log file and is old
		if filepath.Ext(file.Name()) == ".log" {
			info, err := file.Info()
			if err != nil {
				continue
			}
			
			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(logDir, file.Name()))
			}
		}
	}
}

// write writes a log message
func (l *Logger) write(level int, format string, v ...interface{}) {
	if level < l.level {
		return
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.logger == nil {
		return
	}
	
	// Check for rotation
	l.rotateIfNeeded()
	
	// Format message with level prefix
	levelStr := getLevelString(level)
	msg := fmt.Sprintf(format, v...)
	fullMsg := fmt.Sprintf("[%s] %s", levelStr, msg)
	
	// Write to log file
	l.logger.Output(2, fullMsg)
	
	// Update size estimate
	l.currentSize += int64(len(fullMsg)) + 1
}

// getLevelString returns the string representation of a log level
func getLevelString(level int) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Public logging methods

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.write(DEBUG, format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.write(INFO, format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.write(WARN, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.write(ERROR, format, v...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.write(FATAL, format, v...)
	os.Exit(1)
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Close closes the logger
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetLogPath returns the current log file path
func (l *Logger) GetLogPath() string {
	return l.logPath
}

// Package-level convenience functions

// Debug logs a debug message using the global logger
func Debug(format string, v ...interface{}) {
	GetLogger().Debug(format, v...)
}

// Info logs an info message using the global logger
func Info(format string, v ...interface{}) {
	GetLogger().Info(format, v...)
}

// Warn logs a warning message using the global logger
func Warn(format string, v ...interface{}) {
	GetLogger().Warn(format, v...)
}

// Error logs an error message using the global logger
func Error(format string, v ...interface{}) {
	GetLogger().Error(format, v...)
}

// Fatal logs a fatal message using the global logger and exits
func Fatal(format string, v ...interface{}) {
	GetLogger().Fatal(format, v...)
}

// Printf is a compatibility function that logs at INFO level
func Printf(format string, v ...interface{}) {
	GetLogger().Info(format, v...)
}

// Println is a compatibility function that logs at INFO level
func Println(v ...interface{}) {
	GetLogger().Info(fmt.Sprint(v...))
}

// Writer returns an io.Writer for the logger (useful for redirecting standard log)
func Writer() io.Writer {
	return &logWriter{logger: GetLogger()}
}

// logWriter implements io.Writer for the logger
type logWriter struct {
	logger *Logger
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}

// RedirectStandardLog redirects the standard log package to use our logger
func RedirectStandardLog() {
	log.SetOutput(Writer())
	log.SetFlags(0) // Remove standard log flags as we handle them
}