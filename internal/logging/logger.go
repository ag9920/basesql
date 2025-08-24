package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ag9920/basesql/internal/security"
)

// LogLevel 日志级别
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// StructuredLogger 结构化日志器
type StructuredLogger struct {
	level         LogLevel
	output        io.Writer
	formatter     Formatter
	maskSensitive *security.SensitiveDataMasker
	mutex         sync.Mutex
	fields        map[string]interface{}
}

// Formatter 日志格式化器接口
type Formatter interface {
	Format(entry *LogEntry) ([]byte, error)
}

// JSONFormatter JSON格式化器
type JSONFormatter struct{}

// Format 格式化日志条目为JSON
func (f *JSONFormatter) Format(entry *LogEntry) ([]byte, error) {
	return json.Marshal(entry)
}

// TextFormatter 文本格式化器
type TextFormatter struct {
	TimestampFormat string
	ColorEnabled    bool
}

// Format 格式化日志条目为文本
func (f *TextFormatter) Format(entry *LogEntry) ([]byte, error) {
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "2006-01-02 15:04:05"
	}

	timestamp := entry.Timestamp.Format(timestampFormat)
	level := entry.Level.String()

	if f.ColorEnabled {
		level = f.colorizeLevel(entry.Level)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[%s] %s %s", timestamp, level, entry.Message))

	if entry.Error != "" {
		builder.WriteString(fmt.Sprintf(" error=%s", entry.Error))
	}

	if entry.Caller != "" {
		builder.WriteString(fmt.Sprintf(" caller=%s", entry.Caller))
	}

	if entry.TraceID != "" {
		builder.WriteString(fmt.Sprintf(" trace_id=%s", entry.TraceID))
	}

	for key, value := range entry.Fields {
		builder.WriteString(fmt.Sprintf(" %s=%v", key, value))
	}

	builder.WriteString("\n")
	return []byte(builder.String()), nil
}

// colorizeLevel 为日志级别添加颜色
func (f *TextFormatter) colorizeLevel(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "\033[36mDEBUG\033[0m" // 青色
	case LevelInfo:
		return "\033[32mINFO\033[0m" // 绿色
	case LevelWarn:
		return "\033[33mWARN\033[0m" // 黄色
	case LevelError:
		return "\033[31mERROR\033[0m" // 红色
	case LevelFatal:
		return "\033[35mFATAL\033[0m" // 紫色
	default:
		return level.String()
	}
}

// LoggerConfig 日志器配置
type LoggerConfig struct {
	Level         LogLevel  `json:"level"`
	Output        io.Writer `json:"-"`
	Format        string    `json:"format"` // "json" 或 "text"
	ColorEnabled  bool      `json:"color_enabled"`
	CallerEnabled bool      `json:"caller_enabled"`
	MaskSensitive bool      `json:"mask_sensitive"`
}

// DefaultLoggerConfig 返回默认日志器配置
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:         LevelInfo,
		Output:        os.Stdout,
		Format:        "text",
		ColorEnabled:  true,
		CallerEnabled: true,
		MaskSensitive: true,
	}
}

// NewStructuredLogger 创建新的结构化日志器
func NewStructuredLogger(config *LoggerConfig) *StructuredLogger {
	if config == nil {
		config = DefaultLoggerConfig()
	}

	var formatter Formatter
	switch config.Format {
	case "json":
		formatter = &JSONFormatter{}
	default:
		formatter = &TextFormatter{
			ColorEnabled: config.ColorEnabled,
		}
	}

	var maskSensitive *security.SensitiveDataMasker
	if config.MaskSensitive {
		maskSensitive = security.NewSensitiveDataMasker()
	}

	return &StructuredLogger{
		level:         config.Level,
		output:        config.Output,
		formatter:     formatter,
		maskSensitive: maskSensitive,
		fields:        make(map[string]interface{}),
	}
}

// WithField 添加字段
func (l *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		level:         l.level,
		formatter:     l.formatter,
		output:        l.output,
		maskSensitive: l.maskSensitive,
		fields:        make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields 添加多个字段
func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		level:         l.level,
		formatter:     l.formatter,
		output:        l.output,
		maskSensitive: l.maskSensitive,
		fields:        make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithError 添加错误信息
func (l *StructuredLogger) WithError(err error) *StructuredLogger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// WithTraceID 添加追踪ID
func (l *StructuredLogger) WithTraceID(traceID string) *StructuredLogger {
	return l.WithField("trace_id", traceID)
}

// log 内部日志方法
func (l *StructuredLogger) log(level LogLevel, message string, err error) {
	if level < l.level {
		return
	}

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}

	// 复制字段
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// 添加错误信息
	if err != nil {
		entry.Error = err.Error()
	}

	// 添加调用者信息
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.Caller = fmt.Sprintf("%s:%d", file, line)
	}

	// 添加追踪ID（如果存在）
	if traceID, exists := entry.Fields["trace_id"]; exists {
		if id, ok := traceID.(string); ok {
			entry.TraceID = id
			delete(entry.Fields, "trace_id") // 避免重复
		}
	}

	// 敏感数据遮蔽
	if l.maskSensitive != nil {
		entry.Message = l.maskSensitive.MaskSensitiveData(entry.Message)
		if entry.Error != "" {
			entry.Error = l.maskSensitive.MaskSensitiveData(entry.Error)
		}

		// 遮蔽字段中的敏感数据
		for k, v := range entry.Fields {
			if str, ok := v.(string); ok {
				entry.Fields[k] = l.maskSensitive.MaskSensitiveData(str)
			}
		}
	}

	// 格式化并输出
	data, err := l.formatter.Format(entry)
	if err != nil {
		// 如果格式化失败，使用简单的文本输出
		fallbackMsg := fmt.Sprintf("[%s] %s %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"), entry.Level.String(), entry.Message)
		data = []byte(fallbackMsg)
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.output.Write(data)
}

// Debug 记录调试日志
func (l *StructuredLogger) Debug(message string) {
	l.log(LevelDebug, message, nil)
}

// Debugf 记录格式化调试日志
func (l *StructuredLogger) Debugf(format string, args ...interface{}) {
	l.log(LevelDebug, fmt.Sprintf(format, args...), nil)
}

// Info 记录信息日志
func (l *StructuredLogger) Info(message string) {
	l.log(LevelInfo, message, nil)
}

// Infof 记录格式化信息日志
func (l *StructuredLogger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warn 记录警告日志
func (l *StructuredLogger) Warn(message string) {
	l.log(LevelWarn, message, nil)
}

// Warnf 记录格式化警告日志
func (l *StructuredLogger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, fmt.Sprintf(format, args...), nil)
}

// Error 记录错误日志
func (l *StructuredLogger) Error(message string) {
	l.log(LevelError, message, nil)
}

// Errorf 记录格式化错误日志
func (l *StructuredLogger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(format, args...), nil)
}

// ErrorWithErr 记录带错误对象的错误日志
func (l *StructuredLogger) ErrorWithErr(message string, err error) {
	l.log(LevelError, message, err)
}

// Fatal 记录致命错误日志并退出程序
func (l *StructuredLogger) Fatal(message string) {
	l.log(LevelFatal, message, nil)
	os.Exit(1)
}

// Fatalf 记录格式化致命错误日志并退出程序
func (l *StructuredLogger) Fatalf(format string, args ...interface{}) {
	l.log(LevelFatal, fmt.Sprintf(format, args...), nil)
	os.Exit(1)
}

// SetLevel 设置日志级别
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// GetLevel 获取当前日志级别
func (l *StructuredLogger) GetLevel() LogLevel {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.level
}

// SetOutput 设置输出目标
func (l *StructuredLogger) SetOutput(output io.Writer) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.output = output
}

// 全局日志器实例
var defaultLogger *StructuredLogger
var once sync.Once

// GetDefaultLogger 获取默认日志器
func GetDefaultLogger() *StructuredLogger {
	once.Do(func() {
		defaultLogger = NewStructuredLogger(DefaultLoggerConfig())
	})
	return defaultLogger
}

// SetDefaultLogger 设置默认日志器
func SetDefaultLogger(logger *StructuredLogger) {
	defaultLogger = logger
}

// 便捷的全局日志函数
func Debug(message string) {
	GetDefaultLogger().Debug(message)
}

func Debugf(format string, args ...interface{}) {
	GetDefaultLogger().Debugf(format, args...)
}

func Info(message string) {
	GetDefaultLogger().Info(message)
}

func Infof(format string, args ...interface{}) {
	GetDefaultLogger().Infof(format, args...)
}

func Warn(message string) {
	GetDefaultLogger().Warn(message)
}

func Warnf(format string, args ...interface{}) {
	GetDefaultLogger().Warnf(format, args...)
}

func Error(message string) {
	GetDefaultLogger().Error(message)
}

func Errorf(format string, args ...interface{}) {
	GetDefaultLogger().Errorf(format, args...)
}

func ErrorWithErr(message string, err error) {
	GetDefaultLogger().ErrorWithErr(message, err)
}

func Fatal(message string) {
	GetDefaultLogger().Fatal(message)
}

func Fatalf(format string, args ...interface{}) {
	GetDefaultLogger().Fatalf(format, args...)
}

func WithField(key string, value interface{}) *StructuredLogger {
	return GetDefaultLogger().WithField(key, value)
}

func WithFields(fields map[string]interface{}) *StructuredLogger {
	return GetDefaultLogger().WithFields(fields)
}

func WithError(err error) *StructuredLogger {
	return GetDefaultLogger().WithError(err)
}

func WithTraceID(traceID string) *StructuredLogger {
	return GetDefaultLogger().WithTraceID(traceID)
}
