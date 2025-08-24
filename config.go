package basesql

import (
	"time"

	"github.com/ag9920/basesql/internal/common"
	"github.com/ag9920/basesql/internal/security"
)

// AuthType 认证类型
type AuthType string

const (
	// AuthTypeTenant 应用认证（tenant_access_token）
	AuthTypeTenant AuthType = "tenant"
	// AuthTypeUser 用户认证（user_access_token）
	AuthTypeUser AuthType = "user"
)

// Config 飞书多维表格配置
type Config struct {
	// 飞书应用配置
	AppID     string `json:"app_id"`     // 应用 ID
	AppSecret string `json:"app_secret"` // 应用密钥
	BaseURL   string `json:"base_url"`   // API 基础 URL，默认 https://open.feishu.cn

	// 认证配置
	AuthType    AuthType `json:"auth_type"`    // 认证类型
	AccessToken string   `json:"access_token"` // 用户访问令牌（AuthTypeUser 时使用）

	// 多维表格配置
	AppToken string `json:"app_token"` // 多维表格的 app_token
	TableID  string `json:"table_id"`  // 默认表 ID（可选）

	// 连接配置
	Timeout         time.Duration `json:"timeout"`          // 请求超时时间
	MaxRetries      int           `json:"max_retries"`      // 最大重试次数
	RetryInterval   time.Duration `json:"retry_interval"`   // 重试间隔
	RateLimitQPS    int           `json:"rate_limit_qps"`   // 每秒请求限制
	BatchSize       int           `json:"batch_size"`       // 批量操作大小
	CacheEnabled    bool          `json:"cache_enabled"`    // 是否启用缓存
	CacheTTL        time.Duration `json:"cache_ttl"`        // 缓存过期时间
	DebugMode       bool          `json:"debug_mode"`       // 调试模式
	ConsistencyMode bool          `json:"consistency_mode"` // 一致性模式
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		BaseURL:         common.DefaultBaseURL,
		AuthType:        AuthTypeTenant,
		Timeout:         common.DefaultTimeout,
		MaxRetries:      common.DefaultMaxRetries,
		RetryInterval:   1 * time.Second,
		RateLimitQPS:    common.DefaultRateLimitQPS, // 飞书 API 限制 50 次/秒
		BatchSize:       common.DefaultBatchSize,
		CacheEnabled:    true,
		CacheTTL:        5 * time.Minute,
		DebugMode:       false,
		ConsistencyMode: false,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证应用凭据
	if err := security.ValidateAppCredentials(c.AppID, c.AppSecret); err != nil {
		return ErrInvalidConfig(err.Error())
	}

	if c.AppToken == "" {
		return ErrInvalidConfig("app_token is required")
	}
	if c.AuthType == AuthTypeUser && c.AccessToken == "" {
		return ErrInvalidConfig("access_token is required for user auth type")
	}
	return nil
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}
