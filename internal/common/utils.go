package common

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// ValidationError 验证错误类型
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s%s", e.Field, e.Message)
}

// NewValidationError 创建验证错误
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ValidateNotEmpty 验证字符串不为空
// 参数:
//   - value: 要验证的值
//   - fieldName: 字段名称
//
// 返回:
//   - error: 如果为空返回验证错误，否则返回 nil
func ValidateNotEmpty(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return NewValidationError(fieldName, "不能为空")
	}
	return nil
}

// ValidateNotNil 验证指针不为空
// 参数:
//   - value: 要验证的指针
//   - fieldName: 字段名称
//
// 返回:
//   - error: 如果为空返回验证错误，否则返回 nil
func ValidateNotNil(value interface{}, fieldName string) error {
	if value == nil {
		return NewValidationError(fieldName, "不能为空")
	}
	return nil
}

// ValidateSliceNotEmpty 验证切片不为空
// 参数:
//   - slice: 要验证的切片
//   - fieldName: 字段名称
//
// 返回:
//   - error: 如果为空返回验证错误，否则返回 nil
func ValidateSliceNotEmpty(slice []interface{}, fieldName string) error {
	if len(slice) == 0 {
		return NewValidationError(fieldName, "不能为空")
	}
	return nil
}

// ValidateMapNotEmpty 验证映射不为空
// 参数:
//   - m: 要验证的映射
//   - fieldName: 字段名称
//
// 返回:
//   - error: 如果为空返回验证错误，否则返回 nil
func ValidateMapNotEmpty(m map[string]interface{}, fieldName string) error {
	if len(m) == 0 {
		return NewValidationError(fieldName, "不能为空")
	}
	return nil
}

// GetEnv 获取环境变量，如果不存在则返回默认值
// 参数:
//   - key: 环境变量名
//   - defaultValue: 默认值
//
// 返回:
//   - string: 环境变量值或默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetConfigValue 获取配置值，优先使用提供的值，否则从环境变量获取
// 参数:
//   - value: 配置值
//   - envKey: 环境变量名
//
// 返回:
//   - string: 最终的配置值
func GetConfigValue(value, envKey string) string {
	if value != "" {
		return value
	}
	return GetEnv(envKey, "")
}

// GetStringValue 安全地从 map 中获取字符串值
// 参数:
//   - m: 源映射
//   - key: 键名
//
// 返回:
//   - string: 字符串值，如果不存在或类型不匹配返回空字符串
func GetStringValue(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// GetDisplayWidth 计算字符串的显示宽度（中文字符占2个宽度）
// 参数:
//   - s: 要计算的字符串
//
// 返回:
//   - int: 显示宽度
func GetDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		if utf8.RuneLen(r) > 1 {
			// 中文字符占2个宽度
			width += 2
		} else {
			// 英文字符占1个宽度
			width += 1
		}
	}
	return width
}

// TruncateString 截断字符串到指定显示宽度
// 参数:
//   - s: 要截断的字符串
//   - maxLen: 最大显示宽度
//
// 返回:
//   - string: 截断后的字符串
func TruncateString(s string, maxLen int) string {
	if GetDisplayWidth(s) <= maxLen {
		return s
	}

	result := ""
	currentWidth := 0

	for _, r := range s {
		runeWidth := 1
		if utf8.RuneLen(r) > 1 {
			runeWidth = 2
		}

		if currentWidth+runeWidth > maxLen-3 { // 为 "..." 预留空间
			break
		}

		result += string(r)
		currentWidth += runeWidth
	}

	return result + "..."
}

// PadString 填充字符串到指定显示宽度
// 参数:
//   - s: 要填充的字符串
//   - width: 目标显示宽度
//
// 返回:
//   - string: 填充后的字符串
func PadString(s string, width int) string {
	displayWidth := GetDisplayWidth(s)
	if displayWidth >= width {
		return s
	}

	padding := width - displayWidth
	return s + strings.Repeat(" ", padding)
}

// IsValidIdentifier 验证标识符格式（字母开头，只包含字母、数字、下划线）
// 参数:
//   - name: 要验证的标识符
//
// 返回:
//   - bool: 是否为有效标识符
func IsValidIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// 检查首字符是否为字母
	firstChar := rune(name[0])
	if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z')) {
		return false
	}

	// 检查其余字符是否为字母、数字或下划线
	for _, char := range name[1:] {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}

	return true
}

// MaskSensitive 遮蔽敏感信息
// 参数:
//   - value: 敏感值
//
// 返回:
//   - string: 遮蔽后的值
func MaskSensitive(value string) string {
	if value == "" {
		return "未设置"
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

// PreprocessSQL 预处理 SQL 语句
// 参数:
//   - sql: 原始 SQL 语句
//
// 返回:
//   - string: 预处理后的 SQL 语句
func PreprocessSQL(sql string) string {
	// 去除首尾空白
	sql = strings.TrimSpace(sql)
	// 移除末尾的分号
	sql = strings.TrimSuffix(sql, ";")
	return sql
}

// CreateTimeoutContext 创建带超时的上下文
// 参数:
//   - timeout: 超时时间
//
// 返回:
//   - context.Context: 上下文
//   - context.CancelFunc: 取消函数
func CreateTimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// FormatError 格式化错误信息
// 参数:
//   - operation: 操作名称
//   - err: 原始错误
//
// 返回:
//   - error: 格式化后的错误
func FormatError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s失败: %w", operation, err)
}

// SafeStringSlice 安全地将 interface{} 切片转换为字符串切片
// 参数:
//   - slice: 源切片
//
// 返回:
//   - []string: 字符串切片
func SafeStringSlice(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			result[i] = str
		} else {
			result[i] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// Contains 检查字符串切片是否包含指定值
// 参数:
//   - slice: 字符串切片
//   - item: 要查找的项
//
// 返回:
//   - bool: 是否包含
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates 去除字符串切片中的重复项
// 参数:
//   - slice: 源切片
//
// 返回:
//   - []string: 去重后的切片
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// FormatValue 格式化值用于显示
// 参数:
//   - value: 要格式化的值
//
// 返回:
//   - string: 格式化后的字符串
func FormatValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []string:
		return strings.Join(v, ", ")
	case []interface{}:
		var strs []string
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if text, exists := itemMap["text"]; exists {
					strs = append(strs, fmt.Sprintf("%v", text))
				} else if name, exists := itemMap["name"]; exists {
					strs = append(strs, fmt.Sprintf("%v", name))
				} else {
					strs = append(strs, fmt.Sprintf("%v", item))
				}
			} else {
				strs = append(strs, fmt.Sprintf("%v", item))
			}
		}
		return strings.Join(strs, ", ")
	case map[string]interface{}:
		if text, exists := v["text"]; exists {
			return fmt.Sprintf("%v", text)
		} else if name, exists := v["name"]; exists {
			return fmt.Sprintf("%v", name)
		}
		return fmt.Sprintf("%v", v)
	case float64:
		// 检查是否是时间戳（毫秒）
		if v > MillisecondThreshold { // 大于这个值可能是毫秒时间戳
			t := time.Unix(int64(v)/1000, 0)
			return t.Format("2006-01-02 15:04:05")
		}
		// 如果是整数，不显示小数点
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%g", v)
	case int64:
		// 检查是否是时间戳（毫秒）
		if v > MillisecondThreshold { // 大于这个值可能是毫秒时间戳
			t := time.Unix(v/1000, 0)
			return t.Format("2006-01-02 15:04:05")
		}
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// ValidateStatement 验证 GORM 语句的有效性
// 参数:
//   - stmt: GORM 语句
//
// 返回:
//   - error: 验证错误
func ValidateStatement(stmt *gorm.Statement) error {
	if stmt == nil {
		return fmt.Errorf("GORM 语句不能为 nil")
	}
	if stmt.Table == "" {
		return fmt.Errorf("表名不能为空")
	}
	return nil
}

// GetRecordID 从 GORM 语句中获取记录 ID
// 参数:
//   - ctx: 上下文
//   - stmt: GORM 语句
//
// 返回:
//   - string: 记录 ID
//   - error: 获取错误
func GetRecordID(ctx context.Context, stmt *gorm.Statement) (string, error) {
	// 检查是否有主键字段
	if stmt.Schema.PrioritizedPrimaryField == nil {
		return "", fmt.Errorf("表 %s 没有定义主键字段", stmt.Table)
	}

	recordIDValue, ok := stmt.Schema.PrioritizedPrimaryField.ValueOf(ctx, stmt.ReflectValue)
	if !ok || recordIDValue == nil {
		return "", fmt.Errorf("无法获取记录 ID")
	}
	return fmt.Sprintf("%v", recordIDValue), nil
}
