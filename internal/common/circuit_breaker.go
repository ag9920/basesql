package common

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	// StateClosed 关闭状态 - 正常工作
	StateClosed CircuitBreakerState = iota
	// StateOpen 开启状态 - 熔断中
	StateOpen
	// StateHalfOpen 半开状态 - 尝试恢复
	StateHalfOpen
)

// String 返回状态的字符串表示
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// MaxFailures 最大失败次数
	MaxFailures int `json:"max_failures"`
	// Timeout 熔断超时时间
	Timeout time.Duration `json:"timeout"`
	// MaxRequests 半开状态下的最大请求数
	MaxRequests int `json:"max_requests"`
	// Interval 统计间隔
	Interval time.Duration `json:"interval"`
}

// DefaultCircuitBreakerConfig 默认熔断器配置
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures: 5,
		Timeout:     60 * time.Second,
		MaxRequests: 3,
		Interval:    60 * time.Second,
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config        *CircuitBreakerConfig
	state         CircuitBreakerState
	failures      int
	requests      int
	lastFailTime  time.Time
	mutex         sync.RWMutex
	onStateChange func(from, to CircuitBreakerState)
}

// NewCircuitBreaker 创建新的熔断器
// 参数:
//   - config: 熔断器配置
//
// 返回:
//   - *CircuitBreaker: 熔断器实例
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// SetStateChangeCallback 设置状态变化回调
// 参数:
//   - callback: 状态变化回调函数
func (cb *CircuitBreaker) SetStateChangeCallback(callback func(from, to CircuitBreakerState)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.onStateChange = callback
}

// Execute 执行操作
// 参数:
//   - ctx: 上下文
//   - operation: 要执行的操作
//
// 返回:
//   - error: 执行错误
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// 检查是否允许执行
	if !cb.allowRequest() {
		return fmt.Errorf("熔断器开启，拒绝请求")
	}

	// 执行操作
	err := operation()

	// 记录结果
	cb.recordResult(err)

	return err
}

// allowRequest 检查是否允许请求
// 返回:
//   - bool: 是否允许请求
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否可以转为半开状态
		if now.Sub(cb.lastFailTime) > cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.requests = 0
			return true
		}
		return false
	case StateHalfOpen:
		// 半开状态下限制请求数
		return cb.requests < cb.config.MaxRequests
	default:
		return false
	}
}

// recordResult 记录执行结果
// 参数:
//   - err: 执行错误
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		if err != nil {
			cb.failures++
			cb.lastFailTime = now
			// 检查是否需要开启熔断
			if cb.failures >= cb.config.MaxFailures {
				cb.setState(StateOpen)
			}
		} else {
			// 成功时重置失败计数
			cb.failures = 0
		}
	case StateHalfOpen:
		cb.requests++
		if err != nil {
			// 半开状态下失败，重新开启熔断
			cb.setState(StateOpen)
			cb.lastFailTime = now
		} else {
			// 半开状态下成功，检查是否可以关闭熔断
			if cb.requests >= cb.config.MaxRequests {
				cb.setState(StateClosed)
				cb.failures = 0
				cb.requests = 0
			}
		}
	}
}

// setState 设置状态
// 参数:
//   - newState: 新状态
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState

	// 触发状态变化回调
	if cb.onStateChange != nil && oldState != newState {
		// 创建回调函数的副本，避免在goroutine中访问可能变化的字段
		callback := cb.onStateChange
		go func(from, to CircuitBreakerState) {
			// 使用defer和recover确保回调函数的异常不会影响主逻辑
			defer func() {
				if r := recover(); r != nil {
					// 记录回调函数的异常，但不影响主流程
					Debugf("熔断器状态变化回调函数异常: %v", r)
				}
			}()
			callback(from, to)
		}(oldState, newState)
	}
}

// GetState 获取当前状态
// 返回:
//   - CircuitBreakerState: 当前状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats 获取统计信息
// 返回:
//   - map[string]interface{}: 统计信息
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"state":          cb.state.String(),
		"failures":       cb.failures,
		"requests":       cb.requests,
		"last_fail_time": cb.lastFailTime,
	}
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.setState(StateClosed)
	cb.failures = 0
	cb.requests = 0
	cb.lastFailTime = time.Time{}
}
