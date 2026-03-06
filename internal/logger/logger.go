package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type Level int

const (
	Debug Level = iota
	Info
	Warning
	Error
	Fatal
)

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warning:
		return "WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	*log.Logger
	level  Level
	output io.Writer
}

type Config struct {
	Level      Level
	FilePath   string
	MaxSize    int64
	UseConsole bool
}

func New(config Config) (*Logger, error) {
	if config.FilePath != "" {
		logDir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	var writer io.Writer = file
	if config.UseConsole {
		writer = io.MultiWriter(file, os.Stderr)
	}

	logger := &Logger{
		Logger: log.New(writer, "", 0),
		level:  config.Level,
		output: writer,
	}

	return logger, nil
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)

	// Format: [TIMESTAMP] [LEVEL] [FILE:LINE] MESSAGE
	l.Logger.Printf("[%s] [%s] [%s:%d] %s", timestamp, level.String(), file, line, msg)

	if level == Fatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(Debug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(Info, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(Warning, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(Error, format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(Fatal, format, args...)
}

// Close closes the logger's underlying file
func (l *Logger) Close() error {
	if closer, ok := l.output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
