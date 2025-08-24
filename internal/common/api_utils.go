package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// APIRequest API 请求结构体
type APIRequest struct {
	// Method HTTP 方法 (GET, POST, PUT, DELETE)
	Method string `json:"method"`
	// Path API 路径
	Path string `json:"path"`
	// Body 请求体
	Body interface{} `json:"body,omitempty"`
	// Headers 请求头
	Headers map[string]string `json:"headers,omitempty"`
	// QueryParams 查询参数
	QueryParams map[string]string `json:"query_params,omitempty"`
}

// APIResponse API 响应结构体
type APIResponse struct {
	// StatusCode HTTP 状态码
	StatusCode int `json:"status_code"`
	// Body 响应体
	Body []byte `json:"body"`
	// Headers 响应头
	Headers map[string][]string `json:"headers,omitempty"`
}

// APIError API 错误结构体
type APIError struct {
	// Code 错误码
	Code int `json:"code"`
	// Type 错误类型
	Type string `json:"type"`
	// Message 错误消息
	Message string `json:"message"`
	// Details 错误详情
	Details string `json:"details,omitempty"`
}

// Error 实现 error 接口
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("API错误 [%d:%s]: %s (%s)", e.Code, e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("API错误 [%d:%s]: %s", e.Code, e.Type, e.Message)
}

// NewAPIError 创建 API 错误
// 参数:
//   - code: 错误码
//   - errorType: 错误类型
//   - message: 错误消息
//   - details: 错误详情
//
// 返回:
//   - *APIError: API 错误实例
func NewAPIError(code int, errorType, message, details string) *APIError {
	return &APIError{
		Code:    code,
		Type:    errorType,
		Message: message,
		Details: details,
	}
}

// NewAPIRequest 创建 API 请求
// 参数:
//   - method: HTTP 方法
//   - path: API 路径
//
// 返回:
//   - *APIRequest: API 请求实例
func NewAPIRequest(method, path string) *APIRequest {
	return &APIRequest{
		Method:      method,
		Path:        path,
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
	}
}

// SetBody 设置请求体
// 参数:
//   - body: 请求体
func (req *APIRequest) SetBody(body interface{}) {
	req.Body = body
}

// SetHeader 设置请求头
// 参数:
//   - key: 头部键
//   - value: 头部值
func (req *APIRequest) SetHeader(key, value string) {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers[key] = value
}

// SetQueryParam 设置查询参数
// 参数:
//   - key: 参数键
//   - value: 参数值
func (req *APIRequest) SetQueryParam(key, value string) {
	if req.QueryParams == nil {
		req.QueryParams = make(map[string]string)
	}
	req.QueryParams[key] = value
}

// Validate 验证 API 请求
// 返回:
//   - error: 验证错误
func (req *APIRequest) Validate() error {
	if err := ValidateNotEmpty(req.Method, "请求方法"); err != nil {
		return err
	}
	if err := ValidateNotEmpty(req.Path, "请求路径"); err != nil {
		return err
	}
	return nil
}

// IsSuccess 检查响应是否成功
// 返回:
//   - bool: 是否成功
func (resp *APIResponse) IsSuccess() bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// ParseJSON 解析 JSON 响应体
// 参数:
//   - v: 目标结构体指针
//
// 返回:
//   - error: 解析错误
func (resp *APIResponse) ParseJSON(v interface{}) error {
	if len(resp.Body) == 0 {
		return fmt.Errorf("响应体为空")
	}
	return json.Unmarshal(resp.Body, v)
}

// GetHeader 获取响应头
// 参数:
//   - key: 头部键
//
// 返回:
//   - string: 头部值
//   - bool: 是否存在
func (resp *APIResponse) GetHeader(key string) (string, bool) {
	if values, exists := resp.Headers[key]; exists && len(values) > 0 {
		return values[0], true
	}
	return "", false
}

// RetryConfig 重试配置
type RetryConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries"`
	// InitialDelay 初始延迟时间
	InitialDelay time.Duration `json:"initial_delay"`
	// MaxDelay 最大延迟时间
	MaxDelay time.Duration `json:"max_delay"`
	// Multiplier 退避倍数
	Multiplier float64 `json:"multiplier"`
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// CalculateBackoffDelay 计算退避延迟时间
// 参数:
//   - attempt: 尝试次数
//
// 返回:
//   - time.Duration: 延迟时间
func (rc *RetryConfig) CalculateBackoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rc.InitialDelay
	}

	// 指数退避算法
	delay := rc.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rc.Multiplier)
		if delay > rc.MaxDelay {
			delay = rc.MaxDelay
			break
		}
	}

	return delay
}

// IsRetryableError 判断是否为可重试错误
// 参数:
//   - err: 错误
//
// 返回:
//   - bool: 是否可重试
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否为 API 错误
	if apiErr, ok := err.(*APIError); ok {
		// 认证错误不重试
		if apiErr.Type == "auth" {
			return false
		}
		// 4xx 客户端错误通常不重试（除了 429 限流）
		if apiErr.Code >= 400 && apiErr.Code < 500 && apiErr.Code != 429 {
			return false
		}
		// 5xx 服务器错误和 429 限流错误可以重试
		return apiErr.Code >= 500 || apiErr.Code == 429
	}

	// 其他错误类型默认可重试
	return true
}

// ShouldRetry 判断是否应该重试
// 参数:
//   - err: 错误
//   - attempt: 当前尝试次数
//   - config: 重试配置
//
// 返回:
//   - bool: 是否应该重试
func ShouldRetry(err error, attempt int, config *RetryConfig) bool {
	// 如果已达到最大重试次数，不再重试
	if attempt >= config.MaxRetries {
		return false
	}

	// 检查错误是否可重试
	return IsRetryableError(err)
}

// ExecuteWithRetry 带重试机制执行函数
// 参数:
//   - ctx: 上下文
//   - operation: 操作函数
//   - config: 重试配置
//
// 返回:
//   - error: 执行错误
func ExecuteWithRetry(ctx context.Context, operation func() error, config *RetryConfig) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// 如果不是第一次尝试，等待一段时间
		if attempt > 0 {
			delay := config.CalculateBackoffDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// 继续执行
			}
		}

		// 执行操作
		err := operation()
		if err == nil {
			return nil // 成功
		}

		lastErr = err

		// 检查是否应该重试
		if !ShouldRetry(err, attempt, config) {
			break
		}
	}

	return lastErr
}

// BuildAPIPath 构建 API 路径
// 参数:
//   - basePath: 基础路径
//   - segments: 路径段
//
// 返回:
//   - string: 完整路径
func BuildAPIPath(basePath string, segments ...string) string {
	path := basePath
	for _, segment := range segments {
		if segment != "" {
			if !strings.HasSuffix(path, "/") {
				path += "/"
			}
			path += segment
		}
	}
	return path
}

// ParseAPIError 解析 API 错误响应
// 参数:
//   - resp: API 响应
//
// 返回:
//   - *APIError: 解析后的 API 错误
func ParseAPIError(resp *APIResponse) *APIError {
	if resp.IsSuccess() {
		return nil
	}

	// 尝试解析错误响应体
	var errorResp struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := resp.ParseJSON(&errorResp); err == nil {
		message := errorResp.Message
		if message == "" {
			message = errorResp.Msg
		}
		if message == "" {
			message = errorResp.Error
		}

		return NewAPIError(
			resp.StatusCode,
			"api_error",
			message,
			string(resp.Body),
		)
	}

	// 如果无法解析，返回通用错误
	return NewAPIError(
		resp.StatusCode,
		"http_error",
		fmt.Sprintf("HTTP %d", resp.StatusCode),
		string(resp.Body),
	)
}
