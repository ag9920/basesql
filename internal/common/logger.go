package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry 日志条目结构
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger 结构化日志器
type Logger struct {
	level      LogLevel
	output     *os.File
	structured bool
	fields     map[string]interface{}
}

// DefaultLogger 默认日志器实例
var DefaultLogger *Logger

// init 初始化默认日志器
func init() {
	DefaultLogger = NewLogger(LogLevelInfo, false)
}

// NewLogger 创建新的日志器
func NewLogger(level LogLevel, structured bool) *Logger {
	return &Logger{
		level:      level,
		output:     os.Stdout,
		structured: structured,
		fields:     make(map[string]interface{}),
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput 设置输出目标
func (l *Logger) SetOutput(output *os.File) {
	l.output = output
}

// SetStructured 设置是否使用结构化输出
func (l *Logger) SetStructured(structured bool) {
	l.structured = structured
}

// WithField 添加字段
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:      l.level,
		output:     l.output,
		structured: l.structured,
		fields:     make(map[string]interface{}),
	}

	// 复制现有字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	newLogger.fields[key] = value
	return newLogger
}

// WithFields 添加多个字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:      l.level,
		output:     l.output,
		structured: l.structured,
		fields:     make(map[string]interface{}),
	}

	// 复制现有字段
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithError 添加错误字段
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// log 内部日志方法
func (l *Logger) log(level LogLevel, message string, err error) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    l.fields,
	}

	// 添加调用者信息
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// 添加错误信息
	if err != nil {
		entry.Error = err.Error()
	}

	if l.structured {
		// 结构化输出（JSON）
		if data, err := json.Marshal(entry); err == nil {
			fmt.Fprintln(l.output, string(data))
		}
	} else {
		// 人类可读输出
		l.formatHumanReadable(entry)
	}
}

// formatHumanReadable 格式化为人类可读的输出
func (l *Logger) formatHumanReadable(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	levelStr := entry.Level.String()

	// 根据级别选择颜色和图标
	var icon, color string
	switch entry.Level {
	case LogLevelDebug:
		icon = "🔍"
		color = "\033[36m" // 青色
	case LogLevelInfo:
		icon = "ℹ️"
		color = "\033[32m" // 绿色
	case LogLevelWarn:
		icon = "⚠️"
		color = "\033[33m" // 黄色
	case LogLevelError:
		icon = "❌"
		color = "\033[31m" // 红色
	case LogLevelFatal:
		icon = "💀"
		color = "\033[35m" // 紫色
	default:
		icon = "📝"
		color = "\033[0m" // 默认
	}

	reset := "\033[0m"

	// 基本信息
	var parts []string
	parts = append(parts, fmt.Sprintf("%s[%s]%s", color, timestamp, reset))
	parts = append(parts, fmt.Sprintf("%s %s[%s]%s", icon, color, levelStr, reset))

	// 调用者信息
	if entry.Caller != "" {
		parts = append(parts, fmt.Sprintf("[%s]", entry.Caller))
	}

	// 消息
	parts = append(parts, entry.Message)

	// 字段信息
	if len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("[%s]", strings.Join(fieldParts, ", ")))
	}

	// 错误信息
	if entry.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%s", entry.Error))
	}

	fmt.Fprintln(l.output, strings.Join(parts, " "))
}

// Debug 记录调试信息
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message, nil)
}

// Debugf 记录格式化的调试信息
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LogLevelDebug, fmt.Sprintf(format, args...), nil)
}

// Info 记录信息
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message, nil)
}

// Infof 记录格式化的信息
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LogLevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warn 记录警告信息
func (l *Logger) Warn(message string) {
	l.log(LogLevelWarn, message, nil)
}

// Warnf 记录格式化的警告信息
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LogLevelWarn, fmt.Sprintf(format, args...), nil)
}

// Error 记录错误信息
func (l *Logger) Error(message string) {
	l.log(LogLevelError, message, nil)
}

// Errorf 记录格式化的错误信息
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LogLevelError, fmt.Sprintf(format, args...), nil)
}

// ErrorWithErr 记录带错误对象的错误信息
func (l *Logger) ErrorWithErr(message string, err error) {
	l.log(LogLevelError, message, err)
}

// Fatal 记录致命错误信息并退出程序
func (l *Logger) Fatal(message string) {
	l.log(LogLevelFatal, message, nil)
	os.Exit(1)
}

// Fatalf 记录格式化的致命错误信息并退出程序
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(LogLevelFatal, fmt.Sprintf(format, args...), nil)
	os.Exit(1)
}

// FatalWithErr 记录带错误对象的致命错误信息并退出程序
func (l *Logger) FatalWithErr(message string, err error) {
	l.log(LogLevelFatal, message, err)
	os.Exit(1)
}

// 全局日志函数

// SetLogLevel 设置全局日志级别
func SetLogLevel(level LogLevel) {
	DefaultLogger.SetLevel(level)
}

// SetLogOutput 设置全局日志输出
func SetLogOutput(output *os.File) {
	DefaultLogger.SetOutput(output)
}

// SetStructuredLogging 设置全局结构化日志
func SetStructuredLogging(structured bool) {
	DefaultLogger.SetStructured(structured)
}

// Debug 全局调试日志
func Debug(message string) {
	DefaultLogger.Debug(message)
}

// Debugf 全局格式化调试日志
func Debugf(format string, args ...interface{}) {
	DefaultLogger.Debugf(format, args...)
}

// Info 全局信息日志
func Info(message string) {
	DefaultLogger.Info(message)
}

// Infof 全局格式化信息日志
func Infof(format string, args ...interface{}) {
	DefaultLogger.Infof(format, args...)
}

// Warn 全局警告日志
func Warn(message string) {
	DefaultLogger.Warn(message)
}

// Warnf 全局格式化警告日志
func Warnf(format string, args ...interface{}) {
	DefaultLogger.Warnf(format, args...)
}

// Error 全局错误日志
func Error(message string) {
	DefaultLogger.Error(message)
}

// Errorf 全局格式化错误日志
func Errorf(format string, args ...interface{}) {
	DefaultLogger.Errorf(format, args...)
}

// ErrorWithErr 全局带错误对象的错误日志
func ErrorWithErr(message string, err error) {
	DefaultLogger.ErrorWithErr(message, err)
}

// Fatal 全局致命错误日志
func Fatal(message string) {
	DefaultLogger.Fatal(message)
}

// Fatalf 全局格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	DefaultLogger.Fatalf(format, args...)
}

// FatalWithErr 全局带错误对象的致命错误日志
func FatalWithErr(message string, err error) {
	DefaultLogger.FatalWithErr(message, err)
}

// LogSQLExecution 记录SQL执行日志
func LogSQLExecution(sql string, duration time.Duration, err error) {
	logger := DefaultLogger.WithFields(map[string]interface{}{
		"sql":      sql,
		"duration": duration.String(),
	})

	if err != nil {
		logger.WithError(err).Error("SQL execution failed")
	} else {
		logger.Info("SQL execution completed")
	}
}

// LogAPIRequest 记录API请求日志
func LogAPIRequest(method, url string, statusCode int, duration time.Duration, err error) {
	logger := DefaultLogger.WithFields(map[string]interface{}{
		"method":      method,
		"url":         url,
		"status_code": statusCode,
		"duration":    duration.String(),
	})

	if err != nil {
		logger.WithError(err).Error("API request failed")
	} else if statusCode >= 400 {
		logger.Warn("API request completed with error status")
	} else {
		logger.Info("API request completed successfully")
	}
}

// LogPerformanceMetrics 记录性能指标
func LogPerformanceMetrics(operation string, metrics map[string]interface{}) {
	logger := DefaultLogger.WithField("operation", operation).WithFields(metrics)
	logger.Info("Performance metrics")
}

// InitializeLogging 初始化日志系统
func InitializeLogging(debug bool, logFile string) error {
	// 设置日志级别
	if debug {
		SetLogLevel(LogLevelDebug)
		SetStructuredLogging(false) // 调试模式使用人类可读格式
	} else {
		SetLogLevel(LogLevelInfo)
		SetStructuredLogging(false) // 默认使用人类可读格式
	}

	// 设置日志文件
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		SetLogOutput(file)
		SetStructuredLogging(true) // 文件输出使用结构化格式
	}

	return nil
}
