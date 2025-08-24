package common

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	// MaxConnections 最大连接数
	MaxConnections int `json:"max_connections"`
	// MaxIdleConnections 最大空闲连接数
	MaxIdleConnections int `json:"max_idle_connections"`
	// ConnectionTimeout 连接超时时间
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	// IdleTimeout 空闲超时时间
	IdleTimeout time.Duration `json:"idle_timeout"`
	// KeepAlive 保持连接时间
	KeepAlive time.Duration `json:"keep_alive"`
}

// DefaultConnectionPoolConfig 默认连接池配置
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxConnections:     DefaultMaxConnections,
		MaxIdleConnections: 10,
		ConnectionTimeout:  DefaultConnectionTimeout,
		IdleTimeout:        90 * time.Second,
		KeepAlive:          DefaultKeepAlive,
	}
}

// ConnectionPool 连接池
type ConnectionPool struct {
	config     *ConnectionPoolConfig
	httpClient *http.Client
	stats      *PoolStats
	mutex      sync.RWMutex
}

// PoolStats 连接池统计信息
type PoolStats struct {
	ActiveConnections int           `json:"active_connections"`
	IdleConnections   int           `json:"idle_connections"`
	TotalRequests     int64         `json:"total_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	AverageLatency    time.Duration `json:"average_latency"`
	LastRequestTime   time.Time     `json:"last_request_time"`
	mutex             sync.RWMutex
}

// NewConnectionPool 创建新的连接池
// 参数:
//   - config: 连接池配置
//
// 返回:
//   - *ConnectionPool: 连接池实例
func NewConnectionPool(config *ConnectionPoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	// 创建 HTTP 客户端
	transport := &http.Transport{
		MaxIdleConns:        config.MaxConnections,
		MaxIdleConnsPerHost: config.MaxIdleConnections,
		IdleConnTimeout:     config.IdleTimeout,
		DisableKeepAlives:   false,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.ConnectionTimeout,
	}

	return &ConnectionPool{
		config:     config,
		httpClient: httpClient,
		stats:      &PoolStats{},
	}
}

// GetHTTPClient 获取 HTTP 客户端
// 返回:
//   - *http.Client: HTTP 客户端
func (cp *ConnectionPool) GetHTTPClient() *http.Client {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.httpClient
}

// ExecuteRequest 执行 HTTP 请求
// 参数:
//   - ctx: 上下文
//   - req: HTTP 请求
//
// 返回:
//   - *http.Response: HTTP 响应
//   - error: 执行错误
func (cp *ConnectionPool) ExecuteRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	start := time.Now()

	// 更新统计信息
	cp.updateRequestStats(true)

	// 执行请求
	resp, err := cp.httpClient.Do(req.WithContext(ctx))

	// 记录延迟
	latency := time.Since(start)
	cp.updateLatencyStats(latency)

	if err != nil {
		cp.updateRequestStats(false)
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}

	return resp, nil
}

// updateRequestStats 更新请求统计信息
// 参数:
//   - success: 是否成功
func (cp *ConnectionPool) updateRequestStats(success bool) {
	cp.stats.mutex.Lock()
	defer cp.stats.mutex.Unlock()

	cp.stats.TotalRequests++
	cp.stats.LastRequestTime = time.Now()

	if !success {
		cp.stats.FailedRequests++
	}
}

// updateLatencyStats 更新延迟统计信息
// 参数:
//   - latency: 请求延迟
func (cp *ConnectionPool) updateLatencyStats(latency time.Duration) {
	cp.stats.mutex.Lock()
	defer cp.stats.mutex.Unlock()

	// 简单的移动平均
	if cp.stats.AverageLatency == 0 {
		cp.stats.AverageLatency = latency
	} else {
		cp.stats.AverageLatency = (cp.stats.AverageLatency + latency) / 2
	}
}

// GetStats 获取连接池统计信息
// 返回:
//   - *PoolStats: 统计信息
func (cp *ConnectionPool) GetStats() *PoolStats {
	cp.stats.mutex.RLock()
	defer cp.stats.mutex.RUnlock()

	// 返回统计信息的副本
	return &PoolStats{
		ActiveConnections: cp.stats.ActiveConnections,
		IdleConnections:   cp.stats.IdleConnections,
		TotalRequests:     cp.stats.TotalRequests,
		FailedRequests:    cp.stats.FailedRequests,
		AverageLatency:    cp.stats.AverageLatency,
		LastRequestTime:   cp.stats.LastRequestTime,
	}
}

// GetConfig 获取连接池配置
// 返回:
//   - *ConnectionPoolConfig: 连接池配置
func (cp *ConnectionPool) GetConfig() *ConnectionPoolConfig {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.config
}

// UpdateConfig 更新连接池配置
// 参数:
//   - config: 新的连接池配置
//
// 返回:
//   - error: 更新错误
func (cp *ConnectionPool) UpdateConfig(config *ConnectionPoolConfig) error {
	if config == nil {
		return fmt.Errorf("连接池配置不能为空")
	}

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// 更新配置
	cp.config = config

	// 重新创建 HTTP 客户端
	transport := &http.Transport{
		MaxIdleConns:        config.MaxConnections,
		MaxIdleConnsPerHost: config.MaxIdleConnections,
		IdleConnTimeout:     config.IdleTimeout,
		DisableKeepAlives:   false,
	}

	cp.httpClient = &http.Client{
		Transport: transport,
		Timeout:   config.ConnectionTimeout,
	}

	return nil
}

// Close 关闭连接池
// 返回:
//   - error: 关闭错误
func (cp *ConnectionPool) Close() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// 关闭 HTTP 客户端的传输层
	if transport, ok := cp.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// HealthCheck 健康检查
// 参数:
//   - ctx: 上下文
//   - url: 检查的 URL
//
// 返回:
//   - bool: 是否健康
//   - error: 检查错误
func (cp *ConnectionPool) HealthCheck(ctx context.Context, url string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("创建健康检查请求失败: %w", err)
	}

	resp, err := cp.ExecuteRequest(ctx, req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}
