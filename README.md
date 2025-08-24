# BaseSQL - 飞书多维表格 GORM Driver

BaseSQL 是一个将飞书多维表格（Base）封装为关系型数据库接口的 Go 组件，让您可以使用熟悉的 SQL 语法和 GORM ORM 来操作多维表格数据。

## 特性

- 🚀 **SQL 语法支持**: 使用标准 SQL 语法操作多维表格
- 🔧 **GORM 兼容**: 完全兼容 GORM ORM 框架
- 🖥️ **CLI 工具**: 提供命令行工具，支持交互式 SQL 操作
- 🔐 **多种认证**: 支持应用认证和用户认证
- 📊 **类型转换**: 自动处理 Go 类型与飞书字段类型的转换
- 🛠️ **自动迁移**: 支持数据表结构的自动创建和更新
- 🔄 **CRUD 操作**: 支持完整的增删改查操作
- 📈 **查询优化**: 支持条件查询、排序、分页等高级功能
- 🛡️ **稳定性保障**: 内置熔断器、连接池、限流器等稳定性组件
- 🔄 **智能重试**: 支持指数退避的自动重试机制
- 📊 **监控统计**: 提供详细的性能和稳定性统计信息

## 安装

```bash
go get github.com/ag9920/basesql
```

## 快速开始

### 使用 CLI 工具（推荐）

BaseSQL 提供了一个强大的命令行工具，让你可以直接使用 SQL 语法操作飞书多维表格：

```bash
# 1. 构建 CLI 工具
make build

# 2. 快速体验（使用你的飞书应用凭据）
./bin/basesql query "SHOW TABLES" --app-id=your_app_id --app-secret=your_app_secret --app-token=your_app_token
./bin/basesql query "SELECT * FROM users" --app-id=your_app_id --app-secret=your_app_secret --app-token=your_app_token
./bin/basesql shell --app-id=your_app_id --app-secret=your_app_secret --app-token=your_app_token

# 3. 生产环境使用
# 初始化配置
./bin/basesql config init

# 编辑配置文件 ~/.basesql/config.env
# 填入你的飞书应用信息

# 测试连接
./bin/basesql connect

# 执行 SQL 查询
./bin/basesql query "SELECT * FROM users"

# 执行 SQL 操作
./bin/basesql exec "INSERT INTO users (name, email) VALUES ('张三', 'zhangsan@example.com')"

# 启动交互式 SQL shell
./bin/basesql shell
```

详细的 CLI 使用说明请参考 [CLI.md](CLI.md)。

### 1. 配置飞书应用

首先需要在飞书开放平台创建应用并获取相关凭证：

1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 创建企业自建应用
3. 获取 App ID 和 App Secret
4. 创建多维表格并获取 App Token
5. 配置应用权限，确保有多维表格的读写权限

### 2. 基本使用

```go
package main

import (
    "log"
    "time"
    
    basesql "github.com/ag9920/basesql"
    "gorm.io/gorm"
)

// 定义数据模型
type User struct {
    ID        string    `gorm:"primarykey"`
    Name      string    `gorm:"size:100"`
    Email     string    `gorm:"size:100"`
    Age       int
    Active    bool      `gorm:"default:true"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func main() {
    // 配置连接
    config := &basesql.Config{
        AppID:     "your_app_id",     // 替换为你的 App ID
        AppSecret: "your_app_secret", // 替换为你的 App Secret
        AppToken:  "your_app_token",  // 替换为你的多维表格 App Token
        AuthType:  basesql.AuthTypeTenant,
        DebugMode: true,             // 开发时建议开启调试模式
    }

    // 连接数据库
    db, err := gorm.Open(basesql.Open(config), &gorm.Config{})
    if err != nil {
        log.Fatalf("failed to connect database: %v", err)
    }

    // 自动迁移
    if err := db.AutoMigrate(&User{}); err != nil {
        log.Fatalf("failed to migrate database: %v", err)
    }

    // 创建记录
    user := &User{
        Name:  "张三",
        Email: "zhangsan@example.com",
        Age:   25,
    }
    
    if err := db.Create(user).Error; err != nil {
        log.Fatalf("failed to create user: %v", err)
    }
    
    log.Printf("Created user: %+v", user)

    // 查询记录
    var users []User
    if err := db.Where("age > ?", 20).Find(&users).Error; err != nil {
        log.Fatalf("failed to query users: %v", err)
    }
    
    log.Printf("Found %d users", len(users))

    // 更新记录
    if err := db.Model(user).Update("age", 26).Error; err != nil {
        log.Fatalf("failed to update user: %v", err)
    }

    // 删除记录
    if err := db.Delete(user).Error; err != nil {
        log.Fatalf("failed to delete user: %v", err)
    }
}
```

## 认证配置

支持两种认证方式：

### 1. 应用认证 (推荐)
```go
config := &basesql.Config{
    AppID:     "your_app_id",
    AppSecret: "your_app_secret",
    AppToken:  "your_app_token",
    AuthType:  basesql.AuthTypeTenant,
}
```

### 2. 用户认证
```go
config := &basesql.Config{
    AppID:       "your_app_id",
    AppSecret:   "your_app_secret",
    AppToken:    "your_app_token",
    AuthType:    basesql.AuthTypeUser,
    AccessToken: "your_user_token", // 用户访问令牌
}
```

### 3. 环境变量配置 (推荐)
为了安全起见，建议使用环境变量来配置敏感信息。示例代码已经支持环境变量，你可以：

**方式一：设置环境变量**
```bash
export FEISHU_APP_ID=your_app_id
export FEISHU_APP_SECRET=your_app_secret
export FEISHU_APP_TOKEN=your_app_token
export DEBUG_MODE=true
```

**方式二：使用 .env 文件**
```bash
# 复制示例文件
cp example/.env.example .env
# 编辑 .env 文件，填入你的配置
```

示例代码会自动优先使用环境变量，如果没有设置则使用代码中的默认值。

## 支持的数据类型

BaseSQL 支持飞书多维表格的核心字段类型，提供完整的类型转换和 SQL 操作支持：

| 飞书字段类型 | Go 类型 | 说明 | 支持的 SQL 操作 |
|-------------|---------|------|----------------|
| 单行文本 | `string` | 简短文本内容，单行显示 | `=`, `LIKE`, `IN`, `IS NULL`, `IS NOT NULL` |
| 多行文本 | `string` | 支持换行符的长文本内容 | `=`, `LIKE`, `IN`, `IS NULL`, `IS NOT NULL` |
| 数字 | `int`, `int64`, `float64` | 整数和浮点数值 | `=`, `>`, `<`, `>=`, `<=`, `IN`, `IS NULL`, `IS NOT NULL` |
| 复选框 | `bool` | 布尔值，支持 true/false | `=`, `IS NULL`, `IS NOT NULL` |
| 日期 | `time.Time`, `string` | 日期时间类型，支持多种格式 | `=`, `>`, `<`, `>=`, `<=`, `IS NULL`, `IS NOT NULL` |
| 链接 | `string` | URL 格式的链接地址 | `=`, `LIKE`, `IN`, `IS NULL`, `IS NOT NULL` |
| 多选 | `[]string` | 多个选项组成的数组 | `IN`, `IS NULL`, `IS NOT NULL` |
| 单选 | `string` | 从预设选项中选择的单个值 | `=`, `IN`, `IS NULL`, `IS NOT NULL` |
| 人员 | `[]string` | 人员信息，支持多人选择 | `IN`, `IS NULL`, `IS NOT NULL` |

### 类型转换说明

- **自动转换**：BaseSQL 自动处理 Go 类型与飞书字段类型之间的转换
- **空值处理**：所有字段类型都支持空值检查（`IS NULL`/`IS NOT NULL`）
- **数组类型**：多选和人员字段自动处理数组与字符串的转换
- **日期格式**：支持 RFC3339、ISO8601 等标准日期格式
- **布尔值**：复选框字段支持 `true`/`false` 字符串和布尔值转换

## 支持的操作

### 表操作
- `AutoMigrate()` - 自动创建/更新表结构
- `CreateTable()` - 创建表
- `DropTable()` - 删除表
- `HasTable()` - 检查表是否存在

### 记录操作
- `Create()` - 创建记录
- `Find()` - 查询记录
- `First()` - 查询单条记录
- `Update()` - 更新记录
- `Delete()` - 删除记录

### 查询条件
- `Where()` - 条件查询，支持多种操作符
- `Order()` - 排序
- `Limit()` - 限制数量
- `Offset()` - 偏移量

### 支持的 SQL 操作符

BaseSQL 提供完整的 SQL 操作符支持，确保与飞书多维表格 API 的精确兼容：

#### 比较操作符
- `=` - 等于比较，适用于所有字段类型
- `>` - 大于比较（数字、日期字段）
- `<` - 小于比较（数字、日期字段）
- `>=` - 大于等于比较（数字、日期字段）
- `<=` - 小于等于比较（数字、日期字段）

```sql
-- 数字比较
SELECT * FROM users WHERE age > 18
SELECT * FROM products WHERE price <= 100.50

-- 日期比较
SELECT * FROM orders WHERE created_at >= '2024-01-01'
```

#### 模式匹配
- `LIKE` - 模式匹配，支持 `%` 通配符（文本字段）
- 支持前缀、后缀、包含匹配

```sql
-- 包含匹配
SELECT * FROM users WHERE name LIKE '%张%'
-- 前缀匹配
SELECT * FROM users WHERE email LIKE 'admin%'
-- 后缀匹配
SELECT * FROM files WHERE filename LIKE '%.pdf'
```

#### 集合操作
- `IN` - 值在指定集合中
- `NOT IN` - 值不在指定集合中
- 支持多选、单选、人员等字段类型

```sql
-- 单选字段
SELECT * FROM users WHERE status IN ('active', 'pending')
-- 多选字段（检查是否包含指定值）
SELECT * FROM users WHERE skills IN ('Python', 'Go')
-- 人员字段
SELECT * FROM projects WHERE assignee IN ('张三', '李四')
```

#### 空值检查
- `IS NULL` - 字段为空或未设置
- `IS NOT NULL` - 字段不为空且已设置
- 适用于所有字段类型

```sql
-- 检查必填字段
SELECT * FROM users WHERE email IS NOT NULL
-- 查找未完成的任务
SELECT * FROM tasks WHERE completed_at IS NULL
```

#### 布尔操作
- `= true` - 复选框字段为真
- `= false` - 复选框字段为假
- 支持布尔值和字符串形式

```sql
-- 查找活跃用户
SELECT * FROM users WHERE is_active = true
-- 查找未验证用户
SELECT * FROM users WHERE email_verified = false
```

#### 操作符兼容性

| 字段类型 | = | > < >= <= | LIKE | IN | IS NULL | 布尔值 |
|---------|---|-----------|------|----|---------|---------|
| 单行文本 | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ |
| 多行文本 | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ |
| 数字 | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ |
| 复选框 | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ |
| 日期 | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ |
| 链接 | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ |
| 多选 | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ |
| 单选 | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| 人员 | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ |

#### 使用示例

```go
// 文本字段操作
db.Where("name = ?", "张三")
db.Where("name LIKE ?", "%张%")
db.Where("name IN ?", []string{"张三", "李四"})
db.Where("name IS NULL")

// 数字字段操作
db.Where("age = ?", 25)
db.Where("age > ?", 18)
db.Where("salary >= ?", 5000.0)
db.Where("age IN ?", []int{25, 30, 35})

// 布尔字段操作
db.Where("active = ?", true)
db.Where("active = ?", false)
db.Where("active IS NULL")

// 日期字段操作
db.Where("created_at > ?", time.Now().AddDate(0, -1, 0))
db.Where("birth_date IS NOT NULL")

// 多选字段操作
db.Where("skills IN ?", []string{"Go", "Python"})

// 人员字段操作
db.Where("manager = ?", "张三")
db.Where("manager IN ?", []string{"张三", "李四"})
```

## 配置选项

```go
type Config struct {
    // 飞书应用配置
    AppID     string        // 飞书应用 ID
    AppSecret string        // 飞书应用密钥
    BaseURL   string        // API 基础 URL（可选，默认为官方 API）
    
    // 认证配置
    AuthType    AuthType      // 认证类型
    AccessToken string        // 用户访问令牌（用户认证时需要）
    
    // 多维表格配置
    AppToken string        // 多维表格 App Token
    TableID  string        // 默认表 ID（可选）
    
    // 连接配置
    Timeout         time.Duration // 请求超时时间（可选，默认 30 秒）
    MaxRetries      int           // 最大重试次数
    RetryInterval   time.Duration // 重试间隔
    RateLimitQPS    int           // 每秒请求限制
    BatchSize       int           // 批量操作大小
    CacheEnabled    bool          // 是否启用缓存
    CacheTTL        time.Duration // 缓存过期时间
    DebugMode       bool          // 调试模式（可选，开启后会打印详细日志）
    ConsistencyMode bool          // 一致性模式
    
    // 稳定性配置
    CircuitBreakerEnabled    bool          // 是否启用熔断器
    CircuitBreakerThreshold  int           // 熔断器失败阈值
    CircuitBreakerTimeout    time.Duration // 熔断器超时时间
    ConnectionPoolSize       int           // 连接池大小
    ConnectionPoolMaxIdle    int           // 连接池最大空闲连接数
    ConnectionPoolIdleTimeout time.Duration // 连接池空闲超时时间
    RateLimiterEnabled       bool          // 是否启用限流器
    RateLimiterBurst         int           // 限流器突发容量
}
```

## 注意事项

1. **主键字段**: 飞书多维表格的记录 ID 会自动映射为主键，建议使用 `string` 类型
2. **字段命名**: 建议使用英文字段名，避免特殊字符
3. **数据类型**: 某些复杂类型可能需要自定义转换
4. **权限配置**: 确保应用有足够的权限访问多维表格
5. **API 限制**: 注意飞书 API 的调用频率限制
6. **表名映射**: GORM 会自动将结构体名转换为表名（如 `User` -> `users`）
7. **字段映射**: 使用 `gorm` 标签来控制字段映射和属性

## 稳定性功能

BaseSQL 内置了多种稳定性保障机制，确保在高并发和网络不稳定环境下的可靠性：

### 熔断器 (Circuit Breaker)

熔断器可以防止级联故障，当检测到大量失败请求时自动切断请求：

```go
// 获取熔断器统计信息
stats := client.GetStabilityStats()
log.Printf("熔断器状态: %s, 失败次数: %d", stats.CircuitBreakerState, stats.CircuitBreakerFailures)

// 重置熔断器
client.ResetStabilityComponents()
```

### 连接池 (Connection Pool)

连接池管理 HTTP 连接，提高性能并控制资源使用：

```go
// 更新连接池配置
newPoolConfig := &common.ConnectionPoolConfig{
    MaxConnections:    50,
    MaxIdleConns:      10,
    IdleConnTimeout:   time.Minute * 5,
    MaxConnLifetime:   time.Minute * 30,
}
client.UpdateConnectionPoolConfig(newPoolConfig)
```

### 限流器 (Rate Limiter)

基于令牌桶算法的限流器，防止请求过载：

```go
// 更新限流器配置
newRateConfig := &common.RateLimiterConfig{
    Rate:       10,    // 每秒 10 个请求
    Burst:      20,    // 突发容量 20
    Enabled:    true,
}
client.UpdateRateLimiterConfig(newRateConfig)
```

### 健康检查

定期检查客户端和各组件的健康状态：

```go
// 执行健康检查
if err := client.HealthCheck(); err != nil {
    log.Printf("健康检查失败: %v", err)
} else {
    log.Println("系统运行正常")
}
```

### 监控统计

获取详细的性能和稳定性统计信息：

```go
stats := client.GetStabilityStats()
log.Printf("统计信息: %+v", stats)
// 输出包括：
// - 熔断器状态和统计
// - 连接池使用情况
// - 限流器统计
// - 请求成功/失败次数
// - 平均响应时间等
```

## 错误处理

BaseSQL 提供了丰富的错误处理机制：

```go
if err := db.Create(&user).Error; err != nil {
    if basesql.IsPermissionError(err) {
        log.Println("权限不足，请检查应用权限配置")
        // 可能需要重新配置应用权限或检查 App Token
    } else if basesql.IsRetryableError(err) {
        log.Println("网络错误，可以重试")
        // 可以实现重试逻辑
        time.Sleep(time.Second)
        // 重试操作...
    } else {
        log.Printf("其他错误: %v", err)
        // 处理其他类型的错误
    }
}
```

### 常见错误类型

- `ErrConnectionFailed`: 连接失败
- `ErrInvalidCredentials`: 认证信息无效
- `ErrTableNotFound`: 表不存在
- `ErrFieldNotFound`: 字段不存在
- `ErrRecordNotFound`: 记录不存在
- `ErrPermissionDenied`: 权限不足
- `ErrRateLimitExceeded`: 请求频率超限

## 常见问题

### Q: 如何获取多维表格的 App Token？
A: 在飞书多维表格中，点击右上角的"..."菜单，选择"高级设置"，在"应用 Token"部分可以找到。

### Q: 为什么提示权限不足？
A: 请确保：
1. 应用已获得多维表格的读写权限
2. App Token 对应的多维表格允许该应用访问
3. 如果使用用户认证，确保用户有相应权限

### Q: 支持事务吗？
A: 由于飞书多维表格 API 的限制，目前不支持传统意义上的事务。

### Q: 如何处理大量数据？
A: 建议使用分页查询，并注意 API 调用频率限制。

### Q: 如何优化高并发场景下的性能？
A: BaseSQL 提供了多种稳定性机制：
1. 启用连接池来复用 HTTP 连接
2. 使用限流器控制请求频率
3. 配置熔断器防止级联故障
4. 调整重试策略和超时时间

### Q: 熔断器什么时候会触发？
A: 当连续失败次数达到配置的阈值时，熔断器会进入开启状态，暂时拒绝所有请求。经过一段时间后会进入半开状态进行探测。

### Q: 如何监控系统运行状态？
A: 使用 `client.GetStabilityStats()` 获取详细统计信息，包括熔断器状态、连接池使用情况、限流器统计等。建议定期调用 `client.HealthCheck()` 进行健康检查。

## 示例项目

查看 `example/` 目录中的完整示例代码，支持两种配置方式：

**基本示例：**
```bash
cd example

# 方式一：直接修改代码中的配置
# 编辑 main.go，替换为你的真实凭据
go run main.go

# 方式二：使用环境变量（推荐）
cp .env.example .env
# 编辑 .env 文件，填入你的配置
source .env  # 或者 export 各个变量
go run main.go
```

**稳定性功能示例：**
```bash
cd example

# 运行稳定性功能演示
# 编辑 stability_example.go，填入你的配置
go run stability_example.go
```

## 文档

- [API 文档](docs/API.md) - 详细的 API 接口说明
- [稳定性功能指南](example/stability_example.go) - 稳定性功能使用示例
- [配置指南](#配置选项) - 完整的配置选项说明

## 许可证

MIT License