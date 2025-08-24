package common

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ResourceManager 资源管理器
// 负责跟踪和管理应用程序中的各种资源，防止内存泄漏
type ResourceManager struct {
	resources map[string]Resource
	mutex     sync.RWMutex
	closed    bool
	cleanupCh chan struct{}
	stats     *ResourceStats
}

// Resource 资源接口
// 所有需要管理的资源都应该实现这个接口
type Resource interface {
	// Close 关闭资源
	Close() error
	// GetType 获取资源类型
	GetType() string
	// GetID 获取资源ID
	GetID() string
	// IsActive 检查资源是否活跃
	IsActive() bool
}

// ResourceStats 资源统计信息
type ResourceStats struct {
	TotalResources  int            `json:"total_resources"`
	ActiveResources int            `json:"active_resources"`
	ResourcesByType map[string]int `json:"resources_by_type"`
	MemoryUsage     int64          `json:"memory_usage"`
	GoroutineCount  int            `json:"goroutine_count"`
	LastCleanupTime time.Time      `json:"last_cleanup_time"`
	CleanupCount    int64          `json:"cleanup_count"`
	LeakedResources int64          `json:"leaked_resources"`
	mutex           sync.RWMutex
}

// ManagedConnection 受管理的连接
type ManagedConnection struct {
	id        string
	connType  string
	createdAt time.Time
	lastUsed  time.Time
	active    bool
	closeFunc func() error
	mutex     sync.RWMutex
}

// NewResourceManager 创建新的资源管理器
func NewResourceManager() *ResourceManager {
	rm := &ResourceManager{
		resources: make(map[string]Resource),
		cleanupCh: make(chan struct{}),
		stats: &ResourceStats{
			ResourcesByType: make(map[string]int),
		},
	}

	// 启动清理协程
	go rm.startCleanupRoutine()

	return rm
}

// RegisterResource 注册资源
// 参数:
//   - resource: 要注册的资源
//
// 返回:
//   - error: 注册错误
func (rm *ResourceManager) RegisterResource(resource Resource) error {
	if rm.closed {
		return fmt.Errorf("资源管理器已关闭")
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	id := resource.GetID()
	if _, exists := rm.resources[id]; exists {
		return fmt.Errorf("资源 %s 已存在", id)
	}

	rm.resources[id] = resource

	// 更新统计信息
	rm.updateStats()

	Debugf("注册资源: %s (类型: %s)", id, resource.GetType())
	return nil
}

// UnregisterResource 注销资源
// 参数:
//   - resourceID: 资源ID
//
// 返回:
//   - error: 注销错误
func (rm *ResourceManager) UnregisterResource(resourceID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	resource, exists := rm.resources[resourceID]
	if !exists {
		return fmt.Errorf("资源 %s 不存在", resourceID)
	}

	// 关闭资源
	if err := resource.Close(); err != nil {
		Warnf("关闭资源 %s 时出错: %v", resourceID, err)
	}

	delete(rm.resources, resourceID)

	// 更新统计信息
	rm.updateStats()

	Debugf("注销资源: %s", resourceID)
	return nil
}

// GetResource 获取资源
// 参数:
//   - resourceID: 资源ID
//
// 返回:
//   - Resource: 资源实例
//   - bool: 是否存在
func (rm *ResourceManager) GetResource(resourceID string) (Resource, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	resource, exists := rm.resources[resourceID]
	return resource, exists
}

// GetResourcesByType 按类型获取资源
// 参数:
//   - resourceType: 资源类型
//
// 返回:
//   - []Resource: 资源列表
func (rm *ResourceManager) GetResourcesByType(resourceType string) []Resource {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var resources []Resource
	for _, resource := range rm.resources {
		if resource.GetType() == resourceType {
			resources = append(resources, resource)
		}
	}

	return resources
}

// CleanupInactiveResources 清理非活跃资源
// 返回:
//   - int: 清理的资源数量
//   - error: 清理错误
func (rm *ResourceManager) CleanupInactiveResources() (int, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	var toRemove []string
	for id, resource := range rm.resources {
		if !resource.IsActive() {
			toRemove = append(toRemove, id)
		}
	}

	cleanedCount := 0
	for _, id := range toRemove {
		resource := rm.resources[id]
		if err := resource.Close(); err != nil {
			Warnf("关闭非活跃资源 %s 时出错: %v", id, err)
			rm.stats.LeakedResources++
		} else {
			cleanedCount++
		}
		delete(rm.resources, id)
	}

	// 更新统计信息
	rm.updateStats()
	rm.stats.CleanupCount++
	rm.stats.LastCleanupTime = time.Now()

	if cleanedCount > 0 {
		Infof("清理了 %d 个非活跃资源", cleanedCount)
	}

	return cleanedCount, nil
}

// GetStats 获取资源统计信息
// 返回:
//   - *ResourceStats: 统计信息副本
func (rm *ResourceManager) GetStats() *ResourceStats {
	rm.stats.mutex.RLock()
	defer rm.stats.mutex.RUnlock()

	// 更新内存和协程统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &ResourceStats{
		TotalResources:  rm.stats.TotalResources,
		ActiveResources: rm.stats.ActiveResources,
		ResourcesByType: copyMap(rm.stats.ResourcesByType),
		MemoryUsage:     int64(m.Alloc),
		GoroutineCount:  runtime.NumGoroutine(),
		LastCleanupTime: rm.stats.LastCleanupTime,
		CleanupCount:    rm.stats.CleanupCount,
		LeakedResources: rm.stats.LeakedResources,
	}
}

// Close 关闭资源管理器
// 关闭所有注册的资源并停止清理协程
// 返回:
//   - error: 关闭错误
func (rm *ResourceManager) Close() error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if rm.closed {
		return nil
	}

	rm.closed = true
	close(rm.cleanupCh)

	// 关闭所有资源
	var errors []error
	for id, resource := range rm.resources {
		if err := resource.Close(); err != nil {
			errors = append(errors, fmt.Errorf("关闭资源 %s 失败: %w", id, err))
		}
	}

	// 清空资源映射
	rm.resources = make(map[string]Resource)

	if len(errors) > 0 {
		return fmt.Errorf("关闭部分资源时出错: %v", errors)
	}

	Info("资源管理器已关闭")
	return nil
}

// startCleanupRoutine 启动清理协程
func (rm *ResourceManager) startCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !rm.closed {
				rm.CleanupInactiveResources()
			}
		case <-rm.cleanupCh:
			return
		}
	}
}

// updateStats 更新统计信息（需要在持有锁的情况下调用）
func (rm *ResourceManager) updateStats() {
	rm.stats.mutex.Lock()
	defer rm.stats.mutex.Unlock()

	rm.stats.TotalResources = len(rm.resources)
	rm.stats.ActiveResources = 0
	rm.stats.ResourcesByType = make(map[string]int)

	for _, resource := range rm.resources {
		resourceType := resource.GetType()
		rm.stats.ResourcesByType[resourceType]++

		if resource.IsActive() {
			rm.stats.ActiveResources++
		}
	}
}

// copyMap 复制字符串到整数的映射
func copyMap(original map[string]int) map[string]int {
	copy := make(map[string]int)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// NewManagedConnection 创建受管理的连接
// 参数:
//   - id: 连接ID
//   - connType: 连接类型
//   - closeFunc: 关闭函数
//
// 返回:
//   - *ManagedConnection: 受管理的连接
func NewManagedConnection(id, connType string, closeFunc func() error) *ManagedConnection {
	return &ManagedConnection{
		id:        id,
		connType:  connType,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		active:    true,
		closeFunc: closeFunc,
	}
}

// Close 关闭连接
func (mc *ManagedConnection) Close() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if !mc.active {
		return nil
	}

	mc.active = false
	if mc.closeFunc != nil {
		return mc.closeFunc()
	}
	return nil
}

// GetType 获取连接类型
func (mc *ManagedConnection) GetType() string {
	return mc.connType
}

// GetID 获取连接ID
func (mc *ManagedConnection) GetID() string {
	return mc.id
}

// IsActive 检查连接是否活跃
func (mc *ManagedConnection) IsActive() bool {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.active
}

// UpdateLastUsed 更新最后使用时间
func (mc *ManagedConnection) UpdateLastUsed() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.lastUsed = time.Now()
}

// GetLastUsed 获取最后使用时间
func (mc *ManagedConnection) GetLastUsed() time.Time {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.lastUsed
}

// 全局资源管理器实例
var globalResourceManager *ResourceManager
var globalResourceManagerOnce sync.Once

// GetGlobalResourceManager 获取全局资源管理器
func GetGlobalResourceManager() *ResourceManager {
	globalResourceManagerOnce.Do(func() {
		globalResourceManager = NewResourceManager()
	})
	return globalResourceManager
}

// RegisterGlobalResource 注册全局资源
func RegisterGlobalResource(resource Resource) error {
	return GetGlobalResourceManager().RegisterResource(resource)
}

// UnregisterGlobalResource 注销全局资源
func UnregisterGlobalResource(resourceID string) error {
	return GetGlobalResourceManager().UnregisterResource(resourceID)
}

// CleanupGlobalResources 清理全局资源
func CleanupGlobalResources() (int, error) {
	return GetGlobalResourceManager().CleanupInactiveResources()
}

// GetGlobalResourceStats 获取全局资源统计
func GetGlobalResourceStats() *ResourceStats {
	return GetGlobalResourceManager().GetStats()
}

// ShutdownGlobalResourceManager 关闭全局资源管理器
func ShutdownGlobalResourceManager() error {
	if globalResourceManager != nil {
		return globalResourceManager.Close()
	}
	return nil
}
