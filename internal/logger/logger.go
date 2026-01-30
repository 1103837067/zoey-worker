// Package logger 提供统一的日志工具
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 解析日志级别字符串
func ParseLevel(s string) Level {
	switch s {
	case "DEBUG", "debug":
		return DEBUG
	case "INFO", "info":
		return INFO
	case "WARN", "warn", "WARNING", "warning":
		return WARN
	case "ERROR", "error":
		return ERROR
	default:
		return INFO
	}
}

// Logger 日志记录器
type Logger struct {
	mu       sync.Mutex
	level    Level
	enabled  bool
	console  bool
	file     bool
	filePath string
	logger   *log.Logger
	fileOut  *os.File
}

// 全局默认 logger
var defaultLogger = New()

// New 创建新的 Logger 实例
func New() *Logger {
	return &Logger{
		level:   INFO,
		enabled: true,
		console: true,
		file:    false,
		logger:  log.New(os.Stdout, "", 0),
	}
}

// Default 获取默认 logger
func Default() *Logger {
	return defaultLogger
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetEnabled 设置是否启用日志
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// SetConsole 设置是否输出到控制台
func (l *Logger) SetConsole(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.console = enabled
	l.updateOutput()
}

// SetFile 设置是否输出到文件
func (l *Logger) SetFile(enabled bool, path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 关闭旧文件
	if l.fileOut != nil {
		l.fileOut.Close()
		l.fileOut = nil
	}

	l.file = enabled
	l.filePath = path

	if enabled && path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("无法打开日志文件: %w", err)
		}
		l.fileOut = f
	}

	l.updateOutput()
	return nil
}

func (l *Logger) updateOutput() {
	var writers []io.Writer

	if l.console {
		writers = append(writers, os.Stdout)
	}
	if l.file && l.fileOut != nil {
		writers = append(writers, l.fileOut)
	}

	if len(writers) == 0 {
		l.logger.SetOutput(io.Discard)
	} else if len(writers) == 1 {
		l.logger.SetOutput(writers[0])
	} else {
		l.logger.SetOutput(io.MultiWriter(writers...))
	}
}

// log 内部日志方法
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if !l.enabled || level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("%s | %-5s | %s", timestamp, level.String(), msg)
}

// Debug 输出 DEBUG 级别日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 输出 INFO 级别日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 输出 WARN 级别日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 输出 ERROR 级别日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// LogEvent 记录带分类的事件日志
func (l *Logger) LogEvent(category string, ok bool, elapsedMs float64, detail string) {
	status := "OK"
	if !ok {
		status = "NG"
	}

	if ok {
		l.Info("%-4s | %s | %6.1fms | %s", category, status, elapsedMs, detail)
	} else {
		l.Error("%-4s | %s | %6.1fms | %s", category, status, elapsedMs, detail)
	}
}

// Close 关闭 logger，释放资源
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileOut != nil {
		err := l.fileOut.Close()
		l.fileOut = nil
		return err
	}
	return nil
}

// 包级别便捷函数
func Debug(format string, args ...interface{}) { defaultLogger.Debug(format, args...) }
func Info(format string, args ...interface{})  { defaultLogger.Info(format, args...) }
func Warn(format string, args ...interface{})  { defaultLogger.Warn(format, args...) }
func Error(format string, args ...interface{}) { defaultLogger.Error(format, args...) }
func LogEvent(category string, ok bool, elapsedMs float64, detail string) {
	defaultLogger.LogEvent(category, ok, elapsedMs, detail)
}
