package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// SensitiveDataMasker 敏感数据遮蔽器
type SensitiveDataMasker struct {
	sensitivePatterns []*regexp.Regexp
}

// DefaultMaskerConfig 返回默认遮蔽器配置
func DefaultMaskerConfig() *SensitiveDataMasker {
	return NewSensitiveDataMasker()
}

// NewSensitiveDataMasker 创建新的敏感数据遮蔽器
func NewSensitiveDataMasker() *SensitiveDataMasker {
	patterns := []*regexp.Regexp{
		// App Secret 模式
		regexp.MustCompile(`(?i)(app[_-]?secret["'\s]*[:=]["'\s]*)([a-zA-Z0-9]{20,})`),
		// Access Token 模式
		regexp.MustCompile(`(?i)(access[_-]?token["'\s]*[:=]["'\s]*)([a-zA-Z0-9_-]{20,})`),
		// API Key 模式
		regexp.MustCompile(`(?i)(api[_-]?key["'\s]*[:=]["'\s]*)([a-zA-Z0-9]{20,})`),
		// 密码模式
		regexp.MustCompile(`(?i)(password["'\s]*[:=]["'\s]*)([^\s"']{6,})`),
		// 通用密钥模式
		regexp.MustCompile(`(?i)(secret["'\s]*[:=]["'\s]*)([a-zA-Z0-9]{16,})`),
	}

	return &SensitiveDataMasker{
		sensitivePatterns: patterns,
	}
}

// MaskSensitiveData 遮蔽敏感数据
func (m *SensitiveDataMasker) MaskSensitiveData(data string) string {
	masked := data

	for _, pattern := range m.sensitivePatterns {
		masked = pattern.ReplaceAllStringFunc(masked, func(match string) string {
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				prefix := submatches[1]
				sensitiveValue := submatches[2]

				// 保留前2个和后2个字符，中间用*替代
				if len(sensitiveValue) > 4 {
					maskedValue := sensitiveValue[:2] + strings.Repeat("*", len(sensitiveValue)-4) + sensitiveValue[len(sensitiveValue)-2:]
					return prefix + maskedValue
				} else {
					return prefix + strings.Repeat("*", len(sensitiveValue))
				}
			}
			return match
		})
	}

	return masked
}

// SQLInjectionValidator SQL注入验证器
type SQLInjectionValidator struct {
	dangerousPatterns []*regexp.Regexp
}

// NewSQLInjectionValidator 创建新的SQL注入验证器
func NewSQLInjectionValidator() *SQLInjectionValidator {
	patterns := []*regexp.Regexp{
		// 危险的SQL关键字组合
		regexp.MustCompile(`(?i)(union\s+select|drop\s+table|delete\s+from|insert\s+into|update\s+set)`),
		// 注释符号
		regexp.MustCompile(`(--|/\*|\*/|#)`),
		// 字符串拼接攻击
		regexp.MustCompile(`('\s*\+\s*'|"\s*\+\s*")`),
		// 布尔盲注
		regexp.MustCompile(`(?i)(\s+or\s+1\s*=\s*1|\s+and\s+1\s*=\s*1)`),
		// 时间盲注
		regexp.MustCompile(`(?i)(sleep\s*\(|waitfor\s+delay|benchmark\s*\()`),
		// 信息泄露
		regexp.MustCompile(`(?i)(information_schema|sys\.|mysql\.|pg_)`),
	}

	return &SQLInjectionValidator{
		dangerousPatterns: patterns,
	}
}

// ValidateSQL 验证SQL语句是否包含注入攻击
func (v *SQLInjectionValidator) ValidateSQL(sql string) error {
	// 移除多余的空白字符
	cleanSQL := strings.TrimSpace(sql)

	for _, pattern := range v.dangerousPatterns {
		if pattern.MatchString(cleanSQL) {
			return fmt.Errorf("检测到潜在的SQL注入攻击: %s", pattern.String())
		}
	}

	return nil
}

// InputSanitizer 输入清理器
type InputSanitizer struct{}

// NewInputSanitizer 创建新的输入清理器
func NewInputSanitizer() *InputSanitizer {
	return &InputSanitizer{}
}

// SanitizeTableName 清理表名
func (s *InputSanitizer) SanitizeTableName(tableName string) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("表名不能为空")
	}

	// 检查长度
	if len(tableName) > 100 {
		return "", fmt.Errorf("表名长度不能超过100个字符")
	}

	// 检查字符合法性
	for _, r := range tableName {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return "", fmt.Errorf("表名只能包含字母、数字、下划线和连字符")
		}
	}

	// 检查是否以字母开头
	if !unicode.IsLetter(rune(tableName[0])) {
		return "", fmt.Errorf("表名必须以字母开头")
	}

	return tableName, nil
}

// SanitizeFieldName 清理字段名
func (s *InputSanitizer) SanitizeFieldName(fieldName string) (string, error) {
	if fieldName == "" {
		return "", fmt.Errorf("字段名不能为空")
	}

	// 检查长度
	if len(fieldName) > 100 {
		return "", fmt.Errorf("字段名长度不能超过100个字符")
	}

	// 检查字符合法性
	for _, r := range fieldName {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return "", fmt.Errorf("字段名只能包含字母、数字、下划线和连字符")
		}
	}

	// 检查是否以字母开头
	if !unicode.IsLetter(rune(fieldName[0])) {
		return "", fmt.Errorf("字段名必须以字母开头")
	}

	return fieldName, nil
}

// SecureCompare 安全比较两个字符串，防止时序攻击
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateSecureToken 生成安全的随机令牌
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("令牌长度必须大于0")
	}

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("生成随机令牌失败: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// ValidateAppCredentials 验证应用凭据格式
func ValidateAppCredentials(appID, appSecret string) error {
	if appID == "" {
		return fmt.Errorf("App ID 不能为空")
	}

	if appSecret == "" {
		return fmt.Errorf("App Secret 不能为空")
	}

	// 检查App ID格式（通常以cli_开头）
	if !strings.HasPrefix(appID, "cli_") {
		return fmt.Errorf("App ID 格式不正确，应以 'cli_' 开头")
	}

	// 检查App Secret长度（通常为32个字符）
	if len(appSecret) < 20 {
		return fmt.Errorf("App Secret 长度不足，至少需要20个字符")
	}

	return nil
}
