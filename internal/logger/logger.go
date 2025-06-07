package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

var globalLogger *slog.Logger

// LogLevel represents different log levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Config holds logger configuration
type Config struct {
	Level      LogLevel
	OutputPath string
	Format     string // "json" or "text"
}

// Init initializes the structured logger
func Init() error {
	return InitWithConfig(Config{
		Level:      LevelInfo,
		OutputPath: "logs/app.log",
		Format:     "json",
	})
}

// InitWithConfig initializes logger with custom config
func InitWithConfig(config Config) error {
	// Create logs directory if it doesn't exist
	if config.OutputPath != "" {
		dir := filepath.Dir(config.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Configure output
	var output *os.File
	var err error
	if config.OutputPath == "" || config.OutputPath == "stdout" {
		output = os.Stdout
	} else {
		output, err = os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
	}

	// Convert log level
	var level slog.Level
	switch config.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	if config.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)

	return nil
}

// Close closes the logger (for compatibility)
func Close() error {
	// slog doesn't need explicit closing
	return nil
}

// WithContext returns a logger with context values
func WithContext(ctx context.Context) *slog.Logger {
	return globalLogger
}

// WithFields returns a logger with additional fields
func WithFields(fields ...any) *slog.Logger {
	return globalLogger.With(fields...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	globalLogger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	globalLogger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	globalLogger.Warn(msg, args...)
}

// Warning logs a warning message (for compatibility)
func Warning(msg string, args ...any) {
	globalLogger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	globalLogger.Error(msg, args...)
}

// Infof logs an info message with formatting
func Infof(format string, args ...any) {
	globalLogger.Info(fmt.Sprintf(format, args...))
}

// Warningf logs a warning message with formatting
func Warningf(format string, args ...any) {
	globalLogger.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs an error message with formatting
func Errorf(format string, args ...any) {
	globalLogger.Error(fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits
func Fatal(msg string, args ...any) {
	globalLogger.Error(msg, args...)
	os.Exit(1)
}

// Fatalf logs a fatal message with formatting and exits
func Fatalf(format string, args ...any) {
	globalLogger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// GetLogger returns the global logger instance
func GetLogger() *slog.Logger {
	return globalLogger
}
