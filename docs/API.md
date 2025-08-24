# BaseSQL API 文档

本文档详细描述了 BaseSQL 的所有 API 接口和方法。

## 目录

- [客户端 API](#客户端-api)
- [数据库操作 API](#数据库操作-api)
- [稳定性组件 API](#稳定性组件-api)
- [错误处理 API](#错误处理-api)
- [配置 API](#配置-api)

## 客户端 API

### NewClient

创建新的 BaseSQL 客户端实例。

```go
func NewClient(config *Config) (*Client, error)
```

**参数:**
- `config`: 客户端配置

**返回值:**
- `*Client`: 客户端实例
- `error`: 错误信息

**示例:**
```go
config := &basesql.Config{
    AppID:     "your_app_id",
    AppSecret: "your_app_secret",
    AppToken:  "your_app_token",
    AuthType:  basesql.AuthTypeTenant,
}
client, err := basesql.NewClient(config)
```

### Close

关闭客户端，释放所有资源。

```go
func (c *Client) Close() error
```

**返回值:**
- `error`: 错误信息

### DoRequest

执行 HTTP 请求。

```go
func (c *Client) DoRequest(method, url string, body interface{}) (*http.Response, error)
```

**参数:**
- `method`: HTTP 方法
- `url`: 请求 URL
- `body`: 请求体

**返回值:**
- `*http.Response`: HTTP 响应
- `error`: 错误信息

## 数据库操作 API

### Open

打开数据库连接。

```go
func Open(client *Client, tableID string) (*gorm.DB, error)
```

**参数:**
- `client`: BaseSQL 客户端
- `tableID`: 表 ID

**返回值:**
- `*gorm.DB`: GORM 数据库实例
- `error`: 错误信息

### AutoMigrate

自动迁移表结构。

```go
func (db *DB) AutoMigrate(dst ...interface{}) error
```

**参数:**
- `dst`: 要迁移的模型

**返回值:**
- `error`: 错误信息

### Create

创建记录。

```go
func (db *DB) Create(value interface{}) *DB
```

**参数:**
- `value`: 要创建的记录

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### Find

查询记录。

```go
func (db *DB) Find(dest interface{}, conds ...interface{}) *DB
```

**参数:**
- `dest`: 查询结果目标
- `conds`: 查询条件

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### First

查询第一条记录。

```go
func (db *DB) First(dest interface{}, conds ...interface{}) *DB
```

**参数:**
- `dest`: 查询结果目标
- `conds`: 查询条件

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### Update

更新记录。

```go
func (db *DB) Update(column string, value interface{}) *DB
func (db *DB) Updates(values interface{}) *DB
```

**参数:**
- `column`: 要更新的列名
- `value`: 新值
- `values`: 要更新的值（结构体或 map）

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### Delete

删除记录。

```go
func (db *DB) Delete(value interface{}, conds ...interface{}) *DB
```

**参数:**
- `value`: 要删除的记录
- `conds`: 删除条件

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### Where

添加查询条件。

```go
func (db *DB) Where(query interface{}, args ...interface{}) *DB
```

**参数:**
- `query`: 查询条件
- `args`: 查询参数

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### Order

添加排序条件。

```go
func (db *DB) Order(value interface{}) *DB
```

**参数:**
- `value`: 排序条件

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### Limit

限制查询结果数量。

```go
func (db *DB) Limit(limit int) *DB
```

**参数:**
- `limit`: 限制数量

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

### Offset

设置查询偏移量。

```go
func (db *DB) Offset(offset int) *DB
```

**参数:**
- `offset`: 偏移量

**返回值:**
- `*DB`: 数据库实例（支持链式调用）

## 稳定性组件 API

### GetStabilityStats

获取稳定性统计信息。

```go
func (c *Client) GetStabilityStats() *StabilityStats
```

**返回值:**
- `*StabilityStats`: 稳定性统计信息

**StabilityStats 结构体:**
```go
type StabilityStats struct {
    // 熔断器统计
    CircuitBreakerState    string
    CircuitBreakerFailures int64
    
    // 连接池统计
    ConnectionPoolActive int
    ConnectionPoolIdle   int
    ConnectionPoolTotal  int
    
    // 限流器统计
    RateLimiterTokens   int64
    RateLimiterRejected int64
    
    // 请求统计
    TotalRequests       int64
    SuccessfulRequests  int64
    FailedRequests      int64
    AverageResponseTime time.Duration
}
```

### ResetStabilityComponents

重置所有稳定性组件。

```go
func (c *Client) ResetStabilityComponents()
```

### UpdateRateLimiterConfig

更新限流器配置。

```go
func (c *Client) UpdateRateLimiterConfig(config *common.RateLimiterConfig)
```

**参数:**
- `config`: 新的限流器配置

**RateLimiterConfig 结构体:**
```go
type RateLimiterConfig struct {
    Rate    float64       // 每秒允许的请求数
    Burst   int           // 突发容量
    Enabled bool          // 是否启用
}
```

### UpdateConnectionPoolConfig

更新连接池配置。

```go
func (c *Client) UpdateConnectionPoolConfig(config *common.ConnectionPoolConfig)
```

**参数:**
- `config`: 新的连接池配置

**ConnectionPoolConfig 结构体:**
```go
type ConnectionPoolConfig struct {
    MaxConnections    int           // 最大连接数
    MaxIdleConns      int           // 最大空闲连接数
    IdleConnTimeout   time.Duration // 空闲连接超时时间
    MaxConnLifetime   time.Duration // 连接最大生命周期
}
```

### HealthCheck

执行健康检查。

```go
func (c *Client) HealthCheck() error
```

**返回值:**
- `error`: 错误信息（nil 表示健康）

## 错误处理 API

### IsRetryableError

判断错误是否可重试。

```go
func IsRetryableError(err error) bool
```

**参数:**
- `err`: 错误

**返回值:**
- `bool`: 是否可重试

### IsPermissionError

判断是否为权限错误。

```go
func IsPermissionError(err error) bool
```

**参数:**
- `err`: 错误

**返回值:**
- `bool`: 是否为权限错误

### 错误类型

BaseSQL 定义了以下错误类型：

```go
var (
    ErrConnectionFailed    = errors.New("connection failed")
    ErrInvalidCredentials  = errors.New("invalid credentials")
    ErrTableNotFound      = errors.New("table not found")
    ErrFieldNotFound      = errors.New("field not found")
    ErrRecordNotFound     = errors.New("record not found")
    ErrPermissionDenied   = errors.New("permission denied")
    ErrRateLimitExceeded  = errors.New("rate limit exceeded")
    ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)
```

## 配置 API

### Config

客户端配置结构体。

```go
type Config struct {
    // 飞书应用配置
    AppID     string   // 飞书应用 ID
    AppSecret string   // 飞书应用密钥
    BaseURL   string   // API 基础 URL
    
    // 认证配置
    AuthType    AuthType // 认证类型
    AccessToken string   // 用户访问令牌
    
    // 多维表格配置
    AppToken string // 多维表格 App Token
    TableID  string // 默认表 ID
    
    // 连接配置
    Timeout         time.Duration // 请求超时时间
    MaxRetries      int           // 最大重试次数
    RetryInterval   time.Duration // 重试间隔
    RateLimitQPS    int           // 每秒请求限制
    BatchSize       int           // 批量操作大小
    CacheEnabled    bool          // 是否启用缓存
    CacheTTL        time.Duration // 缓存过期时间
    DebugMode       bool          // 调试模式
    ConsistencyMode bool          // 一致性模式
    
    // 稳定性配置
    CircuitBreakerEnabled     bool          // 是否启用熔断器
    CircuitBreakerThreshold   int           // 熔断器失败阈值
    CircuitBreakerTimeout     time.Duration // 熔断器超时时间
    ConnectionPoolSize        int           // 连接池大小
    ConnectionPoolMaxIdle     int           // 连接池最大空闲连接数
    ConnectionPoolIdleTimeout time.Duration // 连接池空闲超时时间
    RateLimiterEnabled        bool          // 是否启用限流器
    RateLimiterBurst          int           // 限流器突发容量
}
```

### AuthType

认证类型枚举。

```go
type AuthType int

const (
    AuthTypeTenant AuthType = iota // 应用认证
    AuthTypeUser                   // 用户认证
)
```

### SetRetryConfig

设置重试配置。

```go
func (c *Client) SetRetryConfig(config *common.RetryConfig)
```

**参数:**
- `config`: 重试配置

**RetryConfig 结构体:**
```go
type RetryConfig struct {
    MaxRetries    int           // 最大重试次数
    BaseDelay     time.Duration // 基础延迟时间
    MaxDelay      time.Duration // 最大延迟时间
    Multiplier    float64       // 延迟倍数
    Jitter        bool          // 是否添加随机抖动
}
```

## 使用示例

### 基本使用

```go
// 创建客户端
config := &basesql.Config{
    AppID:     "your_app_id",
    AppSecret: "your_app_secret",
    AppToken:  "your_app_token",
    AuthType:  basesql.AuthTypeTenant,
}
client, err := basesql.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 打开数据库
db, err := basesql.Open(client, "your_table_id")
if err != nil {
    log.Fatal(err)
}

// 定义模型
type User struct {
    ID   string `gorm:"primaryKey;column:record_id"`
    Name string `gorm:"column:姓名"`
    Age  int    `gorm:"column:年龄"`
}

// 自动迁移
db.AutoMigrate(&User{})

// 创建记录
user := &User{Name: "张三", Age: 25}
db.Create(user)

// 查询记录
var users []User
db.Where("age > ?", 20).Find(&users)
```

### 稳定性功能使用

```go
// 启用稳定性功能的配置
config := &basesql.Config{
    // ... 基本配置
    CircuitBreakerEnabled:    true,
    CircuitBreakerThreshold:  5,
    CircuitBreakerTimeout:    time.Second * 30,
    ConnectionPoolSize:       20,
    RateLimiterEnabled:       true,
    RateLimiterBurst:         10,
}

client, err := basesql.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// 健康检查
if err := client.HealthCheck(); err != nil {
    log.Printf("健康检查失败: %v", err)
}

// 获取统计信息
stats := client.GetStabilityStats()
log.Printf("统计信息: %+v", stats)

// 动态更新配置
newRateConfig := &common.RateLimiterConfig{
    Rate:    5,
    Burst:   10,
    Enabled: true,
}
client.UpdateRateLimiterConfig(newRateConfig)
```

## 注意事项

1. **线程安全**: 所有 API 都是线程安全的，可以在多个 goroutine 中并发使用。
2. **资源管理**: 使用完客户端后，务必调用 `Close()` 方法释放资源。
3. **错误处理**: 建议使用提供的错误判断函数来处理不同类型的错误。
4. **配置优化**: 根据实际使用场景调整稳定性配置参数。
5. **监控**: 定期检查稳定性统计信息，及时发现和解决问题。