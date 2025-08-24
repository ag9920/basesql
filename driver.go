package basesql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/ag9920/basesql/internal/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// Dialector 实现了 GORM 的方言器接口，用于将 GORM 操作转换为飞书多维表格 API 调用
// 它包含了配置信息和客户端实例，是整个驱动的核心组件
type Dialector struct {
	*Config         // 配置信息，包含认证、超时、重试等设置
	Client  *Client // 飞书 API 客户端实例
}

// Open 创建并返回一个新的 BaseSQL 方言器实例
// 该函数会合并用户配置和默认配置，确保所有必要的配置项都有合理的默认值
// 参数:
//   - config: 用户提供的配置，可以为 nil 或部分配置
//
// 返回:
//   - gorm.Dialector: GORM 方言器接口实例
func Open(config *Config) gorm.Dialector {
	// 如果配置为空，使用默认配置
	if config == nil {
		config = DefaultConfig()
	} else {
		// 合并默认配置，使用更高效的配置合并方式
		config = mergeWithDefaults(config)
	}

	return &Dialector{Config: config}
}

// mergeWithDefaults 将用户配置与默认配置合并
// 只有当用户配置中的字段为零值时，才使用默认值
// 参数:
//   - userConfig: 用户提供的配置
//
// 返回:
//   - *Config: 合并后的配置
func mergeWithDefaults(userConfig *Config) *Config {
	defaultConfig := DefaultConfig()

	// 创建新的配置实例，避免修改原始配置
	mergedConfig := *userConfig

	// 只覆盖零值字段
	if mergedConfig.BaseURL == "" {
		mergedConfig.BaseURL = defaultConfig.BaseURL
	}
	if mergedConfig.AuthType == "" {
		mergedConfig.AuthType = defaultConfig.AuthType
	}
	if mergedConfig.Timeout == 0 {
		mergedConfig.Timeout = defaultConfig.Timeout
	}
	if mergedConfig.MaxRetries == 0 {
		mergedConfig.MaxRetries = defaultConfig.MaxRetries
	}
	if mergedConfig.RetryInterval == 0 {
		mergedConfig.RetryInterval = defaultConfig.RetryInterval
	}
	if mergedConfig.RateLimitQPS == 0 {
		mergedConfig.RateLimitQPS = defaultConfig.RateLimitQPS
	}
	if mergedConfig.BatchSize == 0 {
		mergedConfig.BatchSize = defaultConfig.BatchSize
	}
	if mergedConfig.CacheTTL == 0 {
		mergedConfig.CacheTTL = defaultConfig.CacheTTL
	}

	return &mergedConfig
}

// Name 返回方言器的名称
// 实现 gorm.Dialector 接口的 Name 方法
// 返回:
//   - string: 方言器名称 "basesql"
func (d *Dialector) Name() string {
	return "basesql"
}

// Initialize 初始化方言器
// 该方法会创建飞书 API 客户端、设置连接池并注册回调函数
// 参数:
//   - db: GORM 数据库实例
//
// 返回:
//   - error: 初始化过程中的错误
func (d *Dialector) Initialize(db *gorm.DB) error {
	// 验证输入参数
	if db == nil {
		return fmt.Errorf("GORM 数据库实例不能为 nil")
	}
	if d.Config == nil {
		return fmt.Errorf("配置信息不能为 nil")
	}

	// 初始化飞书 API 客户端
	client, err := NewClient(d.Config)
	if err != nil {
		return fmt.Errorf("初始化飞书 API 客户端失败: %w", err)
	}
	d.Client = client

	// 设置自定义连接池，用于拦截 SQL 操作
	db.ConnPool = &ConnPool{Dialector: d}

	// 注册回调函数，将 GORM 操作转换为飞书 API 调用
	RegisterCallbacks(db, d)

	return nil
}

// Migrator 返回数据库迁移器实例
// 实现 gorm.Dialector 接口的 Migrator 方法
// 参数:
//   - db: GORM 数据库实例
//
// 返回:
//   - gorm.Migrator: 迁移器实例
func (d *Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	if db == nil {
		// 返回一个空的迁移器，避免 panic
		return Migrator{d, migrator.Migrator{}}
	}
	return Migrator{d, migrator.Migrator{Config: migrator.Config{DB: db}}}
}

// DataTypeOf 将 GORM 字段类型转换为飞书多维表格字段类型
// 实现 gorm.Dialector 接口的 DataTypeOf 方法
// 参数:
//   - field: GORM 字段定义
//
// 返回:
//   - string: 对应的飞书多维表格字段类型
func (d *Dialector) DataTypeOf(field *schema.Field) string {
	if field == nil {
		return "text" // 默认返回文本类型
	}

	switch field.DataType {
	case schema.Bool:
		return "checkbox" // 布尔值映射为复选框
	case schema.Int, schema.Uint:
		return "number" // 整数映射为数字
	case schema.Float:
		return "number" // 浮点数映射为数字
	case schema.String:
		return "text" // 字符串映射为文本
	case schema.Time:
		return "datetime" // 时间映射为日期时间
	case schema.Bytes:
		return "text" // 字节数组映射为文本
	default:
		return "text" // 未知类型默认为文本
	}
}

// DefaultValueOf 返回字段的默认值表达式
// 实现 gorm.Dialector 接口的 DefaultValueOf 方法
// 参数:
//   - field: GORM 字段定义
//
// 返回:
//   - clause.Expression: 默认值表达式
func (d *Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	if field == nil || field.DefaultValue == "" {
		return clause.Expr{SQL: "NULL"}
	}
	return clause.Expr{SQL: field.DefaultValue}
}

// BindVarTo 写入绑定变量占位符
// 实现 gorm.Dialector 接口的 BindVarTo 方法
// 参数:
//   - writer: 子句写入器
//   - stmt: GORM 语句
//   - v: 绑定的值
func (d *Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	if writer == nil {
		return
	}
	writer.WriteByte('?')
}

// QuoteTo 写入带引号的标识符
// 实现 gorm.Dialector 接口的 QuoteTo 方法
// 参数:
//   - writer: 子句写入器
//   - str: 要引用的字符串
func (d *Dialector) QuoteTo(writer clause.Writer, str string) {
	if writer == nil || str == "" {
		return
	}
	writer.WriteByte('`')
	writer.WriteString(str)
	writer.WriteByte('`')
}

// Explain 解释 SQL 语句，用于调试和日志记录
// 实现 gorm.Dialector 接口的 Explain 方法
// 参数:
//   - sql: SQL 语句
//   - vars: 绑定变量
//
// 返回:
//   - string: 解释后的 SQL 语句
func (d *Dialector) Explain(sql string, vars ...interface{}) string {
	if sql == "" {
		return "[BaseSQL] 空 SQL 语句"
	}
	return logger.ExplainSQL(sql, nil, `'`, vars...)
}

// ConnPool 实现 gorm.ConnPool 接口
// 用于拦截 GORM 的数据库操作，将其转换为飞书多维表格 API 调用
// 注意：由于飞书多维表格不是传统的 SQL 数据库，大部分方法返回不支持的错误
type ConnPool struct {
	Dialector *Dialector // 关联的方言器实例
}

// PrepareContext 准备 SQL 语句
// 由于飞书多维表格不支持预编译语句，此方法返回不支持错误
// 参数:
//   - ctx: 上下文
//   - query: SQL 查询语句
//
// 返回:
//   - *sql.Stmt: 预编译语句（始终为 nil）
//   - error: 不支持错误
func (c *ConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, fmt.Errorf("BaseSQL: 飞书多维表格不支持预编译语句")
}

// ExecContext 执行 SQL 语句
// 由于飞书多维表格不支持直接 SQL 执行，此方法返回不支持错误
// 实际的操作通过 GORM 回调函数处理
// 参数:
//   - ctx: 上下文
//   - query: SQL 查询语句
//   - args: 查询参数
//
// 返回:
//   - sql.Result: 执行结果（始终为 nil）
//   - error: 不支持错误
func (c *ConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("BaseSQL: 飞书多维表格不支持直接 SQL 执行，请使用 GORM 方法")
}

// QueryContext 执行查询语句
// 由于飞书多维表格不支持直接 SQL 查询，此方法返回不支持错误
// 实际的查询通过 GORM 回调函数处理
// 参数:
//   - ctx: 上下文
//   - query: SQL 查询语句
//   - args: 查询参数
//
// 返回:
//   - *sql.Rows: 查询结果（始终为 nil）
//   - error: 不支持错误
func (c *ConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("BaseSQL: 飞书多维表格不支持直接 SQL 查询，请使用 GORM 方法")
}

// QueryRowContext 执行单行查询
// 由于飞书多维表格不支持直接 SQL 查询，此方法返回空行
// 实际的查询通过 GORM 回调函数处理
// 参数:
//   - ctx: 上下文
//   - query: SQL 查询语句
//   - args: 查询参数
//
// 返回:
//   - *sql.Row: 查询结果行（空行）
func (c *ConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// 返回一个空行，避免 nil 指针异常
	return &sql.Row{}
}

// Transaction 实现 sql.driver.Tx 接口的事务结构体
// 用于兼容 GORM 的事务处理机制
type Transaction struct {
	connPool *ConnPool // 关联的连接池
}

// Commit 提交事务
// 实现 sql.driver.Tx 接口
func (tx *Transaction) Commit() error {
	// 飞书多维表格不支持事务，所有操作都是立即提交的
	common.Debugf("Transaction.Commit: 飞书多维表格不支持事务，操作已自动提交")
	return nil
}

// Rollback 回滚事务
// 实现 sql.driver.Tx 接口
func (tx *Transaction) Rollback() error {
	// 飞书多维表格不支持事务回滚，空实现
	common.Warnf("Transaction.Rollback: 飞书多维表格不支持事务回滚，已执行的操作无法撤销")
	return nil
}

// PrepareContext 准备语句
// 实现 gorm.ConnPool 接口
func (tx *Transaction) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return tx.connPool.PrepareContext(ctx, query)
}

// ExecContext 执行语句
// 实现 gorm.ConnPool 接口
func (tx *Transaction) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return tx.connPool.ExecContext(ctx, query, args...)
}

// QueryContext 执行查询
// 实现 gorm.ConnPool 接口
func (tx *Transaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return tx.connPool.QueryContext(ctx, query, args...)
}

// QueryRowContext 执行单行查询
// 实现 gorm.ConnPool 接口
func (tx *Transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return tx.connPool.QueryRowContext(ctx, query, args...)
}

// BeginTx 开始事务
// 实现 gorm.ConnPoolBeginner 接口
// 由于飞书多维表格不支持事务，此方法返回一个Transaction实例
// 参数:
//   - ctx: 上下文
//   - opt: 事务选项
//
// 返回:
//   - gorm.ConnPool: Transaction实例（实现了gorm.ConnPool接口）
//   - error: 错误信息（始终为 nil）
func (c *ConnPool) BeginTx(ctx context.Context, opt *sql.TxOptions) (gorm.ConnPool, error) {
	// 飞书多维表格不支持事务，但为了兼容 GORM，返回Transaction实例
	// 注意：这意味着所有操作都是立即提交的，无法回滚
	common.Debugf("BeginTx: 飞书多维表格不支持真正的事务，操作将立即提交")
	return &Transaction{connPool: c}, nil
}

// SavePoint 创建保存点
// 实现 gorm.SavePointer 接口
// 由于飞书多维表格不支持事务保存点，此方法为空实现
// 参数:
//   - ctx: 上下文
//   - name: 保存点名称
//
// 返回:
//   - error: 错误信息（始终为 nil）
func (c *ConnPool) SavePoint(ctx context.Context, name string) error {
	// 飞书多维表格不支持事务保存点，空实现
	common.Debugf("SavePoint: 飞书多维表格不支持保存点，保存点 '%s' 被忽略", name)
	return nil
}

// RollbackTo 回滚到保存点
// 实现 gorm.SavePointer 接口
// 由于飞书多维表格不支持事务回滚，此方法为空实现
// 参数:
//   - ctx: 上下文
//   - name: 保存点名称
//
// 返回:
//   - error: 错误信息（始终为 nil）
func (c *ConnPool) RollbackTo(ctx context.Context, name string) error {
	// 飞书多维表格不支持事务回滚，空实现
	common.Warnf("RollbackTo: 飞书多维表格不支持事务回滚，回滚到保存点 '%s' 被忽略，数据可能已被修改", name)
	return nil
}

// BaseValue 实现了 GORM 和 SQL 相关的值接口
// 用于在 GORM 和飞书多维表格之间进行值的转换和传递
type BaseValue struct {
	Val interface{} // 存储的值，可以是任意类型
}

// NewBaseValue 创建一个新的 BaseValue 实例
// 参数:
//   - val: 要存储的值
//
// 返回:
//   - *BaseValue: BaseValue 实例
func NewBaseValue(val interface{}) *BaseValue {
	return &BaseValue{Val: val}
}

// GormValue 实现 gorm.Valuer 接口
// 将值转换为 GORM 可以处理的表达式
// 参数:
//   - ctx: 上下文
//   - db: GORM 数据库实例
//
// 返回:
//   - clause.Expr: GORM 表达式
func (v BaseValue) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	if v.Val == nil {
		return clause.Expr{SQL: "NULL"}
	}
	return clause.Expr{SQL: "?", Vars: []interface{}{v.Val}}
}

// Scan 实现 sql.Scanner 接口
// 从数据库扫描值到结构体
// 参数:
//   - value: 从数据库读取的值
//
// 返回:
//   - error: 扫描过程中的错误
func (v *BaseValue) Scan(value interface{}) error {
	if v == nil {
		return fmt.Errorf("BaseValue: 接收器不能为 nil")
	}
	v.Val = value
	return nil
}

// Value 实现 driver.Valuer 接口
// 将值转换为数据库驱动可以处理的类型
// 返回:
//   - driver.Value: 驱动值
//   - error: 转换过程中的错误
func (v BaseValue) Value() (driver.Value, error) {
	// 检查值是否为 nil
	if v.Val == nil {
		return nil, nil
	}

	// 检查值是否已经是有效的驱动值类型
	switch v.Val.(type) {
	case nil, int64, float64, bool, []byte, string, time.Time:
		return v.Val, nil
	default:
		// 对于其他类型，尝试转换为字符串
		return fmt.Sprintf("%v", v.Val), nil
	}
}

// IsNil 检查值是否为 nil
// 返回:
//   - bool: 如果值为 nil 返回 true，否则返回 false
func (v BaseValue) IsNil() bool {
	return v.Val == nil
}

// String 返回值的字符串表示
// 返回:
//   - string: 值的字符串表示
func (v BaseValue) String() string {
	if v.Val == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", v.Val)
}
