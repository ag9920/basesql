package common

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	// Rate 每秒允许的请求数
	Rate float64 `json:"rate"`
	// Burst 突发请求数
	Burst int `json:"burst"`
	// Window 时间窗口
	Window time.Duration `json:"window"`
}

// DefaultRateLimiterConfig 默认限流器配置
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		Rate:   10.0, // 每秒 10 个请求
		Burst:  20,   // 突发 20 个请求
		Window: time.Second,
	}
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	config     *RateLimiterConfig
	tokens     float64
	lastRefill time.Time
	mutex      sync.Mutex
	stats      *RateLimiterStats
}

// RateLimiterStats 限流器统计信息
type RateLimiterStats struct {
	TotalRequests    int64     `json:"total_requests"`
	AllowedRequests  int64     `json:"allowed_requests"`
	RejectedRequests int64     `json:"rejected_requests"`
	LastRequestTime  time.Time `json:"last_request_time"`
	mutex            sync.RWMutex
}

// NewTokenBucket 创建新的令牌桶限流器
// 参数:
//   - config: 限流器配置
//
// 返回:
//   - *TokenBucket: 令牌桶实例
func NewTokenBucket(config *RateLimiterConfig) *TokenBucket {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	return &TokenBucket{
		config:     config,
		tokens:     float64(config.Burst),
		lastRefill: time.Now(),
		stats:      &RateLimiterStats{},
	}
}

// Allow 检查是否允许请求
// 返回:
//   - bool: 是否允许
func (tb *TokenBucket) Allow() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()

	// 更新统计信息
	tb.updateStats(true)

	// 补充令牌
	tb.refill(now)

	// 检查是否有可用令牌
	if tb.tokens >= 1.0 {
		tb.tokens--
		tb.updateStats(false) // 允许请求
		return true
	}

	// 拒绝请求
	tb.updateRejectedStats()
	return false
}

// AllowN 检查是否允许 N 个请求
// 参数:
//   - n: 请求数量
//
// 返回:
//   - bool: 是否允许
func (tb *TokenBucket) AllowN(n int) bool {
	if n <= 0 {
		return true
	}

	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()

	// 更新统计信息
	tb.updateStats(true)

	// 补充令牌
	tb.refill(now)

	// 检查是否有足够的令牌
	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		tb.updateStats(false) // 允许请求
		return true
	}

	// 拒绝请求
	tb.updateRejectedStats()
	return false
}

// Wait 等待直到可以执行请求
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - error: 等待错误
func (tb *TokenBucket) Wait(ctx context.Context) error {
	return tb.WaitN(ctx, 1)
}

// WaitN 等待直到可以执行 N 个请求
// 参数:
//   - ctx: 上下文
//   - n: 请求数量
//
// 返回:
//   - error: 等待错误
func (tb *TokenBucket) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}

	// 检查是否立即可用
	if tb.AllowN(n) {
		return nil
	}

	// 计算需要等待的时间
	waitTime := tb.calculateWaitTime(n)
	if waitTime <= 0 {
		return nil
	}

	// 等待
	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		// 重新检查
		if tb.AllowN(n) {
			return nil
		}
		return fmt.Errorf("等待超时")
	}
}

// calculateWaitTime 计算等待时间
// 参数:
//   - n: 请求数量
//
// 返回:
//   - time.Duration: 等待时间
func (tb *TokenBucket) calculateWaitTime(n int) time.Duration {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// 计算需要的令牌数
	needed := float64(n) - tb.tokens
	if needed <= 0 {
		return 0
	}

	// 计算生成这些令牌需要的时间
	return time.Duration(needed/tb.config.Rate) * time.Second
}

// refill 补充令牌
// 参数:
//   - now: 当前时间
func (tb *TokenBucket) refill(now time.Time) {
	// 计算时间差
	elapsed := now.Sub(tb.lastRefill)
	if elapsed <= 0 {
		return
	}

	// 计算应该添加的令牌数
	tokensToAdd := tb.config.Rate * elapsed.Seconds()
	tb.tokens += tokensToAdd

	// 限制令牌数不超过桶容量
	if tb.tokens > float64(tb.config.Burst) {
		tb.tokens = float64(tb.config.Burst)
	}

	tb.lastRefill = now
}

// updateStats 更新统计信息
// 参数:
//   - isRequest: 是否是新请求
func (tb *TokenBucket) updateStats(isRequest bool) {
	tb.stats.mutex.Lock()
	defer tb.stats.mutex.Unlock()

	if isRequest {
		tb.stats.TotalRequests++
		tb.stats.LastRequestTime = time.Now()
	} else {
		tb.stats.AllowedRequests++
	}
}

// updateRejectedStats 更新拒绝统计信息
func (tb *TokenBucket) updateRejectedStats() {
	tb.stats.mutex.Lock()
	defer tb.stats.mutex.Unlock()
	tb.stats.RejectedRequests++
}

// GetStats 获取统计信息
// 返回:
//   - *RateLimiterStats: 统计信息
func (tb *TokenBucket) GetStats() *RateLimiterStats {
	tb.stats.mutex.RLock()
	defer tb.stats.mutex.RUnlock()

	return &RateLimiterStats{
		TotalRequests:    tb.stats.TotalRequests,
		AllowedRequests:  tb.stats.AllowedRequests,
		RejectedRequests: tb.stats.RejectedRequests,
		LastRequestTime:  tb.stats.LastRequestTime,
	}
}

// GetTokens 获取当前令牌数
// 返回:
//   - float64: 当前令牌数
func (tb *TokenBucket) GetTokens() float64 {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// 先补充令牌
	tb.refill(time.Now())
	return tb.tokens
}

// Reset 重置限流器
func (tb *TokenBucket) Reset() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.tokens = float64(tb.config.Burst)
	tb.lastRefill = time.Now()

	// 重置统计信息
	tb.stats.mutex.Lock()
	tb.stats.TotalRequests = 0
	tb.stats.AllowedRequests = 0
	tb.stats.RejectedRequests = 0
	tb.stats.LastRequestTime = time.Time{}
	tb.stats.mutex.Unlock()
}

// UpdateConfig 更新配置
// 参数:
//   - config: 新配置
func (tb *TokenBucket) UpdateConfig(config *RateLimiterConfig) {
	if config == nil {
		return
	}

	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.config = config
	// 调整当前令牌数
	if tb.tokens > float64(config.Burst) {
		tb.tokens = float64(config.Burst)
	}
}
