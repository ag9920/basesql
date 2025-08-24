package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ag9920/basesql/internal/common"
)

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	maxBatchSize   int
	maxConcurrency int
	cacheEnabled   bool
	queryCache     *QueryCache
	metrics        *PerformanceMetrics
	mutex          sync.RWMutex
}

// QueryCache 查询缓存
type QueryCache struct {
	cache     map[string]*CacheEntry
	mutex     sync.RWMutex
	maxSize   int
	ttl       time.Duration
	cleanupCh chan struct{}
}

// CacheEntry 缓存条目
type CacheEntry struct {
	data      interface{}
	timestamp time.Time
	hitCount  int64
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	QueryCount       int64         `json:"query_count"`
	TotalDuration    time.Duration `json:"total_duration"`
	AverageDuration  time.Duration `json:"average_duration"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	MemoryUsage      int64         `json:"memory_usage"`
	GoroutineCount   int           `json:"goroutine_count"`
	LastOptimization time.Time     `json:"last_optimization"`
	mutex            sync.RWMutex
}

// NewQueryOptimizer 创建新的查询优化器
func NewQueryOptimizer(config *OptimizerConfig) *QueryOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}

	queryCache := &QueryCache{
		cache:     make(map[string]*CacheEntry),
		maxSize:   config.CacheMaxSize,
		ttl:       config.CacheTTL,
		cleanupCh: make(chan struct{}),
	}

	optimizer := &QueryOptimizer{
		maxBatchSize:   config.MaxBatchSize,
		maxConcurrency: config.MaxConcurrency,
		cacheEnabled:   config.CacheEnabled,
		queryCache:     queryCache,
		metrics:        &PerformanceMetrics{},
	}

	// 启动缓存清理协程
	if optimizer.cacheEnabled {
		go optimizer.startCacheCleanup()
	}

	return optimizer
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	MaxBatchSize   int           `json:"max_batch_size"`
	MaxConcurrency int           `json:"max_concurrency"`
	CacheEnabled   bool          `json:"cache_enabled"`
	CacheMaxSize   int           `json:"cache_max_size"`
	CacheTTL       time.Duration `json:"cache_ttl"`
}

// DefaultOptimizerConfig 返回默认优化器配置
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		MaxBatchSize:   common.DefaultBatchSize,
		MaxConcurrency: runtime.NumCPU() * 2,
		CacheEnabled:   true,
		CacheMaxSize:   1000,
		CacheTTL:       5 * time.Minute,
	}
}

// OptimizeBatchQuery 优化批量查询
func (o *QueryOptimizer) OptimizeBatchQuery(ctx context.Context, queries []string, executor func(context.Context, []string) ([]interface{}, error)) ([]interface{}, error) {
	start := time.Now()
	defer func() {
		o.updateMetrics(time.Since(start))
	}()

	if len(queries) == 0 {
		return nil, fmt.Errorf("查询列表不能为空")
	}

	// 检查缓存
	if o.cacheEnabled {
		if cachedResults := o.getCachedResults(queries); cachedResults != nil {
			o.metrics.mutex.Lock()
			o.metrics.QueryCount++
			o.metrics.mutex.Unlock()
			return cachedResults, nil
		}
	}

	// 分批处理
	batches := o.splitIntoBatches(queries)
	results := make([]interface{}, 0, len(queries))

	// 并发执行批次
	semaphore := make(chan struct{}, o.maxConcurrency)
	resultCh := make(chan batchResult, len(batches))

	for i, batch := range batches {
		go func(batchIndex int, batchQueries []string) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			batchResults, err := executor(ctx, batchQueries)
			resultCh <- batchResult{
				index:   batchIndex,
				results: batchResults,
				err:     err,
			}
		}(i, batch)
	}

	// 收集结果
	batchResults := make([]batchResult, len(batches))
	for i := 0; i < len(batches); i++ {
		select {
		case result := <-resultCh:
			batchResults[result.index] = result
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 合并结果
	for _, batchResult := range batchResults {
		if batchResult.err != nil {
			return nil, batchResult.err
		}
		results = append(results, batchResult.results...)
	}

	// 缓存结果
	if o.cacheEnabled {
		o.cacheResults(queries, results)
	}

	return results, nil
}

type batchResult struct {
	index   int
	results []interface{}
	err     error
}

// splitIntoBatches 将查询分割成批次
func (o *QueryOptimizer) splitIntoBatches(queries []string) [][]string {
	var batches [][]string

	for i := 0; i < len(queries); i += o.maxBatchSize {
		end := i + o.maxBatchSize
		if end > len(queries) {
			end = len(queries)
		}
		batches = append(batches, queries[i:end])
	}

	return batches
}

// getCachedResults 获取缓存结果
func (o *QueryOptimizer) getCachedResults(queries []string) []interface{} {
	if !o.cacheEnabled {
		return nil
	}

	cacheKey := o.generateCacheKey(queries)
	o.queryCache.mutex.RLock()
	entry, exists := o.queryCache.cache[cacheKey]
	o.queryCache.mutex.RUnlock()

	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Since(entry.timestamp) > o.queryCache.ttl {
		o.queryCache.mutex.Lock()
		delete(o.queryCache.cache, cacheKey)
		o.queryCache.mutex.Unlock()
		return nil
	}

	// 更新命中次数
	o.queryCache.mutex.Lock()
	entry.hitCount++
	o.queryCache.mutex.Unlock()

	if results, ok := entry.data.([]interface{}); ok {
		return results
	}

	return nil
}

// cacheResults 缓存结果
func (o *QueryOptimizer) cacheResults(queries []string, results []interface{}) {
	if !o.cacheEnabled {
		return
	}

	cacheKey := o.generateCacheKey(queries)

	o.queryCache.mutex.Lock()
	defer o.queryCache.mutex.Unlock()

	// 检查缓存大小限制
	if len(o.queryCache.cache) >= o.queryCache.maxSize {
		o.evictOldestEntry()
	}

	o.queryCache.cache[cacheKey] = &CacheEntry{
		data:      results,
		timestamp: time.Now(),
		hitCount:  0,
	}
}

// generateCacheKey 生成缓存键
func (o *QueryOptimizer) generateCacheKey(queries []string) string {
	// 简单的键生成策略，实际应用中可能需要更复杂的哈希算法
	key := ""
	for _, query := range queries {
		key += query + "|"
	}
	return key
}

// evictOldestEntry 驱逐最旧的缓存条目
func (o *QueryOptimizer) evictOldestEntry() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range o.queryCache.cache {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
		}
	}

	if oldestKey != "" {
		delete(o.queryCache.cache, oldestKey)
	}
}

// startCacheCleanup 启动缓存清理
func (o *QueryOptimizer) startCacheCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			o.cleanupExpiredEntries()
		case <-o.queryCache.cleanupCh:
			return
		}
	}
}

// cleanupExpiredEntries 清理过期条目
func (o *QueryOptimizer) cleanupExpiredEntries() {
	o.queryCache.mutex.Lock()
	defer o.queryCache.mutex.Unlock()

	now := time.Now()
	for key, entry := range o.queryCache.cache {
		if now.Sub(entry.timestamp) > o.queryCache.ttl {
			delete(o.queryCache.cache, key)
		}
	}
}

// updateMetrics 更新性能指标
func (o *QueryOptimizer) updateMetrics(duration time.Duration) {
	o.metrics.mutex.Lock()
	defer o.metrics.mutex.Unlock()

	o.metrics.QueryCount++
	o.metrics.TotalDuration += duration
	o.metrics.AverageDuration = o.metrics.TotalDuration / time.Duration(o.metrics.QueryCount)

	// 更新内存使用情况
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	o.metrics.MemoryUsage = int64(m.Alloc)
	o.metrics.GoroutineCount = runtime.NumGoroutine()

	// 计算缓存命中率
	if o.cacheEnabled {
		o.queryCache.mutex.RLock()
		totalHits := int64(0)
		for _, entry := range o.queryCache.cache {
			totalHits += entry.hitCount
		}
		o.queryCache.mutex.RUnlock()

		if o.metrics.QueryCount > 0 {
			o.metrics.CacheHitRate = float64(totalHits) / float64(o.metrics.QueryCount)
		}
	}
}

// GetMetrics 获取性能指标
func (o *QueryOptimizer) GetMetrics() *PerformanceMetrics {
	o.metrics.mutex.RLock()
	defer o.metrics.mutex.RUnlock()

	return &PerformanceMetrics{
		QueryCount:       o.metrics.QueryCount,
		TotalDuration:    o.metrics.TotalDuration,
		AverageDuration:  o.metrics.AverageDuration,
		CacheHitRate:     o.metrics.CacheHitRate,
		MemoryUsage:      o.metrics.MemoryUsage,
		GoroutineCount:   o.metrics.GoroutineCount,
		LastOptimization: o.metrics.LastOptimization,
	}
}

// ClearCache 清空缓存
func (o *QueryOptimizer) ClearCache() {
	o.queryCache.mutex.Lock()
	defer o.queryCache.mutex.Unlock()

	o.queryCache.cache = make(map[string]*CacheEntry)
}

// Close 关闭优化器
func (o *QueryOptimizer) Close() {
	close(o.queryCache.cleanupCh)
}

// MemoryPool 内存池
type MemoryPool struct {
	bufferPool sync.Pool
	maxSize    int
}

// NewMemoryPool 创建新的内存池
func NewMemoryPool(maxSize int) *MemoryPool {
	return &MemoryPool{
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, maxSize)
			},
		},
		maxSize: maxSize,
	}
}

// Get 获取缓冲区
func (p *MemoryPool) Get() []byte {
	return p.bufferPool.Get().([]byte)
}

// Put 归还缓冲区
func (p *MemoryPool) Put(buf []byte) {
	if cap(buf) <= p.maxSize {
		buf = buf[:0] // 重置长度但保留容量
		p.bufferPool.Put(buf)
	}
}
