package common

import (
	"sync"
	"sync/atomic"
	"time"
)

// ConcurrencySafetyChecker 并发安全检查器
// 用于检测和报告潜在的并发安全问题
type ConcurrencySafetyChecker struct {
	detectedIssues []ConcurrencyIssue
	mutex          sync.RWMutex
}

// ConcurrencyIssue 并发安全问题
type ConcurrencyIssue struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Severity    string    `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
	Fixed       bool      `json:"fixed"`
}

// SafeCounter 线程安全的计数器
// 使用原子操作确保并发安全
type SafeCounter struct {
	value int64
}

// NewSafeCounter 创建新的安全计数器
func NewSafeCounter() *SafeCounter {
	return &SafeCounter{}
}

// Increment 原子递增
func (sc *SafeCounter) Increment() int64 {
	return atomic.AddInt64(&sc.value, 1)
}

// Decrement 原子递减
func (sc *SafeCounter) Decrement() int64 {
	return atomic.AddInt64(&sc.value, -1)
}

// Get 原子获取值
func (sc *SafeCounter) Get() int64 {
	return atomic.LoadInt64(&sc.value)
}

// Set 原子设置值
func (sc *SafeCounter) Set(value int64) {
	atomic.StoreInt64(&sc.value, value)
}

// SafeMap 线程安全的映射
// 使用读写锁保护并发访问
type SafeMap struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewSafeMap 创建新的安全映射
func NewSafeMap() *SafeMap {
	return &SafeMap{
		data: make(map[string]interface{}),
	}
}

// Set 设置键值对
func (sm *SafeMap) Set(key string, value interface{}) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.data[key] = value
}

// Get 获取值
func (sm *SafeMap) Get(key string) (interface{}, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	value, exists := sm.data[key]
	return value, exists
}

// Delete 删除键值对
func (sm *SafeMap) Delete(key string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.data, key)
}

// Keys 获取所有键
func (sm *SafeMap) Keys() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	keys := make([]string, 0, len(sm.data))
	for key := range sm.data {
		keys = append(keys, key)
	}
	return keys
}

// Size 获取映射大小
func (sm *SafeMap) Size() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.data)
}

// NewConcurrencySafetyChecker 创建并发安全检查器
func NewConcurrencySafetyChecker() *ConcurrencySafetyChecker {
	return &ConcurrencySafetyChecker{
		detectedIssues: make([]ConcurrencyIssue, 0),
	}
}

// ReportIssue 报告并发安全问题
func (csc *ConcurrencySafetyChecker) ReportIssue(issueType, description, location, severity string) {
	csc.mutex.Lock()
	defer csc.mutex.Unlock()

	issue := ConcurrencyIssue{
		Type:        issueType,
		Description: description,
		Location:    location,
		Severity:    severity,
		DetectedAt:  time.Now(),
		Fixed:       false,
	}

	csc.detectedIssues = append(csc.detectedIssues, issue)
	Warnf("检测到并发安全问题: %s - %s (位置: %s)", issueType, description, location)
}

// MarkIssueFixed 标记问题已修复
func (csc *ConcurrencySafetyChecker) MarkIssueFixed(index int) {
	csc.mutex.Lock()
	defer csc.mutex.Unlock()

	if index >= 0 && index < len(csc.detectedIssues) {
		csc.detectedIssues[index].Fixed = true
		Infof("并发安全问题已修复: %s", csc.detectedIssues[index].Description)
	}
}

// reportIssueSilent 静默报告并发安全问题（不输出日志）
func (csc *ConcurrencySafetyChecker) reportIssueSilent(issueType, description, location, severity string) {
	csc.mutex.Lock()
	defer csc.mutex.Unlock()

	issue := ConcurrencyIssue{
		Type:        issueType,
		Description: description,
		Location:    location,
		Severity:    severity,
		DetectedAt:  time.Now(),
		Fixed:       false,
	}

	csc.detectedIssues = append(csc.detectedIssues, issue)
	// 静默模式：不输出任何日志
}

// markIssueFixedSilent 静默标记问题已修复（不输出日志）
func (csc *ConcurrencySafetyChecker) markIssueFixedSilent(index int) {
	csc.mutex.Lock()
	defer csc.mutex.Unlock()

	if index >= 0 && index < len(csc.detectedIssues) {
		csc.detectedIssues[index].Fixed = true
		// 静默模式：不输出任何日志
	}
}

// GetIssues 获取所有检测到的问题
func (csc *ConcurrencySafetyChecker) GetIssues() []ConcurrencyIssue {
	csc.mutex.RLock()
	defer csc.mutex.RUnlock()

	// 返回副本
	issues := make([]ConcurrencyIssue, len(csc.detectedIssues))
	copy(issues, csc.detectedIssues)
	return issues
}

// GetUnfixedIssues 获取未修复的问题
func (csc *ConcurrencySafetyChecker) GetUnfixedIssues() []ConcurrencyIssue {
	csc.mutex.RLock()
	defer csc.mutex.RUnlock()

	var unfixed []ConcurrencyIssue
	for _, issue := range csc.detectedIssues {
		if !issue.Fixed {
			unfixed = append(unfixed, issue)
		}
	}
	return unfixed
}

// ConcurrencySafetyReport 并发安全报告
type ConcurrencySafetyReport struct {
	TotalIssues   int                `json:"total_issues"`
	FixedIssues   int                `json:"fixed_issues"`
	UnfixedIssues int                `json:"unfixed_issues"`
	IssuesByType  map[string]int     `json:"issues_by_type"`
	Issues        []ConcurrencyIssue `json:"issues"`
	GeneratedAt   time.Time          `json:"generated_at"`
}

// GenerateReport 生成并发安全报告
func (csc *ConcurrencySafetyChecker) GenerateReport() *ConcurrencySafetyReport {
	csc.mutex.RLock()
	defer csc.mutex.RUnlock()

	report := &ConcurrencySafetyReport{
		TotalIssues:  len(csc.detectedIssues),
		IssuesByType: make(map[string]int),
		Issues:       make([]ConcurrencyIssue, len(csc.detectedIssues)),
		GeneratedAt:  time.Now(),
	}

	copy(report.Issues, csc.detectedIssues)

	for _, issue := range csc.detectedIssues {
		report.IssuesByType[issue.Type]++
		if issue.Fixed {
			report.FixedIssues++
		} else {
			report.UnfixedIssues++
		}
	}

	return report
}

// 全局并发安全检查器
var globalConcurrencyChecker *ConcurrencySafetyChecker
var globalConcurrencyCheckerOnce sync.Once

// GetGlobalConcurrencyChecker 获取全局并发安全检查器
func GetGlobalConcurrencyChecker() *ConcurrencySafetyChecker {
	globalConcurrencyCheckerOnce.Do(func() {
		globalConcurrencyChecker = NewConcurrencySafetyChecker()
	})
	return globalConcurrencyChecker
}

// ReportConcurrencyIssue 报告全局并发安全问题
func ReportConcurrencyIssue(issueType, description, location, severity string) {
	GetGlobalConcurrencyChecker().ReportIssue(issueType, description, location, severity)
}

// GetConcurrencyReport 获取全局并发安全报告
func GetConcurrencyReport() *ConcurrencySafetyReport {
	return GetGlobalConcurrencyChecker().GenerateReport()
}

// 并发安全最佳实践检查

// CheckDoubleCheckedLocking 检查双重检查锁定模式
func CheckDoubleCheckedLocking(description string) {
	// 这是一个示例函数，用于检查双重检查锁定模式的正确性
	ReportConcurrencyIssue(
		"double_checked_locking",
		"可能存在双重检查锁定问题: "+description,
		"unknown",
		"medium",
	)
}

// CheckRaceCondition 检查竞态条件
func CheckRaceCondition(description string) {
	// 这是一个示例函数，用于检查竞态条件
	ReportConcurrencyIssue(
		"race_condition",
		"可能存在竞态条件: "+description,
		"unknown",
		"high",
	)
}

// CheckDeadlock 检查死锁
func CheckDeadlock(description string) {
	// 这是一个示例函数，用于检查死锁
	ReportConcurrencyIssue(
		"deadlock",
		"可能存在死锁: "+description,
		"unknown",
		"high",
	)
}

// CheckGoroutineLeak 检查goroutine泄漏
func CheckGoroutineLeak(description string) {
	// 这是一个示例函数，用于检查goroutine泄漏
	ReportConcurrencyIssue(
		"goroutine_leak",
		"可能存在goroutine泄漏: "+description,
		"unknown",
		"medium",
	)
}

// 已修复的并发安全问题记录

// 1. 熔断器状态变化回调的并发安全问题
// 问题：在setState方法中，状态变化回调是在goroutine中异步执行的，
//       但没有适当的错误处理和资源保护机制
// 修复：添加了defer recover机制，确保回调函数的异常不会影响主逻辑，
//       并创建回调函数的副本避免在goroutine中访问可能变化的字段

// 2. 性能优化器缓存清理的并发安全问题
// 问题：缓存清理goroutine在缓存未启用时仍然启动
// 修复：只在缓存启用时才启动清理goroutine

// 3. 资源管理器的并发安全
// 问题：资源管理器的清理协程和资源访问之间可能存在竞态条件
// 修复：使用适当的锁机制保护共享状态，确保清理协程的安全停止

func init() {
	// 静默初始化并发安全检查器，避免启动时显示日志
	go func() {
		// 等待一小段时间确保logger初始化完成
		time.Sleep(100 * time.Millisecond)

		// 静默记录已修复的问题（仅用于内部统计，不输出日志）
		checker := GetGlobalConcurrencyChecker()

		// 静默记录熔断器修复
		checker.reportIssueSilent(
			"callback_safety",
			"熔断器状态变化回调的并发安全问题",
			"internal/common/circuit_breaker.go:setState",
			"medium",
		)
		checker.markIssueFixedSilent(0)

		// 静默记录优化器修复
		checker.reportIssueSilent(
			"goroutine_management",
			"性能优化器缓存清理goroutine的启动条件",
			"internal/performance/optimizer.go:NewQueryOptimizer",
			"low",
		)
		checker.markIssueFixedSilent(1)

		// 使用 DEBUG 级别记录初始化完成（正常情况下不会显示）
		Debug("并发安全检查器已初始化，已记录修复的问题")
	}()
}
