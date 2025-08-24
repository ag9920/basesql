package basesql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ag9920/basesql/internal/common"
	"github.com/ag9920/basesql/internal/security"
)

// Client 飞书 API 客户端，负责处理与飞书多维表格 API 的所有通信
// 包括认证、令牌管理、请求发送等核心功能
type Client struct {
	config         *Config                       // 客户端配置
	httpClient     *http.Client                  // HTTP 客户端
	accessToken    string                        // 当前访问令牌
	tokenMutex     sync.RWMutex                  // 令牌读写锁，保证并发安全
	tokenExpiry    time.Time                     // 令牌过期时间
	retryConfig    *RetryConfig                  // 重试配置
	circuitBreaker *common.CircuitBreaker        // 熔断器
	connectionPool *common.ConnectionPool        // 连接池
	rateLimiter    *common.TokenBucket           // 限流器
	stabilityMutex sync.RWMutex                  // 稳定性组件锁
	maskSensitive  *security.SensitiveDataMasker // 敏感数据遮蔽器
}

// 使用公共工具包的 RetryConfig 类型
type RetryConfig = common.RetryConfig

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: time.Second,
	MaxDelay:     30 * time.Second,
	Multiplier:   2.0,
}

// NewClient 创建新的飞书 API 客户端
// 参数:
//   - config: 客户端配置，包含认证信息和其他设置
//
// 返回:
//   - *Client: 初始化完成的客户端实例
//   - error: 创建过程中的错误
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 初始化连接池
	connectionPool := common.NewConnectionPool(common.DefaultConnectionPoolConfig())

	// 初始化熔断器
	circuitBreaker := common.NewCircuitBreaker(common.DefaultCircuitBreakerConfig())

	// 初始化限流器
	rateLimiter := common.NewTokenBucket(common.DefaultRateLimiterConfig())

	// 初始化敏感数据遮蔽器
	maskSensitive := security.DefaultMaskerConfig()

	client := &Client{
		config:         config,
		httpClient:     connectionPool.GetHTTPClient(),
		retryConfig:    &DefaultRetryConfig,
		circuitBreaker: circuitBreaker,
		connectionPool: connectionPool,
		rateLimiter:    rateLimiter,
		maskSensitive:  maskSensitive,
	}

	// 注册资源到全局资源管理器
	connPoolResource := common.NewManagedConnection(
		fmt.Sprintf("connection_pool_%p", connectionPool),
		"connection_pool",
		connectionPool.Close,
	)
	if err := common.RegisterGlobalResource(connPoolResource); err != nil {
		common.Warnf("注册连接池资源失败: %v", err)
	}

	cbResource := common.NewManagedConnection(
		fmt.Sprintf("circuit_breaker_%p", circuitBreaker),
		"circuit_breaker",
		func() error { return nil }, // 熔断器不需要特殊清理
	)
	if err := common.RegisterGlobalResource(cbResource); err != nil {
		common.Warnf("注册熔断器资源失败: %v", err)
	}

	rlResource := common.NewManagedConnection(
		fmt.Sprintf("rate_limiter_%p", rateLimiter),
		"rate_limiter",
		func() error { return nil }, // 限流器不需要特殊清理
	)
	if err := common.RegisterGlobalResource(rlResource); err != nil {
		common.Warnf("注册限流器资源失败: %v", err)
	}

	// 设置熔断器状态变化回调
	circuitBreaker.SetStateChangeCallback(func(from, to common.CircuitBreakerState) {
		if config.DebugMode {

		}
	})

	// 初始化时获取访问令牌
	if err := client.refreshToken(context.Background()); err != nil {
		return nil, fmt.Errorf("获取访问令牌失败: %w", err)
	}

	return client, nil
}

// SetRetryConfig 设置重试配置
// 参数:
//   - config: 重试配置
func (c *Client) SetRetryConfig(config *RetryConfig) {
	if config != nil {
		c.retryConfig = config
	}
}

// TokenResponse 飞书 API 令牌响应结构
type TokenResponse struct {
	Code   int    `json:"code"`                // 响应状态码，0 表示成功
	Msg    string `json:"msg"`                 // 响应消息
	Expire int64  `json:"expire"`              // 令牌有效期（秒）
	Token  string `json:"tenant_access_token"` // 租户访问令牌
}

// refreshToken 刷新访问令牌
// 支持两种认证方式：用户令牌和应用令牌
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//
// 返回:
//   - error: 刷新过程中的错误
func (c *Client) refreshToken(ctx context.Context) error {
	// 如果是用户认证且已提供访问令牌，直接使用
	if c.config.AuthType == AuthTypeUser && c.config.AccessToken != "" {
		c.tokenMutex.Lock()
		c.accessToken = c.config.AccessToken
		c.tokenExpiry = time.Now().Add(24 * time.Hour) // 用户令牌假设24小时有效
		c.tokenMutex.Unlock()
		return nil
	}

	// 应用认证
	reqBody := map[string]string{
		"app_id":     c.config.AppID,
		"app_secret": c.config.AppSecret,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.config.BaseURL+"/open-apis/auth/v3/tenant_access_token/internal",
		bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return common.NewAPIError(0, "network", fmt.Sprintf("token request failed: %v", err), "")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return err
	}

	if tokenResp.Code != 0 {
		return common.NewAPIError(tokenResp.Code, "auth", fmt.Sprintf("token request failed: %s", tokenResp.Msg), "")
	}

	c.tokenMutex.Lock()
	c.accessToken = tokenResp.Token
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.Expire) * time.Second)
	c.tokenMutex.Unlock()

	return nil
}

// getAccessToken 获取有效的访问令牌
// 自动检查令牌是否即将过期，如果是则自动刷新
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//
// 返回:
//   - string: 有效的访问令牌
//   - error: 获取过程中的错误
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	token := c.accessToken
	expiry := c.tokenExpiry
	c.tokenMutex.RUnlock()

	// 检查令牌是否即将过期（提前5分钟刷新，避免请求时令牌失效）
	if time.Now().Add(5 * time.Minute).After(expiry) {
		if err := c.refreshToken(ctx); err != nil {
			return "", fmt.Errorf("刷新令牌失败: %w", err)
		}
		c.tokenMutex.RLock()
		token = c.accessToken
		c.tokenMutex.RUnlock()
	}

	return token, nil
}

// 使用公共工具包的 API 类型
type APIRequest = common.APIRequest
type APIResponse = common.APIResponse
type APIError = common.APIError

// DoRequest 执行飞书 API 请求
// 这是所有 API 调用的核心方法，处理认证、请求构建、发送和响应解析
// 支持自动重试机制，提高请求的稳定性
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - req: API 请求结构，包含请求的所有信息
//
// 返回:
//   - *APIResponse: API 响应结构
//   - error: 请求过程中的错误
func (c *Client) DoRequest(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	// 参数校验
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("请求方法不能为空")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("请求路径不能为空")
	}

	// 使用重试机制执行请求
	return c.doRequestWithRetry(ctx, req)
}

// doRequestWithRetry 带重试机制的请求执行
func (c *Client) doRequestWithRetry(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// 如果不是第一次尝试，等待一段时间
		if attempt > 0 {
			delay := c.retryConfig.CalculateBackoffDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// 继续执行
			}
		}

		resp, err := c.doSingleRequest(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// 检查是否应该重试
		if !common.ShouldRetry(err, attempt, c.retryConfig) {
			break
		}
	}

	return nil, fmt.Errorf("请求失败，已重试 %d 次: %w", c.retryConfig.MaxRetries, lastErr)
}

// doSingleRequest 执行单次请求
func (c *Client) doSingleRequest(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	// 限流检查
	if !c.rateLimiter.Allow() {
		return nil, common.NewAPIError(429, "rate_limit", "请求频率过高，请稍后重试", "")
	}

	// 使用熔断器执行请求
	var resp *http.Response
	var err error

	err = c.circuitBreaker.Execute(ctx, func() error {
		// 获取有效的访问令牌
		token, tokenErr := c.getAccessToken(ctx)
		if tokenErr != nil {
			return fmt.Errorf("获取访问令牌失败: %w", tokenErr)
		}

		// 构建完整的请求 URL
		reqURL := c.config.BaseURL + "/open-apis" + req.Path
		if len(req.QueryParams) > 0 {
			query := url.Values{}
			for key, value := range req.QueryParams {
				query.Set(key, value)
			}
			reqURL += "?" + query.Encode()
		}

		// 构建请求体
		var body io.Reader
		var bodyBytes []byte
		if req.Body != nil {
			var marshalErr error
			bodyBytes, marshalErr = json.Marshal(req.Body)
			if marshalErr != nil {
				return fmt.Errorf("序列化请求体失败: %w", marshalErr)
			}
			body = bytes.NewBuffer(bodyBytes)
		}

		// 创建 HTTP 请求
		httpReq, reqErr := http.NewRequestWithContext(ctx, req.Method, reqURL, body)
		if reqErr != nil {
			return fmt.Errorf("创建 HTTP 请求失败: %w", reqErr)
		}

		// 设置请求头
		httpReq.Header.Set("Authorization", "Bearer "+token)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("User-Agent", "BaseSQL/1.0.0")

		// 设置自定义请求头
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}

		// 使用连接池执行请求
		resp, err = c.connectionPool.ExecuteRequest(ctx, httpReq)
		return err
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 创建 APIResponse
	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 尝试解析错误响应
		var errorResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if json.Unmarshal(respBody, &errorResp) == nil && errorResp.Code != 0 {
			return nil, common.NewAPIError(errorResp.Code, "api", fmt.Sprintf("API 错误 %d: %s", errorResp.Code, errorResp.Msg), "")
		}
		return nil, fmt.Errorf("API 请求失败: status=%d", resp.StatusCode)
	}

	return apiResp, nil
}

// Close 关闭客户端并清理资源
// 这个方法是幂等的，可以安全地多次调用
func (c *Client) Close() error {
	// 清理令牌相关资源
	c.tokenMutex.Lock()
	c.accessToken = ""
	c.tokenExpiry = time.Time{}
	c.tokenMutex.Unlock()

	// 从全局资源管理器注销连接池
	if c.connectionPool != nil {
		common.UnregisterGlobalResource(fmt.Sprintf("connection_pool_%p", c.connectionPool))
		c.connectionPool.Close()
	}

	// 注销其他稳定性组件
	if c.circuitBreaker != nil {
		common.UnregisterGlobalResource(fmt.Sprintf("circuit_breaker_%p", c.circuitBreaker))
	}

	if c.rateLimiter != nil {
		common.UnregisterGlobalResource(fmt.Sprintf("rate_limiter_%p", c.rateLimiter))
	}

	common.Debug("客户端资源已清理")
	return nil
}

// GetStabilityStats 获取稳定性组件统计信息
// 返回:
//   - map[string]interface{}: 统计信息
func (c *Client) GetStabilityStats() map[string]interface{} {
	c.stabilityMutex.RLock()
	defer c.stabilityMutex.RUnlock()

	stats := make(map[string]interface{})

	if c.circuitBreaker != nil {
		stats["circuit_breaker"] = c.circuitBreaker.GetStats()
	}

	if c.connectionPool != nil {
		stats["connection_pool"] = c.connectionPool.GetStats()
	}

	if c.rateLimiter != nil {
		stats["rate_limiter"] = c.rateLimiter.GetStats()
	}

	return stats
}

// ResetStabilityComponents 重置稳定性组件
// 返回:
//   - error: 重置错误
func (c *Client) ResetStabilityComponents() error {
	c.stabilityMutex.Lock()
	defer c.stabilityMutex.Unlock()

	if c.circuitBreaker != nil {
		c.circuitBreaker.Reset()
	}

	if c.rateLimiter != nil {
		c.rateLimiter.Reset()
	}

	return nil
}

// UpdateRateLimiterConfig 更新限流器配置
// 参数:
//   - config: 新的限流器配置
//
// 返回:
//   - error: 更新错误
func (c *Client) UpdateRateLimiterConfig(config *common.RateLimiterConfig) error {
	if config == nil {
		return fmt.Errorf("限流器配置不能为空")
	}

	c.stabilityMutex.Lock()
	defer c.stabilityMutex.Unlock()

	if c.rateLimiter != nil {
		c.rateLimiter.UpdateConfig(config)
	}

	return nil
}

// UpdateConnectionPoolConfig 更新连接池配置
// 参数:
//   - config: 新的连接池配置
//
// 返回:
//   - error: 更新错误
func (c *Client) UpdateConnectionPoolConfig(config *common.ConnectionPoolConfig) error {
	if config == nil {
		return fmt.Errorf("连接池配置不能为空")
	}

	c.stabilityMutex.Lock()
	defer c.stabilityMutex.Unlock()

	if c.connectionPool != nil {
		return c.connectionPool.UpdateConfig(config)
	}

	return nil
}

// HealthCheck 执行健康检查
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - bool: 是否健康
//   - error: 检查错误
func (c *Client) HealthCheck(ctx context.Context) (bool, error) {
	// 检查基础连接
	healthURL := c.config.BaseURL + "/open-apis/auth/v3/tenant_access_token/internal"

	if c.connectionPool != nil {
		return c.connectionPool.HealthCheck(ctx, healthURL)
	}

	// 如果没有连接池，使用简单的健康检查
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return false, fmt.Errorf("创建健康检查请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 500, nil
}
