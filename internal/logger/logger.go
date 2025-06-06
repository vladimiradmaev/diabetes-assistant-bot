package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	infoLogger    *log.Logger
	errorLogger   *log.Logger
	debugLogger   *log.Logger
	warningLogger *log.Logger
	file          *os.File
}

var GlobalLogger *Logger

func Init() error {
	// Create logs directory if it doesn't exist
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logFileName := filepath.Join(logsDir, fmt.Sprintf("diabetes-helper-%s.log", timestamp))

	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer to write to both file and console
	multiWriter := io.MultiWriter(os.Stdout, file)

	GlobalLogger = &Logger{
		infoLogger:    log.New(multiWriter, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger:   log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger:   log.New(multiWriter, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
		warningLogger: log.New(multiWriter, "[WARNING] ", log.Ldate|log.Ltime|log.Lshortfile),
		file:          file,
	}

	GlobalLogger.Info("Logger initialized successfully")
	return nil
}

func (l *Logger) Info(v ...interface{}) {
	l.infoLogger.Println(v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.errorLogger.Println(v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

func (l *Logger) Debug(v ...interface{}) {
	l.debugLogger.Println(v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.debugLogger.Printf(format, v...)
}

func (l *Logger) Warning(v ...interface{}) {
	l.warningLogger.Println(v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.warningLogger.Printf(format, v...)
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Convenience functions for global logger
func Info(v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Info(v...)
	}
}

func Infof(format string, v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Infof(format, v...)
	}
}

func Error(v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Error(v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Errorf(format, v...)
	}
}

func Debug(v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Debug(v...)
	}
}

func Debugf(format string, v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Debugf(format, v...)
	}
}

func Warning(v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Warning(v...)
	}
}

func Warningf(format string, v ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Warningf(format, v...)
	}
}

func Close() error {
	if GlobalLogger != nil {
		return GlobalLogger.Close()
	}
	return nil
}
